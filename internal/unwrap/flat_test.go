package unwrap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Duplicate names are legal on the platform — the identifier is the key — so
// pull-all must give same-named artifacts distinct directories. Sharing one
// Options across the batch is what makes that work; a fresh map per artifact
// silently collapses them onto the same path.
func TestSharedOptionsSeparatesSameNamedArtifacts(t *testing.T) {
	const a = `{"name":"weather","identifier":"cf-aaaaaaaaaaaa","javascript":"function f(){return 1;}"}`
	const b = `{"name":"weather","identifier":"cf-bbbbbbbbbbbb","javascript":"function f(){return 2;}"}`

	dir := t.TempDir()
	opts := &Options{}

	first, err := UnwrapConnectorFunction(json.RawMessage(a), dir, opts)
	if err != nil {
		t.Fatalf("first unwrap: %v", err)
	}
	second, err := UnwrapConnectorFunction(json.RawMessage(b), dir, opts)
	if err != nil {
		t.Fatalf("second unwrap: %v", err)
	}

	if first == second {
		t.Fatalf("both artifacts landed in the same directory: %s", first)
	}

	// Neither may have been overwritten: each keeps its own code.
	for dirPath, want := range map[string]string{
		first:  "function f(){return 1;}",
		second: "function f(){return 2;}",
	} {
		got, err := os.ReadFile(filepath.Join(dirPath, "javascript.js"))
		if err != nil {
			t.Fatalf("reading %s: %v", dirPath, err)
		}
		if string(got) != want {
			t.Errorf("%s: code = %q, want %q", dirPath, got, want)
		}
	}
}

// The same guarantee for the other two flat families, which share the code path.
func TestSharedOptionsSeparatesSameNamedRulesAndProvisions(t *testing.T) {
	t.Run("rules", func(t *testing.T) {
		dir := t.TempDir()
		opts := &Options{}
		a, err := UnwrapRule(json.RawMessage(`{"name":"dup","identifier":"r-111111111111"}`), dir, opts)
		if err != nil {
			t.Fatal(err)
		}
		b, err := UnwrapRule(json.RawMessage(`{"name":"dup","identifier":"r-222222222222"}`), dir, opts)
		if err != nil {
			t.Fatal(err)
		}
		if a == b {
			t.Errorf("same directory for both rules: %s", a)
		}
	})

	t.Run("provision functions", func(t *testing.T) {
		dir := t.TempDir()
		opts := &Options{}
		a, err := UnwrapProvisionProcessor(json.RawMessage(`{"name":"dup","provisionProcessorId":"p-111111111111"}`), dir, opts)
		if err != nil {
			t.Fatal(err)
		}
		b, err := UnwrapProvisionProcessor(json.RawMessage(`{"name":"dup","provisionProcessorId":"p-222222222222"}`), dir, opts)
		if err != nil {
			t.Fatal(err)
		}
		if a == b {
			t.Errorf("same directory for both provision functions: %s", a)
		}
	})
}

// A destination left by an earlier run is refused unless Force is set.
func TestOptionsForceGuardsExistingDestination(t *testing.T) {
	const cf = `{"name":"weather","identifier":"cf-aaaaaaaaaaaa","javascript":"function f(){return 1;}"}`
	dir := t.TempDir()

	if _, err := UnwrapConnectorFunction(json.RawMessage(cf), dir, &Options{}); err != nil {
		t.Fatalf("first unwrap: %v", err)
	}

	// A second run with a fresh Options hits the directory on disk.
	if _, err := UnwrapConnectorFunction(json.RawMessage(cf), dir, &Options{}); err == nil {
		t.Error("expected an error for an existing destination without Force")
	}

	if _, err := UnwrapConnectorFunction(json.RawMessage(cf), dir, &Options{Force: true}); err != nil {
		t.Errorf("Force should overwrite: %v", err)
	}
}

// A nil Taken map must not panic — callers doing a single pull pass &Options{}.
func TestOptionsNilTakenMap(t *testing.T) {
	dir := t.TempDir()
	if _, err := UnwrapRule(json.RawMessage(`{"name":"solo","identifier":"r-1"}`), dir, &Options{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
