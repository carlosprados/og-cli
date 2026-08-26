package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosprados/og-cli/v2/internal/unwrap"
)

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A false positive is worse than no check at all: validate blocks a deploy, and
// watch refuses to push. These are shapes taken from real production code.
func TestBalanceAcceptsRealCode(t *testing.T) {
	cases := map[string]string{
		// From a live connector function on sensehat. The escaped quote inside
		// the regex desynced an earlier version of this check and reported an
		// error on working code.
		"regex with an escaped quote":   `payload.error = httpResp.body.message.replace(/\'/g, '');`,
		"regex with a slash in a class": `var re = /[/{}]/g; if (re.test(s)) { log('yes'); }`,
		"division is not a regex":       `var half = (a + b) / 2; var q = total/count;`,
		"regex after return":            `function f() { return /ab+c/.test(s); }`,
		"brace inside a string":         `log('} not a brace {'); var o = { a: 1 };`,
		"brace inside a template":       "var s = `${a} } {`; var o = { b: 2 };",
		"line comment with brackets":    "// ) ] }\nvar o = { c: 3 };",
		"block comment with brackets":   "/* ) ] } */\nvar o = { d: 4 };",
		"escaped quote in a string":     `var s = 'it\'s fine'; var o = { e: 5 };`,
		"nested calls and objects":      `f(g(h({a: [1, 2, {b: 3}]})));`,
		"division then regex":           `var x = a / b; var re = /c/;`,
	}
	for label, code := range cases {
		if got := checkBalance("f.js", code); len(got) != 0 {
			t.Errorf("%s: false positive: %v\n  code: %s", label, got, code)
		}
	}
}

func TestBalanceCatchesRealErrors(t *testing.T) {
	cases := map[string]string{
		"unclosed brace":     "if (x) {\n  log('a');\n",
		"unclosed paren":     "log('a';\n",
		"extra close":        "if (x) { log('a'); }}\n",
		"mismatched":         "if (x) { log('a'); )\n",
		"unclosed in a nest": "f(g(h(1));\n",
	}
	for label, code := range cases {
		got := checkBalance("f.js", code)
		if len(got) == 0 {
			t.Errorf("%s: not detected\n  code: %s", label, code)
			continue
		}
		if got[0].Severity != Error {
			t.Errorf("%s: severity = %s, want error", label, got[0].Severity)
		}
		if got[0].Line < 1 {
			t.Errorf("%s: no line reported", label)
		}
	}
}

func TestArtifactValidRule(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "rule.json", `{"identifier":"r-1","name":"R","mode":"ADVANCED"}`)
	write(t, dir, "javascript.js", "if (entity['a']) { logger.info('ok'); }\n")

	r := Artifact(unwrap.RuleDescriptor(), dir)
	if !r.OK() || len(r.Findings) != 0 {
		t.Errorf("a valid rule should produce nothing: %+v", r.Findings)
	}
}

func TestArtifactMissingMetadata(t *testing.T) {
	r := Artifact(unwrap.RuleDescriptor(), t.TempDir())
	if r.OK() {
		t.Error("a directory with no metadata file must not validate")
	}
}

func TestArtifactMalformedJSONReportsLine(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "rule.json", "{\n  \"identifier\": \"r-1\",\n  \"name\",: \"R\"\n}")

	r := Artifact(unwrap.RuleDescriptor(), dir)
	if r.OK() {
		t.Fatal("malformed JSON must be an error")
	}
	if r.Findings[0].Line != 3 {
		t.Errorf("line = %d, want 3 — a stray comma should be findable", r.Findings[0].Line)
	}
}

// An ADVANCED rule with no code would deploy and do nothing.
func TestArtifactAdvancedRuleNeedsCode(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "rule.json", `{"identifier":"r-1","name":"R","mode":"ADVANCED"}`)

	r := Artifact(unwrap.RuleDescriptor(), dir)
	if r.OK() {
		t.Error("an ADVANCED rule with no javascript.js must be an error")
	}
}

