package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/carlosprados/og-cli/v2/internal/config"
	"github.com/carlosprados/og-cli/v2/internal/output"
	"github.com/carlosprados/og-cli/v2/internal/typegen"
	"github.com/carlosprados/og-cli/v2/internal/unwrap"
	"github.com/carlosprados/og-cli/v2/pkg/opengate"
	"github.com/spf13/cobra"
)

var connectorsCmd = &cobra.Command{
	Use:     "connectors",
	Aliases: []string{"cf"},
	Short:   "Manage OpenGate connector functions",
	Long: `Manage OpenGate connector functions (channel-scoped).

Connector functions are JavaScript hooks in the device-integration pipeline:
  REQUEST    — transform an outgoing operation request before it reaches the device
  RESPONSE   — process an operation response coming back from the device
  COLLECTION — process collected data and emit datapoints (collection.addDatapoint/send)

REQUEST functions match by operationName + northCriterias; RESPONSE/COLLECTION
match by southCriterias (URIs, topics, OIDs). The code lives in the 'javascript'
field. operationalStatus is one of DISABLED | PRODUCTION | TEST.

The pull/wrap/deploy verbs mirror the rules and workspace lifecycle: pull a
connector function into an editable directory (connectorfunction.json +
javascript.js), edit the JS locally with your IDE, and deploy it back.

Examples:
  og connectors list --org sensehat
  og connectors get <cf-id> --org sensehat
  og connectors pull <cf-id> --dir connectors/
  og connectors deploy connectors/<cf-slug> --update
  og connectors disable <cf-id> --org sensehat`,
}

var (
	connectorsChannel string
	cfCreateFile      string
	cfUpdateFile      string
	cfPullDir         string
	cfPullForce       bool
	cfWrapOut         string
	cfDeployUpdate    bool
	cfNoTypings       bool
)

// --- list ---

var connectorsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List connector functions in a channel",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, orgName, err := connectorsClient()
		if err != nil {
			return err
		}
		resp, err := c.ListConnectorFunctions(cmd.Context(), orgName, connectorsChannel)
		if err != nil {
			return err
		}

		return output.Print(outFmt, resp.ConnectorFunctions,
			[]string{"Identifier", "Name", "Type", "Status", "Operation"},
			func(data any) [][]string {
				cfs := data.([]json.RawMessage)
				rows := make([][]string, len(cfs))
				for i, raw := range cfs {
					s := opengate.ParseConnectorFunctionSummary(raw)
					rows[i] = []string{s.Identifier, s.DisplayName(), s.Type, s.OperationalStatus, s.OperationName}
				}
				return rows
			},
		)
	},
}

// --- get ---

var connectorsGetCmd = &cobra.Command{
	Use:   "get <cf-id>",
	Short: "Get a connector function by identifier",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, orgName, err := connectorsClient()
		if err != nil {
			return err
		}
		data, err := c.GetConnectorFunction(cmd.Context(), orgName, connectorsChannel, args[0])
		if err != nil {
			return err
		}
		return printJSON(data)
	},
}

// --- create / update / delete ---

var connectorsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a connector function from a JSON file",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, orgName, err := connectorsClient()
		if err != nil {
			return err
		}
		body, err := os.ReadFile(cfCreateFile)
		if err != nil {
			return err
		}
		if _, err := c.CreateConnectorFunction(cmd.Context(), orgName, connectorsChannel, body); err != nil {
			return err
		}
		fmt.Println("Connector function created successfully.")
		return nil
	},
}

var connectorsUpdateCmd = &cobra.Command{
	Use:   "update <cf-id>",
	Short: "Update a connector function from a JSON file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, orgName, err := connectorsClient()
		if err != nil {
			return err
		}
		body, err := os.ReadFile(cfUpdateFile)
		if err != nil {
			return err
		}
		if err := c.UpdateConnectorFunction(cmd.Context(), orgName, connectorsChannel, args[0], body); err != nil {
			return err
		}
		fmt.Println("Connector function updated successfully.")
		return nil
	},
}

var connectorsDeleteCmd = &cobra.Command{
	Use:   "delete <cf-id>",
	Short: "Delete a connector function",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := confirmDestructive(fmt.Sprintf("delete connector function %q", args[0])); err != nil {
			return err
		}
		c, orgName, err := connectorsClient()
		if err != nil {
			return err
		}
		if err := c.DeleteConnectorFunction(cmd.Context(), orgName, connectorsChannel, args[0]); err != nil {
			return err
		}
		fmt.Println("Connector function deleted successfully.")
		return nil
	},
}

// --- status / enable / disable ---

var connectorsStatusCmd = &cobra.Command{
	Use:   "status <cf-id> <PRODUCTION|TEST|DISABLED>",
	Short: "Set a connector function's operationalStatus",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return setConnectorStatus(cmd.Context(), args[0], strings.ToUpper(args[1]))
	},
}

