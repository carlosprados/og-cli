package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/carlosprados/og-cli/v2/internal/config"
	"github.com/carlosprados/og-cli/v2/internal/typegen"
	"github.com/carlosprados/og-cli/v2/pkg/opengate"
	"github.com/spf13/cobra"
)

var (
	typegenContext   string
	typegenOut       string
	typegenDatamodel string
	typegenNoJSConf  bool
)

var typegenCmd = &cobra.Command{
	Use:   "typegen",
	Short: "Generate TypeScript declarations for the platform globals in artifact JavaScript",
	Long: `Generate og-globals.d.ts (and jsconfig.json) so any LSP-capable editor gives
completion and diagnostics for the OpenGate globals in scope inside an
artifact's JavaScript — Neovim, VS Code, Cursor and Zed all pick these up
through tsserver with no editor-specific setup.

Two halves are written: the platform catalogue for the execution context
(functions, entity value shape, alarm severities), and — when a datamodel is
resolved — the organization's real datastream identifiers with their value
types, so entity['sensro.temperature'] is flagged before it is ever deployed.

By default every datamodel in the organization contributes its datastreams,
since a device's entity can carry datastreams from any of them. --datamodel
restricts the typings to one. Where two datamodels declare the same identifier
with different types, it is left untyped rather than guessed.

Regenerate after a datamodel change.

Examples:
  og typegen --context rule/ADVANCED --org sensehat
  og typegen --context rule/ADVANCED --org sensehat --datamodel multisensor --out rules/env-anomaly/
  og typegen --context rule/ADVANCED --out .` + "\n\nContexts: " + strings.Join(typegen.Contexts(), ", "),
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := typegen.Options{
			Context: typegen.Context(typegenContext),
			Version: version,
		}

		// The datamodel half is optional: typegen still helps without it.
		p, err := cfg.ActiveProfile(profile)
		if err == nil {
			if orgName, orgErr := resolveOrg(p); orgErr == nil {
				dms, dmErr := resolveDatamodels(cmd, p, orgName)
				if dmErr != nil {
					fmt.Fprintf(os.Stderr, "  hint: %v — generating platform datastreams only\n", dmErr)
				}
				opts.Datamodels, opts.OrgName = dms, orgName
			}
		}

		// When the destination is an unwrapped rule directory, type its declared
		// parameters too — the rule states each parameter's schema.
		if raw, err := os.ReadFile(filepath.Join(dirOrDot(typegenOut), "rule.json")); err == nil {
			opts.Parameters = typegen.ParametersFrom(raw)
			opts.ExtraDatastreams = typegen.DatastreamsTriggering(raw)
		}
		opts.ExtraDatastreams = append(opts.ExtraDatastreams, datastreamsInCode(dirOrDot(typegenOut))...)

		out, err := typegen.Generate(opts)
		if err != nil {
			return err
		}

		dir := dirOrDot(typegenOut)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		dtsPath := filepath.Join(dir, "og-globals.d.ts")
		if err := os.WriteFile(dtsPath, []byte(out), 0o644); err != nil {
			return err
		}
		fmt.Printf("Wrote %s\n", dtsPath)

		if !typegenNoJSConf {
			jsPath := filepath.Join(dir, "jsconfig.json")
			if err := os.WriteFile(jsPath, []byte(typegen.JSConfig), 0o644); err != nil {
				return err
			}
			fmt.Printf("Wrote %s\n", jsPath)
		}
		return nil
	},
}

// resolveDatamodels returns the datamodels to type against: the one named with
// --datamodel, or every datamodel in the organization.
//
// All of them by default, because a real tenant has many — sensehat has 27 — and
// a device's entity can carry datastreams from any of them. Typing against a
// single one would flag correct code as wrong, and refusing to choose (as this
// first did) throws away the whole point of the feature.
//
// One request: the datamodel search response already carries each model's
// categories and datastreams, so merging them costs nothing extra.
func resolveDatamodels(cmd *cobra.Command, p *config.Profile, orgName string) ([]opengate.Datamodel, error) {
	c := opengate.New(p.Host, p.Token, p.ClientOptions()...)

	resp, err := c.SearchDatamodels(cmd.Context(), nil)
	if err != nil {
		return nil, fmt.Errorf("listing datamodels: %w", err)
	}
	if len(resp.Datamodels) == 0 {
		return nil, fmt.Errorf("no datamodels found for organization %s", orgName)
	}

	if typegenDatamodel == "" {
		return resp.Datamodels, nil
	}

	for _, dm := range resp.Datamodels {
		if dm.Identifier == typegenDatamodel {
			return []opengate.Datamodel{dm}, nil
		}
	}
	names := make([]string, 0, len(resp.Datamodels))
	for _, dm := range resp.Datamodels {
		names = append(names, dm.Identifier)
	}
	sort.Strings(names)
	return nil, fmt.Errorf("datamodel %q not found in organization %s (available: %s)",
		typegenDatamodel, orgName, strings.Join(names, ", "))
}

func init() {
	typegenCmd.Flags().StringVar(&typegenContext, "context", string(typegen.ContextRuleAdvanced),
		"execution context: "+strings.Join(typegen.Contexts(), " | "))
	typegenCmd.Flags().StringVar(&typegenOut, "out", ".", "directory to write og-globals.d.ts and jsconfig.json into")
	typegenCmd.Flags().StringVar(&typegenDatamodel, "datamodel", "", "restrict the typings to one datamodel (default: every datamodel in the organization)")
	typegenCmd.Flags().BoolVar(&typegenNoJSConf, "no-jsconfig", false, "do not write jsconfig.json")

	rootCmd.AddCommand(typegenCmd)
}

// dirOrDot defaults an empty directory to the working directory.
func dirOrDot(dir string) string {
	if dir == "" {
		return "."
	}
	return dir
}

// writeTypings generates og-globals.d.ts and jsconfig.json into an unwrapped
// artifact directory. Failures are reported and swallowed: a pull that fetched
// the artifact correctly must not fail because the typings could not be built.
func writeTypings(dir string, ctx typegen.Context, dms []opengate.Datamodel, orgName string, params []typegen.Parameter, extra []string) {
	out, err := typegen.Generate(typegen.Options{
		Context: ctx, Datamodels: dms, OrgName: orgName, Parameters: params,
		ExtraDatastreams: extra, Version: version,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "  hint: typings not generated: %v\n", err)
		return
	}
	if err := os.WriteFile(filepath.Join(dir, "og-globals.d.ts"), []byte(out), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "  hint: writing og-globals.d.ts: %v\n", err)
		return
	}
	if err := os.WriteFile(filepath.Join(dir, "jsconfig.json"), []byte(typegen.JSConfig), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "  hint: writing jsconfig.json: %v\n", err)
	}
}

// datamodelForTypings resolves the organization's datamodel once per command,
// for the typings written during a pull. A failure is not fatal: the typings
// degrade to platform datastreams only.
func datamodelForTypings(cmd *cobra.Command, p *config.Profile, orgName string) []opengate.Datamodel {
	dms, err := resolveDatamodels(cmd, p, orgName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  hint: %v — typings will declare platform datastreams only\n", err)
		return nil
	}
	return dms
}

// datastreamsInCode collects the datastream identifiers the .js files in an
// artifact directory already read, so the declarations cover working code.
func datastreamsInCode(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".js" {
			continue
		}
		code, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		out = append(out, typegen.DatastreamsReferencedBy(string(code))...)
	}
	return out
}