// An EASY rule legitimately has no code.
func TestArtifactEasyRuleNeedsNoCode(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "rule.json", `{"identifier":"r-1","name":"R","mode":"EASY"}`)

	if r := Artifact(unwrap.RuleDescriptor(), dir); !r.OK() {
		t.Errorf("an EASY rule needs no code: %+v", r.Findings)
	}
}

// A stray .js is silently not deployed, which is exactly the surprise worth
// warning about.
func TestArtifactStrayJSWarns(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "rule.json", `{"identifier":"r-1","name":"R","mode":"EASY"}`)
	write(t, dir, "helper.js", "function helper(){ return 1; }\n")

	r := Artifact(unwrap.RuleDescriptor(), dir)
	if !r.OK() {
		t.Error("a stray file is a warning, not an error — it does not break the deploy")
	}
	if len(r.Findings) != 1 || !strings.Contains(r.Findings[0].Message, "will NOT be deployed") {
		t.Errorf("expected a warning about the stray file, got %+v", r.Findings)
	}
}

// A provision function that does not define both entry points fails mid-bulk,
// with entities half-processed.
func TestArtifactProvisionEntryPoints(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "provisionfunction.json", `{"provisionProcessorId":"p-1","name":"P"}`)
	write(t, dir, "scriptProcessor__script.js", "function normalizeRawObject(o){ return o; }\n")

	r := Artifact(unwrap.ProvisionFunctionDescriptor(), dir)
	if r.OK() {
		t.Fatal("a missing actionsPlanning must be an error")
	}
	if !strings.Contains(r.Findings[0].Message, "actionsPlanning") {
		t.Errorf("the finding should name the missing entry point: %+v", r.Findings)
	}

	write(t, dir, "scriptProcessor__script.js",
		"function normalizeRawObject(o){ return o; }\nfunction actionsPlanning(n){ return []; }\n")
	if r := Artifact(unwrap.ProvisionFunctionDescriptor(), dir); !r.OK() {
		t.Errorf("both entry points present should validate: %+v", r.Findings)
	}
}

// A connector function that can never be matched deploys fine and never fires —
// the failure mode this tooling exists to prevent.
func TestArtifactConnectorMatchability(t *testing.T) {
	cases := []struct {
		name, meta string
		wantIssue  string
	}{
		{"REQUEST without operationName", `{"identifier":"c","name":"C","type":"REQUEST"}`, "operationName"},
		{"COLLECTION without southCriterias", `{"identifier":"c","name":"C","type":"COLLECTION"}`, "southCriterias"},
		{"no type at all", `{"identifier":"c","name":"C"}`, "has no type"},
		{"unknown type", `{"identifier":"c","name":"C","type":"WEIRD"}`, "unknown type"},
	}
	for _, c := range cases {
		dir := t.TempDir()
		write(t, dir, "connectorfunction.json", c.meta)
		write(t, dir, "javascript.js", "return null;\n")

		r := Artifact(unwrap.ConnectorFunctionDescriptor(), dir)
		found := false
		for _, f := range r.Findings {
			if strings.Contains(f.Message, c.wantIssue) {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: expected a finding mentioning %q, got %+v", c.name, c.wantIssue, r.Findings)
		}
	}

	// A fully specified one is clean.
	dir := t.TempDir()
	write(t, dir, "connectorfunction.json",
		`{"identifier":"c","name":"C","type":"COLLECTION","southCriterias":[{"path":"/x"}]}`)
	write(t, dir, "javascript.js", "return ogCollection();\n")
	if r := Artifact(unwrap.ConnectorFunctionDescriptor(), dir); !r.OK() {
		t.Errorf("a matchable connector function should validate: %+v", r.Findings)
	}
}

func TestMissingIdentifierIsOnlyAWarning(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "rule.json", `{"name":"R","mode":"EASY"}`)

	r := Artifact(unwrap.RuleDescriptor(), dir)
	if !r.OK() {
		t.Error("no identifier must not block: deploy can still create the artifact")
	}
	if len(r.Findings) != 1 || r.Findings[0].Severity != Warning {
		t.Errorf("expected one warning, got %+v", r.Findings)
	}
}
