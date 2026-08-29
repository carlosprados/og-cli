package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/carlosprados/og-cli/v2/internal/output"
	"github.com/carlosprados/og-cli/v2/internal/typegen"
	"github.com/spf13/cobra"
)

var (
	widgetCheckExitCode bool
	widgetCheckStrict   bool
)

var widgetCmd = &cobra.Command{
	Use:   "widget",
	Short: "Work with an unwrapped widget directory",
	Long: `Commands for a single widget unwrapped by 'og workspace pull'.

A widget directory holds widget.json and, for the two kinds that carry code,
one JavaScript file. See 'og workspace pull' for the tree it lives in.`,
}

var widgetCheckCmd = &cobra.Command{
	Use:   "check [widget-dir]",
	Short: "Type-check a widget's JavaScript against the real $api",
	Long: `Type-check a widget's JavaScript, including its $api calls.

Runs locally against the opengate-js declarations installed in the widget
directory. No API call and no credentials, so it works offline and as a CI gate.

Why this exists rather than just a jsconfig: the platform wraps widget code in
an async function, so the script returns its result at the top level and may
await there. TypeScript reports both as errors on a plain file, which is why the
generated jsconfig keeps completion and turns checking off. This command
rebuilds that wrapper and checks the code inside it, where the return and the
await are ordinary — so nothing has to be excused, and a misspelled $api method
becomes an error here instead of "Data not found" at render time:

    _widgetConfigCode.js:17:15  TS2551  Property 'datapointsSearchBuildr' does
    not exist on type 'OpenGateAPI'. Did you mean 'datapointsSearchBuilder'?

Positions refer to your file: the wrapper's own lines are subtracted.

Requires the directory's dependencies to be installed — 'og typegen' writes the
package.json, then 'npm install' once. With --exit-code: 1 when there are
diagnostics, 0 when there are none, 2 on failure.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := "."
		if len(args) == 1 {
			dir = args[0]
		}

		ctx, codeFile, code, err := widgetSource(dir)
		if err != nil {
			return err
		}

		tsc, err := findTSC(dir)
		if err != nil {
			return err
		}

		all, err := runWidgetCheck(tsc, dir, ctx, codeFile, code)
		if err != nil {
			return err
		}
		diags, friction := typegen.Classify(all, code)
		if widgetCheckStrict {
			diags, friction = all, nil
		}

		if outFmt == output.FormatJSON {
			payload := struct {
				Widget      string               `json:"widget"`
				Context     string               `json:"context"`
				File        string               `json:"file"`
				Diagnostics []typegen.Diagnostic `json:"diagnostics"`
				SetAside    []typegen.Diagnostic `json:"setAside"`
				OK          bool                 `json:"ok"`
			}{widgetName(dir), string(ctx), codeFile, diags, friction, len(diags) == 0}
			if err := output.PrintEnvelope(os.Stdout, "widget-check", payload); err != nil {
				return err
			}
		} else {
			path := filepath.Join(dir, codeFile)
			for _, d := range diags {
				fmt.Printf("%s:%d:%d  %s  %s\n", path, d.Line, d.Column, d.Code, d.Message)
			}
			if len(diags) == 0 {
				fmt.Printf("%s: no problems found.\n", path)
			} else {
				fmt.Printf("\n%d problem(s).\n", len(diags))
			}
			if len(friction) > 0 {
				fmt.Printf("\n%d diagnostic(s) set aside as TypeScript disagreeing with correct JavaScript\n"+
					"(a date subtraction, or a variable narrowed to `never` after `= null`). --strict shows them.\n",
					len(friction))
			}
		}

		if widgetCheckExitCode && len(diags) > 0 {
			return diffFound()
		}
		return nil
	},
}

// widgetSource resolves what to check: the widget's context, its code file name
// and the code itself.
func widgetSource(dir string) (typegen.Context, string, string, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "widget.json"))
	if err != nil {
		return "", "", "", fmt.Errorf("%s is not an unwrapped widget directory: %w", dir, err)
	}
	var w struct {
		Definition struct {
			Type string `json:"type"`
		} `json:"definition"`
	}
	if err := json.Unmarshal(raw, &w); err != nil {
		return "", "", "", fmt.Errorf("reading %s: %w", filepath.Join(dir, "widget.json"), err)
	}
	ctx, scriptable := typegen.ContextForWidget(w.Definition.Type)
	if !scriptable {
		return "", "", "", fmt.Errorf("widget type %q carries no JavaScript, so there is nothing to check\n"+
			"  only customChart and customTable have a script", w.Definition.Type)
	}

	name, code := widgetCodeInDir(dir)
	if name == "" {
		return "", "", "", fmt.Errorf("no .js file in %s — nothing to check", dir)
	}
	return ctx, name, code, nil
}

// findTSC locates the TypeScript compiler installed in the widget directory.
//
// Deliberately local and never `npx`: npx would silently download a compiler
// from the network on a machine that has none, which is not something a check
// command should do behind the user's back.
func findTSC(dir string) (string, error) {
	name := "tsc"
	if runtime.GOOS == "windows" {
		name = "tsc.cmd"
	}
	candidate := filepath.Join(dir, "node_modules", ".bin", name)
	if _, err := os.Stat(candidate); err == nil {
		abs, err := filepath.Abs(candidate)
		if err != nil {
			return candidate, nil
		}
		return abs, nil
	}
	return "", fmt.Errorf("no TypeScript compiler in %s\n"+
		"  run `npm install` there first — `og typegen` writes the package.json that declares it\n"+
		"  (checking is local: og never downloads a compiler on your behalf)",
		filepath.Join(dir, "node_modules"))
}

// runWidgetCheck writes the wrapped source and its config into the widget
// directory, runs the compiler, and removes them again.
//
// They have to live in the directory rather than in a temporary one: `$api`
// resolves through that directory's node_modules, and og-globals.d.ts sits
// beside the code.
func runWidgetCheck(tsc, dir string, ctx typegen.Context, codeFile, code string) ([]typegen.Diagnostic, error) {
	wrapped, offset := typegen.WrapWidget(ctx, code)

	base := ".og-widget-check"
	srcName := base + ".js"
	confName := base + ".tsconfig.json"
	srcPath := filepath.Join(dir, srcName)
	confPath := filepath.Join(dir, confName)

	if err := os.WriteFile(srcPath, []byte(wrapped), 0o644); err != nil {
		return nil, fmt.Errorf("writing the wrapped source: %w", err)
	}
	defer func() { _ = os.Remove(srcPath) }()

	if err := os.WriteFile(confPath, []byte(typegen.WidgetCheckConfig(srcName)), 0o644); err != nil {
		return nil, fmt.Errorf("writing the check config: %w", err)
	}
	defer func() { _ = os.Remove(confPath) }()

	out, err := exec.Command(tsc, "-p", confPath).CombinedOutput()
	// A non-zero exit is how tsc reports diagnostics, so it is not a failure on
	// its own. Output that parses into none, with a non-zero exit, is.
	diags := typegen.ParseDiagnostics(string(out), offset)
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return nil, fmt.Errorf("running %s: %w", tsc, err)
		}
		if len(diags) == 0 {
			return nil, fmt.Errorf("the type-checker failed without reporting a diagnostic:\n%s", strings.TrimSpace(string(out)))
		}
	}
	return diags, nil
}

func init() {
	widgetCheckCmd.Flags().BoolVar(&widgetCheckExitCode, "exit-code", false,
		"exit 1 when there are diagnostics, 0 when there are none, 2 on failure")
	widgetCheckCmd.Flags().BoolVar(&widgetCheckStrict, "strict", false,
		"also report the diagnostics normally set aside as TypeScript disagreeing with correct JavaScript")
	widgetCmd.AddCommand(widgetCheckCmd)
	rootCmd.AddCommand(widgetCmd)
}

// widgetName is the directory's own name, resolved so that running the command
// from inside the widget does not report the widget as ".".
func widgetName(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return filepath.Base(dir)
	}
	return filepath.Base(abs)
}