var connectorsEnableCmd = &cobra.Command{
	Use:   "enable <cf-id>",
	Short: "Enable a connector function (sets operationalStatus=PRODUCTION)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return setConnectorStatus(cmd.Context(), args[0], "PRODUCTION")
	},
}

var connectorsDisableCmd = &cobra.Command{
	Use:   "disable <cf-id>",
	Short: "Disable a connector function (sets operationalStatus=DISABLED)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return setConnectorStatus(cmd.Context(), args[0], "DISABLED")
	},
}

func setConnectorStatus(ctx context.Context, id, status string) error {
	switch status {
	case "PRODUCTION", "TEST", "DISABLED":
	default:
		return fmt.Errorf("invalid status %q (want PRODUCTION, TEST, or DISABLED)", status)
	}
	c, orgName, err := connectorsClient()
	if err != nil {
		return err
	}
	if err := c.SetConnectorFunctionStatus(ctx, orgName, connectorsChannel, id, status); err != nil {
		return err
	}
	fmt.Printf("Connector function %s set to %s.\n", id, status)
	return nil
}

// --- catalog ---

var connectorsCatalogCmd = &cobra.Command{
	Use:   "catalog",
	Short: "Show the platform connector functions catalog (predefined templates)",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := activeProfile()
		if err != nil {
			return err
		}
		c := opengate.New(p.Host, p.Token, p.ClientOptions()...)
		data, err := c.ConnectorFunctionsCatalog(cmd.Context())
		if err != nil {
			return err
		}
		return printJSON(data)
	},
}

// --- pull / pull-all / wrap / deploy ---

var connectorsPullCmd = &cobra.Command{
	Use:     "pull <cf-id>",
	Aliases: []string{"unwrap"},
	Short:   "Pull a connector function into an editable directory (connectorfunction.json + javascript.js)",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, orgName, err := connectorsClient()
		if err != nil {
			return err
		}
		raw, err := c.GetConnectorFunction(cmd.Context(), orgName, connectorsChannel, args[0])
		if err != nil {
			return err
		}
		dir, err := unwrapArtifactTo(unwrap.ConnectorFunctionDescriptor(), raw, cfPullDir, &unwrap.Options{Force: cfPullForce, Warn: hintWarner()})
		if err != nil {
			return err
		}
		fmt.Printf("Connector function unwrapped to %s\n", dir)

		if p, perr := activeProfile(); perr == nil {
			d := unwrap.ConnectorFunctionDescriptor()
			recordBase(d.Kind, opengate.ParseConnectorFunctionSummary(raw).Identifier, d.NameOf(raw),
				dir, cfPullDir, raw, syncTarget(p, orgName, connectorsChannel))
			if !cfNoTypings {
				writeCFTypings(cmd, p, orgName, dir, raw)
			}
		}
		return nil
	},
}

var connectorsPullAllCmd = &cobra.Command{
	Use:     "pull-all",
	Aliases: []string{"unwrap-all"},
	Short:   "Pull every connector function of the channel into <dir>/<cf-slug>/",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, orgName, err := connectorsClient()
		if err != nil {
			return err
		}
		resp, err := c.ListConnectorFunctions(cmd.Context(), orgName, connectorsChannel)
		if err != nil {
			return err
		}
		if len(resp.ConnectorFunctions) == 0 {
			fmt.Println("No connector functions found.")
			return nil
		}
		// One Options for the whole batch: slug deduplication only works when
		// every artifact sees the slugs its siblings already claimed.
		opts := &unwrap.Options{Force: cfPullForce, Warn: hintWarner()}
		count := 0
		p, _ := activeProfile()
		var dms []opengate.Datamodel
		dmsResolved := false
		for _, raw := range resp.ConnectorFunctions {
			dir, err := unwrapArtifactTo(unwrap.ConnectorFunctionDescriptor(), raw, cfPullDir, opts)
			if err != nil {
				return err
			}
			fmt.Printf("  %s\n", dir)
			if p != nil {
				d := unwrap.ConnectorFunctionDescriptor()
				recordBase(d.Kind, opengate.ParseConnectorFunctionSummary(raw).Identifier, d.NameOf(raw),
					dir, cfPullDir, raw, syncTarget(p, orgName, connectorsChannel))
				if !cfNoTypings {
					if dms == nil && !dmsResolved {
						dms, dmsResolved = datamodelForTypings(cmd, p, orgName), true
					}
					writeTypings(dir, typegen.ContextForConnectorFunction(cfTypeOfPayload(raw)), dms, orgName, nil,
						typegen.DatastreamsReferencedBy(codeOf(raw)), typegen.ProtocolsFromCriteria(raw))
				}
			}
			count++
		}
		fmt.Printf("%d connector functions unwrapped to %s\n", count, cfPullDir)
		return nil
	},
}

