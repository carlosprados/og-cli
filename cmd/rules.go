package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/carlosprados/og-cli/internal/output"
	"github.com/carlosprados/og-cli/internal/unwrap"
	"github.com/carlosprados/og-cli/pkg/opengate"
	"github.com/spf13/cobra"
)

const defaultChannel = "default_channel"

var rulesCmd = &cobra.Command{
	Use:   "rules",
	Short: "Manage OpenGate automation rules",
	Long: `Manage OpenGate automation rules (channel-scoped).

Rules come in two modes:
  EASY     — declarative condition + actions (open/close alarms, email, HTTP, operations)
  ADVANCED — a JavaScript function evaluates the trigger and decides the actions

The pull/wrap/deploy verbs mirror the workspace lifecycle: pull a rule into an
editable directory (rule.json + javascript.js for ADVANCED rules), edit the JS
locally with your IDE, and deploy it back.

Examples:
  og rules search -w "rule.active eq true"
  og rules search -w "rule.mode eq ADVANCED"
  og rules get <rule-id>
  og rules pull <rule-id> --dir rules/
  og rules deploy rules/<rule-slug> --update
  og rules disable <rule-id>`,
}

var (
	rulesChannel      string
	rulesSearchWhere  []string
	rulesSearchLimit  int
	rulesSearchFilter string
	ruleCreateFile    string
	ruleUpdateFile    string
	rulePullDir       string
	rulePullForce     bool
	ruleWrapOut       string
	ruleDeployUpdate  bool
)

// --- search ---

var rulesSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search rules",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := activeProfile()
		if err != nil {
			return err
		}
		c := opengate.New(p.Host, p.Token)

		filter, err := buildSearchFilter(rulesSearchWhere, rulesSearchLimit, nil, rulesSearchFilter)
		if err != nil {
			return err
		}

		resp, err := c.SearchRules(filter)
		if err != nil {
			return err
		}

		return output.Print(outFmt, resp.Rules,
			[]string{"Identifier", "Name", "Mode", "Active", "Trigger"},
			func(data any) [][]string {
				rules := data.([]json.RawMessage)
				rows := make([][]string, len(rules))
				for i, raw := range rules {
					s := opengate.ParseRuleSummary(raw)
					rows[i] = []string{s.Identifier, s.Name, s.Mode, fmt.Sprintf("%v", s.Active), s.RuleTriggerName()}
				}
				return rows
			},
		)
	},
}

// --- get ---

var rulesGetCmd = &cobra.Command{
	Use:   "get <rule-id>",
	Short: "Get a rule by identifier",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, orgName, err := rulesClient()
		if err != nil {
			return err
		}
		data, err := c.GetRule(orgName, rulesChannel, args[0])
		if err != nil {
			return err
		}
		return printJSON(data)
	},
}

// --- create / update / delete ---

var rulesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a rule from a JSON file",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, orgName, err := rulesClient()
		if err != nil {
			return err
		}
		body, err := os.ReadFile(ruleCreateFile)
		if err != nil {
			return err
		}
		if _, err := c.CreateRule(orgName, rulesChannel, body); err != nil {
			return err
		}
		fmt.Println("Rule created successfully.")
		return nil
	},
}

var rulesUpdateCmd = &cobra.Command{
	Use:   "update <rule-id>",
	Short: "Update a rule from a JSON file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, orgName, err := rulesClient()
		if err != nil {
			return err
		}
		body, err := os.ReadFile(ruleUpdateFile)
		if err != nil {
			return err
		}
		if err := c.UpdateRule(orgName, rulesChannel, args[0], body); err != nil {
			return err
		}
		fmt.Println("Rule updated successfully.")
		return nil
	},
}

var rulesDeleteCmd = &cobra.Command{
	Use:   "delete <rule-id>",
	Short: "Delete a rule",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, orgName, err := rulesClient()
		if err != nil {
			return err
		}
		if err := c.DeleteRule(orgName, rulesChannel, args[0]); err != nil {
			return err
		}
		fmt.Println("Rule deleted successfully.")
		return nil
	},
}

// --- enable / disable ---

var rulesEnableCmd = &cobra.Command{
	Use:   "enable <rule-id>",
	Short: "Enable a rule (sets active=true)",
	Args:  cobra.ExactArgs(1),
	RunE:  func(cmd *cobra.Command, args []string) error { return setRuleActive(args[0], true) },
}

