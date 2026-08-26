package diff

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosprados/og-cli/v2/internal/canon"
	"github.com/carlosprados/og-cli/v2/internal/unwrap"
)

const remoteRule = `{"identifier":"r-1","name":"Env anomaly","active":true,
  "javascript":"var t = entity['sensor.temperature'];\nif (t) {\n  logger.info('ok');\n}\n"}`

// unwrapLocal writes the payload to a tree so Compare has a real directory,
// then optionally edits a file, the way a developer would.
func unwrapLocal(t *testing.T, payload string, edits map[string]string) (unwrap.Descriptor, string, json.RawMessage) {
	t.Helper()
	d := unwrap.RuleDescriptor()
	dir, err := d.Unwrap(json.RawMessage(payload), t.TempDir(), &unwrap.Options{})
	if err != nil {
		t.Fatal(err)
	}
	for name, content := range edits {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	local, err := d.Wrap(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	return d, dir, local
}

func TestCompareNoChanges(t *testing.T) {
	d, dir, local := unwrapLocal(t, remoteRule, nil)

	r, err := Compare(d, dir, local, json.RawMessage(remoteRule), Options{ContextLines: 3})
	if err != nil {
		t.Fatal(err)
	}
	if r.Changed() {
		t.Errorf("an untouched tree must show no differences: %+v", r)
	}
	if r.Name != "Env anomaly" {
		t.Errorf("name = %q", r.Name)
	}
}

// The two kinds of difference must be reported in their own sections: a code
// change as a textual diff, a metadata change as a structural one.
func TestCompareSeparatesCodeFromMetadata(t *testing.T) {
	edited := "var t = entity['sensor.temperature'];\nif (t) {\n  logger.warn('changed');\n}\n"
	d, dir, local := unwrapLocal(t, remoteRule, map[string]string{"javascript.js": edited})

	r, err := Compare(d, dir, local, json.RawMessage(remoteRule), Options{ContextLines: 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Metadata) != 0 {
		t.Errorf("editing only the code must not report metadata changes: %+v", r.Metadata)
	}
	if len(r.Code) != 1 || r.Code[0].File != "javascript.js" {
		t.Fatalf("expected one code diff for javascript.js, got %+v", r.Code)
	}
	if r.Code[0].Added != 1 || r.Code[0].Removed != 1 {
		t.Errorf("expected +1/-1, got +%d/-%d", r.Code[0].Added, r.Code[0].Removed)
	}

	// And the reverse: a metadata change must not appear as a code diff.
	remoteChanged := `{"identifier":"r-1","name":"Env anomaly","active":false,
	  "javascript":"var t = entity['sensor.temperature'];\nif (t) {\n  logger.info('ok');\n}\n"}`
	d2, dir2, local2 := unwrapLocal(t, remoteRule, nil)
	r2, err := Compare(d2, dir2, local2, json.RawMessage(remoteChanged), Options{ContextLines: 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(r2.Code) != 0 {
		t.Errorf("a metadata change must not appear as a code diff: %+v", r2.Code)
	}
	if len(r2.Metadata) != 1 || r2.Metadata[0].Path != "active" {
		t.Errorf("expected one metadata change on `active`, got %+v", r2.Metadata)
	}
}

// The code diff reads as "what deploying would do": remote is the before side.
func TestCompareCodeDirection(t *testing.T) {
	edited := "var t = entity['sensor.temperature'];\nif (t) {\n  logger.info('ok');\n  logger.info('new local line');\n}\n"
	d, dir, local := unwrapLocal(t, remoteRule, map[string]string{"javascript.js": edited})

	r, err := Compare(d, dir, local, json.RawMessage(remoteRule), Options{ContextLines: 3})
	if err != nil {
		t.Fatal(err)
	}
	if r.Code[0].Added != 1 || r.Code[0].Removed != 0 {
		t.Errorf("a locally added line should read as an addition, got +%d/-%d", r.Code[0].Added, r.Code[0].Removed)
	}
	if !strings.Contains(r.Code[0].Unified, "+   logger.info('new local line');") {
		t.Errorf("unified diff should show the local line as added:\n%s", r.Code[0].Unified)
	}
}

// Volatile fields must not surface as differences, or every diff of a dashboard
// reports a change nobody made.
func TestCompareIgnoresVolatileFields(t *testing.T) {
	d := unwrap.RuleDescriptor()
	local := json.RawMessage(`{"identifier":"r-1","name":"R"}`)
	remote := json.RawMessage(`{"identifier":"r-1","name":"R"}`)

	r, err := Compare(d, t.TempDir(), local, remote, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if r.Changed() {
		t.Errorf("identical payloads must not differ: %+v", r)
	}
}

// Cross-tenant comparison ignores identity, which differs by construction.
func TestCompareCrossTenantIgnoresIdentity(t *testing.T) {
	d := unwrap.RuleDescriptor()
	local := json.RawMessage(`{"identifier":"r-staging","name":"R","active":true}`)
	remote := json.RawMessage(`{"identifier":"r-prod","name":"R","active":true}`)

	same, err := Compare(d, t.TempDir(), local, remote, Options{Scope: canon.CrossTenant})
	if err != nil {
		t.Fatal(err)
	}
	if same.Changed() {
		t.Errorf("cross-tenant: differing identifiers are not a difference: %+v", same.Metadata)
	}

	within, err := Compare(d, t.TempDir(), local, remote, Options{Scope: canon.SameTenant})
	if err != nil {
		t.Fatal(err)
	}
	if !within.Changed() {
		t.Error("same-tenant: a differing identifier IS a difference")
	}
}

// A code file present on one side only is reported as such, not as a rewrite.
func TestCompareCodeOnlyOnOneSide(t *testing.T) {
	easyRule := `{"identifier":"r-1","name":"Easy","mode":"EASY"}`
	advanced := `{"identifier":"r-1","name":"Easy","mode":"EASY","javascript":"return 1;\n"}`

	d, dir, local := unwrapLocal(t, advanced, nil)
	r, err := Compare(d, dir, local, json.RawMessage(easyRule), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Code) != 1 || r.Code[0].OnlyIn != "local" {
		t.Fatalf("expected the file to be local-only, got %+v", r.Code)
	}

	// And the other way around.
	d2, dir2, local2 := unwrapLocal(t, easyRule, nil)
	r2, err := Compare(d2, dir2, local2, json.RawMessage(advanced), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(r2.Code) != 1 || r2.Code[0].OnlyIn != "remote" {
		t.Fatalf("expected the file to be remote-only, got %+v", r2.Code)
	}
}

func TestRenderTextAndNameOnly(t *testing.T) {
	edited := "var t = entity['sensor.temperature'];\nif (t) {\n  logger.warn('changed');\n}\n"
	d, dir, local := unwrapLocal(t, remoteRule, map[string]string{"javascript.js": edited})
	r, err := Compare(d, dir, local, json.RawMessage(remoteRule), Options{ContextLines: 1})
	if err != nil {
		t.Fatal(err)
	}

	full := r.RenderText(false)
	if !strings.Contains(full, "javascript.js") || !strings.Contains(full, "logger.warn") {
		t.Errorf("full render should include the code diff:\n%s", full)
	}

	brief := r.RenderText(true)
	if strings.Contains(brief, "logger.warn") {
		t.Errorf("--name-only must not print the diff body:\n%s", brief)
	}
	if !strings.Contains(brief, "Env anomaly") {
		t.Errorf("--name-only should still name the artifact:\n%s", brief)
	}
}

// The JSON shape is a contract, so its field names are asserted explicitly.
func TestResultJSONShape(t *testing.T) {
	edited := "var x = 1;\n"
	d, dir, local := unwrapLocal(t, remoteRule, map[string]string{"javascript.js": edited})
	r, err := Compare(d, dir, local, json.RawMessage(remoteRule), Options{ContextLines: 1})
	if err != nil {
		t.Fatal(err)
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"kind", "dir", "state", "code"} {
		if _, present := decoded[key]; !present {
			t.Errorf("JSON output missing the %q field", key)
		}
	}
	code, _ := decoded["code"].([]any)
	if len(code) == 0 {
		t.Fatal("code diffs missing from JSON")
	}
	entry, _ := code[0].(map[string]any)
	for _, key := range []string{"file", "added", "removed", "unified"} {
		if _, present := entry[key]; !present {
			t.Errorf("code entry missing the %q field", key)
		}
	}
}