var connectorsWrapCmd = &cobra.Command{
	Use:   "wrap <cf-dir>",
	Short: "Rebuild a connector function JSON from an unwrapped directory (no upload)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := unwrap.WrapConnectorFunction(args[0], hintWarner())
		if err != nil {
			return err
		}
		if cfWrapOut != "" {
			if err := os.WriteFile(cfWrapOut, data, 0o644); err != nil {
				return err
			}
			fmt.Printf("Connector function written to %s\n", cfWrapOut)
			return nil
		}
		fmt.Println(string(data))
		return nil
	},
}

var connectorsDeployCmd = &cobra.Command{
	Use:   "deploy <cf-dir>",
	Short: "Wrap + upload a connector function in one step (POST, or PUT with --update)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, orgName, err := connectorsClient()
		if err != nil {
			return err
		}
		body, err := unwrap.WrapConnectorFunction(args[0], hintWarner())
		if err != nil {
			return err
		}
		if p, perr := activeProfile(); perr == nil {
			warnIfMovedTarget(args[0], syncTarget(p, orgName, connectorsChannel))
		}

		if cfDeployUpdate {
			var cf struct {
				Identifier string `json:"identifier"`
			}
			if err := json.Unmarshal(body, &cf); err != nil || cf.Identifier == "" {
				return fmt.Errorf("--update requires an 'identifier' field in connectorfunction.json (pull the connector function first)")
			}
			if err := c.UpdateConnectorFunction(cmd.Context(), orgName, connectorsChannel, cf.Identifier, body); err != nil {
				return err
			}
			fmt.Printf("Connector function %s updated successfully.\n", cf.Identifier)
			return nil
		}

		if _, err := c.CreateConnectorFunction(cmd.Context(), orgName, connectorsChannel, body); err != nil {
			return err
		}
		fmt.Println("Connector function created successfully.")
		return nil
	},
}

// --- logs ---

var connectorsLogsCmd = &cobra.Command{
	Use:   "logs <cf-id>",
	Short: "Stream a connector function's execution logs (live, colourised by severity)",
	Long: `Stream the live execution logs of a connector function over WebSocket.

Traces are emitted by logger.trace/debug/info/warn/error calls inside the
connector function's JavaScript, colourised by severity. Press Ctrl-C to stop.

NOTE: a device only emits logs while its administrativeState is TESTING. With an
ACTIVE device the connector function still runs and collects data, but no logs are
streamed. Use --level DEBUG/TRACE to see logger.debug/trace (default is INFO).

Examples:
  og connectors logs <cf-id> --org sensehat
  og connectors logs <cf-id> --level TRACE --org sensehat`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return streamFunctionLogs(cmd.Context(), opengate.LoggerConnectorFunctions, connectorsChannel, args[0])
	},
}

// --- helpers ---

func connectorsClient() (*opengate.Client, string, error) {
	p, err := activeProfile()
	if err != nil {
		return nil, "", err
	}
	orgName, err := resolveOrg(p)
	if err != nil {
		return nil, "", err
	}
	return opengate.New(p.Host, p.Token, p.ClientOptions()...), orgName, nil
}

// --- init ---

var connectorsDiffCmd = newDiffCmd(diffSpec{
	Descriptor: unwrap.ConnectorFunctionDescriptor(),
	Use:        "diff <cf-dir>",
	Short:      "Compare a local connector function directory against the platform",
	Long: `Compare a locally-edited connector function against the one on the platform.

Metadata is reported structurally and the JavaScript textually. See
'og rules diff --help' for the state markers.

Examples:
  og connectors diff connectors/weather --org sensehat
  og connectors diff connectors/weather --against production
  og connectors diff connectors/weather --exit-code -o json`,
	Fetch: func(ctx context.Context, c *opengate.Client, org, id string) (json.RawMessage, error) {
		return c.GetConnectorFunction(ctx, org, connectorsChannel, id)
	},
})

var connectorsShowCmd = newShowCmd(unwrap.ConnectorFunctionDescriptor(),
	func(ctx context.Context, c *opengate.Client, org, id string) (json.RawMessage, error) {
		return c.GetConnectorFunction(ctx, org, connectorsChannel, id)
	},
	"show <cf-id>", "Print a connector function's remote code files",
	"og connectors show <cf-id> --org sensehat",
	"og connectors show <cf-id> --org sensehat --path javascript.js")

var connectorsValidateCmd = newValidateCmd(unwrap.ConnectorFunctionDescriptor(), "validate <cf-dir>", "Check a local connector function directory before deploying it")