var rulesDisableCmd = &cobra.Command{
	Use:   "disable <rule-id>",
	Short: "Disable a rule (sets active=false)",
	Args:  cobra.ExactArgs(1),
	RunE:  func(cmd *cobra.Command, args []string) error { return setRuleActive(args[0], false) },
}

func setRuleActive(id string, active bool) error {
	c, orgName, err := rulesClient()
	if err != nil {
		return err
	}
	if err := c.SetRuleActive(orgName, rulesChannel, id, active); err != nil {
		return err
	}
	state := "disabled"
	if active {
		state = "enabled"
	}
	fmt.Printf("Rule %s %s.\n", id, state)
	return nil
}

// --- catalog ---

var rulesCatalogCmd = &cobra.Command{
	Use:   "catalog",
	Short: "Show the platform rules catalog (predefined templates)",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := activeProfile()
		if err != nil {
			return err
		}
		c := opengate.New(p.Host, p.Token)
		data, err := c.RulesCatalog()
		if err != nil {
			return err
		}
		return printJSON(data)
	},
}

// --- pull / pull-all / wrap / deploy ---

var rulesPullCmd = &cobra.Command{
	Use:     "pull <rule-id>",
	Aliases: []string{"unwrap"},
	Short:   "Pull a rule into an editable directory (rule.json + .js files)",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, orgName, err := rulesClient()
		if err != nil {
			return err
		}
		raw, err := c.GetRule(orgName, rulesChannel, args[0])
		if err != nil {
			return err
		}
		dir, err := unwrapRuleTo(raw, rulePullDir)
		if err != nil {
			return err
		}
		fmt.Printf("Rule unwrapped to %s\n", dir)
		return nil
	},
}

var rulesPullAllCmd = &cobra.Command{
	Use:     "pull-all",
	Aliases: []string{"unwrap-all"},
	Short:   "Pull every rule of the organization into <dir>/<rule-slug>/",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := activeProfile()
		if err != nil {
			return err
		}
		c := opengate.New(p.Host, p.Token)

		resp, err := c.SearchRules(nil)
		if err != nil {
			return err
		}
		if len(resp.Rules) == 0 {
			fmt.Println("No rules found.")
			return nil
		}
		count := 0
		for _, raw := range resp.Rules {
			dir, err := unwrapRuleTo(raw, rulePullDir)
			if err != nil {
				return err
			}
			fmt.Printf("  %s\n", dir)
			count++
		}
		fmt.Printf("%d rules unwrapped to %s\n", count, rulePullDir)
		return nil
	},
}

var rulesWrapCmd = &cobra.Command{
	Use:   "wrap <rule-dir>",
	Short: "Rebuild a rule JSON from an unwrapped directory (no upload)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := unwrap.WrapRule(args[0])
		if err != nil {
			return err
		}
		if ruleWrapOut != "" {
			if err := os.WriteFile(ruleWrapOut, data, 0o644); err != nil {
				return err
			}
			fmt.Printf("Rule written to %s\n", ruleWrapOut)
			return nil
		}
		fmt.Println(string(data))
		return nil
	},
}

var rulesDeployCmd = &cobra.Command{
	Use:   "deploy <rule-dir>",
	Short: "Wrap + upload a rule in one step (POST, or PUT with --update)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, orgName, err := rulesClient()
		if err != nil {
			return err
		}
		body, err := unwrap.WrapRule(args[0])
		if err != nil {
			return err
		}

		if ruleDeployUpdate {
			var rule struct {
				Identifier string `json:"identifier"`
			}
			if err := json.Unmarshal(body, &rule); err != nil || rule.Identifier == "" {
				return fmt.Errorf("--update requires an 'identifier' field in rule.json (pull the rule first)")
			}
			if err := c.UpdateRule(orgName, rulesChannel, rule.Identifier, body); err != nil {
				return err
			}
			fmt.Printf("Rule %s updated successfully.\n", rule.Identifier)
			return nil
		}

		if _, err := c.CreateRule(orgName, rulesChannel, body); err != nil {
			return err
		}
		fmt.Println("Rule created successfully.")
		return nil
	},
}

// --- logs ---

