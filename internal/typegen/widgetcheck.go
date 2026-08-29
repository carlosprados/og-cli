package typegen

import (
	"fmt"
	"strconv"
	"strings"
)

// This file recovers the diagnostics that a widget's own contract takes away.
//
// A widget script returns its result at the top level and may await at the top
// level, because the platform wraps it in an async function. TypeScript reports
// both — TS1108 and TS1375 — in every module configuration, so `og typegen`
// turns checking off for widgets and keeps completion. That trade loses the one
// check worth having: a misspelled $api method, which today only shows up at
// render time as "Data not found".
//
// So instead of asking TypeScript to accept the script as a file, this rebuilds
// the wrapper the platform puts around it and checks THAT. Inside a function
// body the return and the await are ordinary, nothing is suppressed, and the
// diagnostics that come back are real ones. The wrapper's own lines are then
// subtracted so every position points at the file the author edits.

// WrapperParams returns the parameters of the async function the platform wraps
// a widget's code in, in order.
//
// From the verified widget-JS reference: a customChart is handed the data and
// returns an ECharts option object; a customTable is additionally paged and
// delivers its rows through the callback.
func WrapperParams(c Context) []string {
	common := []string{
		"entityData", "relatedEntities", "timeserieData", "alarmData",
		"dashboardFilters", "filters",
	}
	if c == ContextWidgetTable {
		return append(common, "pageElements", "page", "callback")
	}
	return append(common, "callback")
}

// wrapperPrefix is the source placed above the widget's own first line. Kept on
// as few lines as possible and never indented, so a diagnostic's column is the
// author's column and only the line needs adjusting.
func wrapperPrefix(c Context) string {
	return fmt.Sprintf("async function __ogWidget(%s) {\n", strings.Join(WrapperParams(c), ", "))
}

// WrapWidget returns the widget's code inside the platform's wrapper, and the
// number of lines added above it.
func WrapWidget(c Context, code string) (wrapped string, offset int) {
	prefix := wrapperPrefix(c)
	offset = strings.Count(prefix, "\n")
	if !strings.HasSuffix(code, "\n") {
		code += "\n"
	}
	// The trailing call is what makes the parameters used rather than dead, and
	// keeps a lint that flags unused declarations quiet about the wrapper.
	return prefix + code + "}\n__ogWidget;\n", offset
}

// WidgetCheckConfig is the tsconfig used to check the wrapped source.
//
// Unlike the config written next to the widget for the editor, this one turns
// checking ON unconditionally: the whole point is that inside the wrapper there
// is nothing left to excuse.
func WidgetCheckConfig(wrappedFile string) string {
	return `{
  "compilerOptions": {
    "target": "es2017",
    "lib": ["es2017", "dom"],
    "allowJs": true,
    "checkJs": true,
    "noEmit": true,
    "strict": false,
    "noImplicitAny": false,
    "module": "esnext",
    "moduleResolution": "bundler",
    "moduleDetection": "force",
    "skipLibCheck": true
  },
  "include": ["` + wrappedFile + `", "og-globals.d.ts"]
}
`
}

// Diagnostic is one message from the type-checker, positioned in the author's
// file rather than in the generated wrapper.
type Diagnostic struct {
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (d Diagnostic) String() string {
	return fmt.Sprintf("%d:%d  %s  %s", d.Line, d.Column, d.Code, d.Message)
}

// ParseDiagnostics reads tsc's default output and maps it back onto the widget's
// own file.
//
// tsc writes `file.js(12,7): error TS2551: …`. Lines at or above the wrapper's
// own are dropped: a diagnostic there is about the generated scaffolding, not
// about anything the author wrote, and reporting it would be worse than useless.
func ParseDiagnostics(out string, offset int) []Diagnostic {
	var diags []Diagnostic
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		open := strings.LastIndex(line, "(")
		close := strings.Index(line, "): ")
		if open < 0 || close < open {
			continue
		}
		pos := strings.SplitN(line[open+1:close], ",", 2)
		if len(pos) != 2 {
			continue
		}
		lineNo, err1 := strconv.Atoi(strings.TrimSpace(pos[0]))
		colNo, err2 := strconv.Atoi(strings.TrimSpace(pos[1]))
		if err1 != nil || err2 != nil {
			continue
		}
		rest := line[close+3:]
		code, message := splitDiagnostic(rest)

		mapped := lineNo - offset
		if mapped < 1 {
			continue
		}
		diags = append(diags, Diagnostic{Line: mapped, Column: colNo, Code: code, Message: message})
	}
	return diags
}

// splitDiagnostic separates `error TS2551: message` into its code and text.
func splitDiagnostic(s string) (code, message string) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "error ")
	s = strings.TrimPrefix(s, "warning ")
	if idx := strings.Index(s, ": "); idx > 0 {
		return s[:idx], strings.TrimSpace(s[idx+2:])
	}
	return "", s
}

// ── known friction ───────────────────────────────────────────────────────────
//
// Two idioms are correct JavaScript that TypeScript rejects on principle, and
// both appear in production widgets. Reporting them would make the command
// redden working code, which is how a check earns its way into a .gitignore.
//
// They are separated rather than dropped: `--strict` shows them, because on a
// script being rewritten they are occasionally worth seeing.

// Classify splits diagnostics into the ones worth acting on and the ones that
// are TypeScript disagreeing with JavaScript.
//
// The tests are deliberately narrow. TS2339 is also the code for a genuinely
// misspelled member — the whole reason this command exists — so only the
// `never` form is set aside, and the arithmetic codes only when the offending
// line really does subtract dates.
func Classify(diags []Diagnostic, code string) (real, friction []Diagnostic) {
	lines := strings.Split(code, "\n")
	lineAt := func(n int) string {
		if n >= 1 && n <= len(lines) {
			return lines[n-1]
		}
		return ""
	}

	for _, d := range diags {
		switch {
		// `var x = null; … x.field` — TypeScript narrows x to `never` and
		// reports a read that happens after an assignment it cannot follow.
		case d.Code == "TS2339" && strings.Contains(d.Message, "type 'never'"):
			friction = append(friction, d)

		// `new Date(a) - new Date(b)` — the ordinary chronological sort, which
		// JavaScript resolves through valueOf.
		case (d.Code == "TS2362" || d.Code == "TS2363") && strings.Contains(lineAt(d.Line), "new Date("):
			friction = append(friction, d)

		default:
			real = append(real, d)
		}
	}
	return real, friction
}