var connectorsWatchCmd = newWatchCmd(watchSpec{
	Descriptor: unwrap.ConnectorFunctionDescriptor(),
	Use:        "watch <dir>",
	Short:      "Deploy connector functions as their files change",
	Fetch: func(ctx context.Context, c *opengate.Client, org, id string) (json.RawMessage, error) {
		return c.GetConnectorFunction(ctx, org, connectorsChannel, id)
	},
	Deploy: func(ctx context.Context, c *opengate.Client, org, id string, body json.RawMessage) error {
		return c.UpdateConnectorFunction(ctx, org, connectorsChannel, id, body)
	},
	Channel: func() string { return connectorsChannel },
})

func init() {
	connectorsCmd.AddCommand(connectorsWatchCmd)
	addWatchFlags(connectorsWatchCmd)
	connectorsCmd.AddCommand(connectorsValidateCmd)
	addValidateFlags(connectorsValidateCmd)
	connectorsCmd.AddCommand(connectorsDiffCmd)
	addDiffFlags(connectorsDiffCmd)
	connectorsCmd.AddCommand(connectorsShowCmd)
	addShowFlags(connectorsShowCmd)
	connectorsCmd.PersistentFlags().StringVar(&connectorsChannel, "channel", defaultChannel, "channel the connector function belongs to")

	connectorsCreateCmd.Flags().StringVarP(&cfCreateFile, "file", "f", "", "path to JSON file with connector function definition")
	connectorsCreateCmd.MarkFlagRequired("file")

	connectorsUpdateCmd.Flags().StringVarP(&cfUpdateFile, "file", "f", "", "path to JSON file with connector function definition")
	connectorsUpdateCmd.MarkFlagRequired("file")

	connectorsPullCmd.Flags().StringVar(&cfPullDir, "dir", "connectors", "destination directory")
	connectorsPullCmd.Flags().BoolVar(&cfPullForce, "force", false, "overwrite existing destination")
	connectorsPullCmd.Flags().BoolVar(&cfNoTypings, "no-typings", false, "skip generating og-globals.d.ts and jsconfig.json")
	connectorsPullAllCmd.Flags().StringVar(&cfPullDir, "dir", "connectors", "destination directory")
	connectorsPullAllCmd.Flags().BoolVar(&cfPullForce, "force", false, "overwrite existing destinations")
	connectorsPullAllCmd.Flags().BoolVar(&cfNoTypings, "no-typings", false, "skip generating og-globals.d.ts and jsconfig.json")

	connectorsWrapCmd.Flags().StringVar(&cfWrapOut, "out", "", "write connector function JSON to this file (default: stdout)")

	connectorsDeployCmd.Flags().BoolVar(&cfDeployUpdate, "update", false, "update an existing connector function (PUT) instead of creating (POST)")

	connectorsLogsCmd.Flags().StringVar(&logsLevel, "level", "INFO", "log level: ERROR | WARN | INFO | DEBUG | TRACE")

	connectorsCmd.AddCommand(connectorsListCmd)
	connectorsCmd.AddCommand(connectorsGetCmd)
	connectorsCmd.AddCommand(connectorsCreateCmd)
	connectorsCmd.AddCommand(connectorsUpdateCmd)
	connectorsCmd.AddCommand(connectorsDeleteCmd)
	connectorsCmd.AddCommand(connectorsStatusCmd)
	connectorsCmd.AddCommand(connectorsEnableCmd)
	connectorsCmd.AddCommand(connectorsDisableCmd)
	connectorsCmd.AddCommand(connectorsCatalogCmd)
	connectorsCmd.AddCommand(connectorsPullCmd)
	connectorsCmd.AddCommand(connectorsPullAllCmd)
	connectorsCmd.AddCommand(connectorsWrapCmd)
	connectorsCmd.AddCommand(connectorsDeployCmd)
	connectorsCmd.AddCommand(connectorsLogsCmd)

	rootCmd.AddCommand(connectorsCmd)
}

// writeCFTypings generates the declarations for one connector function: its
// execution context follows its type, and the protocol objects in scope follow
// the scheme of its south criteria.
func writeCFTypings(cmd *cobra.Command, p *config.Profile, orgName, dir string, raw json.RawMessage) {
	writeTypings(dir, typegen.ContextForConnectorFunction(cfTypeOfPayload(raw)),
		datamodelForTypings(cmd, p, orgName), orgName, nil,
		typegen.DatastreamsReferencedBy(codeOf(raw)), typegen.ProtocolsFromCriteria(raw))
}

// cfTypeOfPayload reads a connector function's type from its payload.
func cfTypeOfPayload(raw json.RawMessage) string {
	var cf struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(raw, &cf) != nil {
		return ""
	}
	return cf.Type
}

// codeOf returns an artifact's javascript field, for scanning the datastreams it
// references.
func codeOf(raw json.RawMessage) string {
	var a struct {
		JavaScript string `json:"javascript"`
	}
	if json.Unmarshal(raw, &a) != nil {
		return ""
	}
	return a.JavaScript
}