var rulesLogsCmd = &cobra.Command{
	Use:   "logs <rule-id>",
	Short: "Stream a rule's execution logs (live, colourised by severity)",
	Long: `Stream the live execution logs of an automation rule over WebSocket.

Traces are emitted by logger.trace/debug/info/warn/error calls inside the rule's
JavaScript, colourised by severity. Press Ctrl-C to stop.

NOTE: a device only emits logs while its administrativeState is TESTING. With an
ACTIVE device the rule still runs but no logs are streamed. Use --level DEBUG/TRACE
to see logger.debug/trace (default is INFO).

Examples:
  og rules logs <rule-id> --org sensehat
  og rules logs <rule-id> --level TRACE --org sensehat`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return streamFunctionLogs(opengate.LoggerRules, rulesChannel, args[0])
	},
}

// --- helpers ---

func rulesClient() (*opengate.Client, string, error) {
	p, err := activeProfile()
	if err != nil {
		return nil, "", err
	}
	orgName, err := resolveOrg(p)
	if err != nil {
		return nil, "", err
	}
	return opengate.New(p.Host, p.Token), orgName, nil
}

func unwrapRuleTo(raw json.RawMessage, dir string) (string, error) {
	s := opengate.ParseRuleSummary(raw)

	slug := unwrap.DedupedSlug(s.Name, s.Identifier, map[string]bool{})
	target := filepath.Join(dir, slug)
	if _, err := os.Stat(target); err == nil && !rulePullForce {
		return "", fmt.Errorf("destination %s already exists (use --force to overwrite)", target)
	}

	ruleDir, err := unwrap.UnwrapRule(raw, dir)
	if err != nil {
		return "", fmt.Errorf("unwrapping rule %s: %w", s.Name, err)
	}
	return ruleDir, nil
}

func printJSON(data json.RawMessage) error {
	pretty, err := json.MarshalIndent(json.RawMessage(data), "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(pretty))
	return nil
}

// --- init ---

func init() {
	rulesCmd.PersistentFlags().StringVar(&rulesChannel, "channel", defaultChannel, "channel the rule belongs to")

	rulesSearchCmd.Flags().StringArrayVarP(&rulesSearchWhere, "where", "w", nil, `filter condition: "field op value" (repeatable)`)
	rulesSearchCmd.Flags().IntVar(&rulesSearchLimit, "limit", 0, "max number of results")
	rulesSearchCmd.Flags().StringVar(&rulesSearchFilter, "filter", "", "raw search filter as JSON (overrides -w)")

	rulesCreateCmd.Flags().StringVarP(&ruleCreateFile, "file", "f", "", "path to JSON file with rule definition")
	rulesCreateCmd.MarkFlagRequired("file")

	rulesUpdateCmd.Flags().StringVarP(&ruleUpdateFile, "file", "f", "", "path to JSON file with rule definition")
	rulesUpdateCmd.MarkFlagRequired("file")

	rulesPullCmd.Flags().StringVar(&rulePullDir, "dir", "rules", "destination directory")
	rulesPullCmd.Flags().BoolVar(&rulePullForce, "force", false, "overwrite existing destination")
	rulesPullAllCmd.Flags().StringVar(&rulePullDir, "dir", "rules", "destination directory")
	rulesPullAllCmd.Flags().BoolVar(&rulePullForce, "force", false, "overwrite existing destinations")

	rulesWrapCmd.Flags().StringVar(&ruleWrapOut, "out", "", "write rule JSON to this file (default: stdout)")

	rulesDeployCmd.Flags().BoolVar(&ruleDeployUpdate, "update", false, "update an existing rule (PUT) instead of creating (POST)")

	rulesLogsCmd.Flags().StringVar(&logsLevel, "level", "INFO", "log level: ERROR | WARN | INFO | DEBUG | TRACE")

	rulesCmd.AddCommand(rulesSearchCmd)
	rulesCmd.AddCommand(rulesGetCmd)
	rulesCmd.AddCommand(rulesCreateCmd)
	rulesCmd.AddCommand(rulesUpdateCmd)
	rulesCmd.AddCommand(rulesDeleteCmd)
	rulesCmd.AddCommand(rulesEnableCmd)
	rulesCmd.AddCommand(rulesDisableCmd)
	rulesCmd.AddCommand(rulesCatalogCmd)
	rulesCmd.AddCommand(rulesPullCmd)
	rulesCmd.AddCommand(rulesPullAllCmd)
	rulesCmd.AddCommand(rulesWrapCmd)
	rulesCmd.AddCommand(rulesDeployCmd)
	rulesCmd.AddCommand(rulesLogsCmd)

	rootCmd.AddCommand(rulesCmd)
}
