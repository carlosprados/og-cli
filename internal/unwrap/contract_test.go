package unwrap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func decode(t *testing.T, raw string) any {
	t.Helper()
	var node any
	if err := json.Unmarshal([]byte(raw), &node); err != nil {
		t.Fatal(err)
	}
	return node
}

// A declared path is extracted whatever its contents. This is the property the
// content heuristic could not provide: the file exists because the schema says
// the field carries code, so its path is stable across edits.
func TestDeclaredPathExtractedRegardlessOfContent(t *testing.T) {
	cases := map[string]string{
		"bare assignment, no keyword": "cellFormatter.customValue = value;",
		"short arrow":                 "v => v",
		"empty":                       "",
		"one word":                    "x",
	}
	for label, code := range cases {
		body, _ := json.Marshal(map[string]any{"javascript": code, "name": "r"})
		_, files, ctxs := RuleContract().Extract(decode(t, string(body)), nil)
		got, ok := files["javascript.js"]
		if !ok {
			t.Errorf("%s: javascript.js not extracted", label)
			continue
		}
		if got != code {
			t.Errorf("%s: content = %q, want %q", label, got, code)
		}
		if ctxs["javascript.js"] != CtxRuleAdvanced {
			t.Errorf("%s: context = %q, want %q", label, ctxs["javascript.js"], CtxRuleAdvanced)
		}
	}
}

// The connector function's execution context follows its type, because the
// globals in scope differ per type.
func TestConnectorFunctionContextFollowsType(t *testing.T) {
	for cfType, want := range map[string]ExecCtx{
		"REQUEST":    CtxCFRequest,
		"RESPONSE":   CtxCFResponse,
		"COLLECTION": CtxCFCollection,
		"":           CtxUnknown,
	} {
		raw := `{"type":"` + cfType + `","javascript":"function f(){return 1;}"}`
		_, _, ctxs := ConnectorFunctionContract(cfType).Extract(decode(t, raw), nil)
		if got := ctxs["javascript.js"]; got != want {
			t.Errorf("type %q: context = %q, want %q", cfType, got, want)
		}
	}
}

// Where the declared paths are exhaustive, an undeclared string that looks like
// code stays inline and is reported. Silently extracting it would put a file in
// the tree that wrap cannot map back to a field.
func TestUndeclaredCodeIsReportedNotExtracted(t *testing.T) {
	raw := `{"javascript":"function f(){return 1;}","notes":"remember to return the device before the let expires"}`

	var warns []Warning
	_, files, _ := RuleContract().Extract(decode(t, raw), func(w Warning) { warns = append(warns, w) })

	if _, extracted := files["notes.js"]; extracted {
		t.Error("an undeclared field must not be extracted for a flat family")
	}
	if len(warns) != 1 || warns[0].Path != "notes" {
		t.Fatalf("expected one warning about `notes`, got %v", warns)
	}
	if !strings.Contains(warns[0].Message, "not a declared code path") {
		t.Errorf("warning should say the field is undeclared: %q", warns[0].Message)
	}
}

// Widgets keep the heuristic as a transitional fallback — refusing to extract
// an unlisted field would cost a user their editable file — but it warns.
func TestWidgetHeuristicFallbackWarns(t *testing.T) {
	code := "let total = 0; for (const x of items) { total += x; } return total;"
	body, _ := json.Marshal(map[string]any{"customLogic": code})

	var warns []Warning
	_, files, ctxs := WidgetContract().Extract(decode(t, string(body)), func(w Warning) { warns = append(warns, w) })

	if files["customLogic.js"] != code {
		t.Error("widget fallback should still extract undeclared code")
	}
	if ctxs["customLogic.js"] != CtxUnknown {
		t.Errorf("fallback context = %q, want %q", ctxs["customLogic.js"], CtxUnknown)
	}
	if len(warns) != 1 || !strings.Contains(warns[0].Message, "not a declared code field") {
		t.Fatalf("expected a warning naming the undeclared field, got %v", warns)
	}
}

// `operation` holds an operation name (REFRESH_INFO), not code. It used to be
// on the allowlist and produced one-word .js files.
func TestOperationNameIsNotCode(t *testing.T) {
	raw := `{"operation":"REFRESH_INFO","_widgetConfigCode":"chart.series[0].data = ds;"}`
	_, files, _ := WidgetContract().Extract(decode(t, raw), nil)

	if _, extracted := files["operation.js"]; extracted {
		t.Error("`operation` is an operation name, not code — it must stay inline")
	}
	if _, extracted := files["_widgetConfigCode.js"]; !extracted {
		t.Error("_widgetConfigCode must still be extracted")
	}
}

// A .js file the contract does not declare is ignored on wrap, not injected as
// a bogus payload field. This is what makes an auxiliary file harmless — and
// is a precondition for shared modules and generated typings living in the
// artifact directory.
func TestStrayJSFileIsNotInjected(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "rule.json"), `{"name":"r","identifier":"r-1"}`)
	writeFile(t, filepath.Join(dir, "javascript.js"), "function f(){return 1;}")
	writeFile(t, filepath.Join(dir, "helper.js"), "function helper(){return 2;}")

	var warns []Warning
	out, err := WrapRule(dir, func(w Warning) { warns = append(warns, w) })
	if err != nil {
		t.Fatal(err)
	}

	var rule map[string]any
	if err := json.Unmarshal(out, &rule); err != nil {
		t.Fatal(err)
	}
	if _, present := rule["helper"]; present {
		t.Error("a stray helper.js must not become a payload field")
	}
	if rule["javascript"] != "function f(){return 1;}" {
		t.Errorf("declared code not reinjected: %v", rule["javascript"])
	}
	if len(warns) != 1 || !strings.Contains(warns[0].Message, "ignored") {
		t.Errorf("expected a warning about the ignored file, got %v", warns)
	}
}

// The full pull → edit → wrap cycle for a provision function, whose code sits
// at a nested path.
func TestProvisionFunctionNestedCodePath(t *testing.T) {
	raw := `{"provisionProcessorId":"p-1","name":"pf","configurationParams":{"spreadsheet":{"sheetName":"S"}},
	          "scriptProcessor":{"script":"function normalizeRawObject(o){return o;}"}}`
	dir := t.TempDir()

	ppDir, err := UnwrapProvisionProcessor(json.RawMessage(raw), dir, &Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(ppDir, "scriptProcessor__script.js")); err != nil {
		t.Fatalf("nested code path not extracted: %v", err)
	}

	// Edit it, wrap, and confirm the edit lands at the nested field.
	writeFile(t, filepath.Join(ppDir, "scriptProcessor__script.js"), "function normalizeRawObject(o){return null;}")
	out, err := WrapProvisionProcessor(ppDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	var pp struct {
		ScriptProcessor struct {
			Script string `json:"script"`
		} `json:"scriptProcessor"`
		ConfigurationParams struct {
			Spreadsheet struct {
				SheetName string `json:"sheetName"`
			} `json:"spreadsheet"`
		} `json:"configurationParams"`
	}
	if err := json.Unmarshal(out, &pp); err != nil {
		t.Fatal(err)
	}
	if pp.ScriptProcessor.Script != "function normalizeRawObject(o){return null;}" {
		t.Errorf("edited code did not land at scriptProcessor.script: %q", pp.ScriptProcessor.Script)
	}
	if pp.ConfigurationParams.Spreadsheet.SheetName != "S" {
		t.Error("unrelated config lost across the cycle")
	}
}
