package cmd

import (
	"encoding/json"
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
		// A connector function directory: its type decides the context and its
		// south criteria the protocols, so neither has to be passed by hand.
		if raw, err := os.ReadFile(filepath.Join(dirOrDot(typegenOut), "connectorfunction.json")); err == nil {
			if !cmd.Flags().Changed("context") {
				opts.Context = typegen.ContextForConnectorFunction(cfTypeOfPayload(raw))
			}
			opts.Protocols = typegen.ProtocolsFromCriteria(raw)
		}
		// A widget directory: its definition.type decides the context. Widget code
		// has no datamodel half at all — it reads the platform through $api — so
		// anything resolved above is dropped rather than emitted meaninglessly.
		if raw, err := os.ReadFile(filepath.Join(dirOrDot(typegenOut), "widget.json")); err == nil {
			wType := widgetTypeOf(raw)
			ctx, scriptable := typegen.ContextForWidget(wType)
			switch {
			case scriptable && !cmd.Flags().Changed("context"):
				opts.Context = ctx
			case !scriptable && !cmd.Flags().Changed("context"):
				// A widget directory whose kind takes no script. Falling through
				// would write rule globals — an `entity`, a datastream map — into a
				// directory where none of it is in scope, which is worse than
				// writing nothing.
				return fmt.Errorf("widget type %q takes no JavaScript, so there is nothing to type here\n"+
					"  only customChart and customTable carry a script; other widgets are configured, not programmed\n"+
					"  pass --context explicitly to override", wType)
			}
		}
		if typegen.IsWidgetContext(opts.Context) {
			opts.Datamodels, opts.Parameters, opts.ExtraDatastreams, opts.Protocols = nil, nil, nil, nil
		} else {
			opts.ExtraDatastreams = append(opts.ExtraDatastreams, datastreamsInCode(dirOrDot(typegenOut))...)
		}

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
			var conf string
			if typegen.IsWidgetContext(opts.Context) {
				file, code := widgetCodeInDir(dir)
				conf = typegen.JSConfigForWidget(file, code)
			} else {
				conf = typegen.JSConfigFor(codeInDir(dir))
			}
			if err := os.WriteFile(jsPath, []byte(conf), 0o644); err != nil {
				return err
			}
			fmt.Printf("Wrote %s\n", jsPath)
		}

		// $api is the opengate-js package, so the editor needs it installed to
		// resolve the import in og-globals.d.ts. Declaring the dependency is the
		// whole of it: `npm install` in the directory and completion works.
		if typegen.IsWidgetContext(opts.Context) {
			pkgPath := filepath.Join(dir, "package.json")
			if _, err := os.Stat(pkgPath); os.IsNotExist(err) {
				if err := os.WriteFile(pkgPath, []byte(widgetPackageJSON), 0o644); err != nil {
					return err
				}
				fmt.Printf("Wrote %s  (run `npm install` here so $api resolves)\n", pkgPath)
			}
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
func writeTypings(dir string, ctx typegen.Context, dms []opengate.Datamodel, orgName string,
	params []typegen.Parameter, extra []string, protocols []string) {
	out, err := typegen.Generate(typegen.Options{
		Context: ctx, Datamodels: dms, OrgName: orgName, Parameters: params,
		ExtraDatastreams: extra, Protocols: protocols, Version: version,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "  hint: typings not generated: %v\n", err)
		return
	}
	if err := os.WriteFile(filepath.Join(dir, "og-globals.d.ts"), []byte(out), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "  hint: writing og-globals.d.ts: %v\n", err)
		return
	}
	if err := os.WriteFile(filepath.Join(dir, "jsconfig.json"), []byte(typegen.JSConfigFor(codeInDir(dir))), 0o644); err != nil {
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

// codeInDir concatenates the .js files in an artifact directory, so the
// generated jsconfig can adapt to what they contain.
func codeInDir(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	var b strings.Builder
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".js" {
			continue
		}
		if code, err := os.ReadFile(filepath.Join(dir, e.Name())); err == nil {
			b.Write(code)
			b.WriteString("\n")
		}
	}
	return b.String()
}

// widgetPackageJSON declares the one dependency a widget directory needs.
//
// opengate-js publishes its own TypeScript declarations from 16.0.0 — 246 of
// them, generated from its JSDoc by tsc on `prepack` — so nothing about the
// $api surface is generated here. The version is a caret range because the
// declarations track the library, and typing against an older minor than the
// platform serves would report methods as missing that exist.
//
// TypeScript itself is a devDependency so that `og widget check` has a compiler
// to run without downloading one: one `npm install` in the directory sets up
// both the editor and the check.
const widgetPackageJSON = `{
  "name": "og-widget",
  "private": true,
  "description": "Type declarations for OpenGate widget JavaScript. Run npm install so $api resolves.",
  "dependencies": {
    "opengate-js": "^16.0.0"
  },
  "devDependencies": {
    "typescript": "^5"
  }
}
`

// widgetTypeOf reads definition.type out of a widget.json.
func widgetTypeOf(raw []byte) string {
	var w struct {
		Definition struct {
			Type string `json:"type"`
		} `json:"definition"`
	}
	if err := json.Unmarshal(raw, &w); err != nil {
		return ""
	}
	return w.Definition.Type
}

// widgetCodeInDir returns the widget's code file and its contents.
//
// A widget directory holds exactly one script under the name the extractor gave
// it — `_widgetConfigCode.js` for the custom kinds — and the config is scoped to
// that single file rather than to *.js, because sibling widgets unwrapped into
// neighbouring directories share the global scope when they are checked
// together and their top-level `var`s collide.
func widgetCodeInDir(dir string) (string, string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", ""
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".js" {
			continue
		}
		code, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		return e.Name(), string(code)
	}
	return "", ""
}
