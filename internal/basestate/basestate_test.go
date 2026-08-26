package basestate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/carlosprados/og-cli/v2/internal/unwrap"
)

const rulePayload = `{"identifier":"r-1","name":"Env anomaly","active":true,"javascript":"return 1;"}`

func TestRecordAndLookup(t *testing.T) {
	root := t.TempDir()
	artifactDir := filepath.Join(root, "env-anomaly")
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		t.Fatal(err)
	}

	store := Open(root)
	target := Target{Profile: "staging", Host: "https://staging.example", Org: "acme", Channel: "default_channel"}
	if err := store.Record(unwrap.KindRule, "r-1", "Env anomaly", artifactDir, json.RawMessage(rulePayload), target); err != nil {
		t.Fatalf("record: %v", err)
	}

	// The snapshot and the index must both be on disk.
	if _, err := os.Stat(filepath.Join(root, DirName, "manifest.json")); err != nil {
		t.Errorf("manifest missing: %v", err)
	}

	entry, ok := store.Lookup(unwrap.KindRule, "r-1")
	if !ok {
		t.Fatal("entry not found by kind and id")
	}
	if entry.Org != "acme" || entry.Channel != "default_channel" || entry.Profile != "staging" {
		t.Errorf("provenance not recorded: %+v", entry)
	}
	if entry.Hash == "" || entry.BaseFile == "" {
		t.Error("hash and base file must be recorded")
	}
	if entry.PulledAt.IsZero() {
		t.Error("pull time must be recorded")
	}

	// And by directory, which is what a deploy has in hand.
	byDir, ok := store.LookupByDir(artifactDir)
	if !ok || byDir.ID != "r-1" {
		t.Errorf("lookup by directory failed: %+v", byDir)
	}

	base, err := store.ReadBase(entry)
	if err != nil {
		t.Fatal(err)
	}
	if len(base) == 0 {
		t.Error("base snapshot is empty")
	}
}

// Find walks up from an artifact directory the way git finds its repository.
func TestFindWalksUp(t *testing.T) {
	root := t.TempDir()
	deep := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, ok := Find(deep); ok {
		t.Fatal("no store exists yet, Find must report that")
	}

	if err := os.MkdirAll(filepath.Join(root, DirName), 0o755); err != nil {
		t.Fatal(err)
	}
	store, ok := Find(deep)
	if !ok {
		t.Fatal("store not found from a nested directory")
	}
	wantRoot, _ := filepath.Abs(root)
	gotRoot, _ := filepath.Abs(store.Root)
	if gotRoot != wantRoot {
		t.Errorf("root = %q, want %q", gotRoot, wantRoot)
	}
}

// The whole point of the base snapshot: four distinguishable outcomes.
func TestClassifyTable(t *testing.T) {
	base := json.RawMessage(`{"identifier":"r-1","name":"R","active":true}`)
	edited := json.RawMessage(`{"identifier":"r-1","name":"R","active":false}`)
	otherEdit := json.RawMessage(`{"identifier":"r-1","name":"R","active":true,"description":"theirs"}`)

	canonBase := canonOf(t, base)

	cases := []struct {
		name          string
		local, remote json.RawMessage
		want          State
		marker        string
		safe          bool
	}{
		{"clean", base, base, Clean, " ", true},
		{"local changes", edited, base, LocalChanges, "~", true},
		{"remote changes", base, otherEdit, RemoteChanges, "↓", false},
		{"conflict", edited, otherEdit, Conflict, "!", false},
		// Both sides made the SAME edit: not a conflict worth blocking.
		{"converged", edited, edited, Clean, " ", true},
	}

	for _, c := range cases {
		got, err := Classify(unwrap.KindRule, c.local, c.remote, canonBase)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got.State != c.want {
			t.Errorf("%s: state = %v, want %v", c.name, got.State, c.want)
		}
		if got.State.Marker() != c.marker {
			t.Errorf("%s: marker = %q, want %q", c.name, got.State.Marker(), c.marker)
		}
		if got.State.SafeToDeploy() != c.safe {
			t.Errorf("%s: safe to deploy = %v, want %v", c.name, got.State.SafeToDeploy(), c.safe)
		}
	}
}

// Without a base nothing can be attributed, and that must be visible rather
// than reported as "clean".
func TestClassifyWithoutBase(t *testing.T) {
	a := json.RawMessage(`{"identifier":"r-1","active":true}`)
	b := json.RawMessage(`{"identifier":"r-1","active":false}`)

	got, err := Classify(unwrap.KindRule, a, b, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != Unknown {
		t.Errorf("state = %v, want Unknown", got.State)
	}
	if got.State.Marker() != "?" {
		t.Errorf("marker = %q, want ?", got.State.Marker())
	}
	// Unknown must not block a deploy: trees pulled before the store existed
	// are in this state, and refusing them would break the existing workflow.
	if !got.State.SafeToDeploy() {
		t.Error("Unknown must not block a deploy")
	}
}

// Volatile fields must not turn into phantom remote changes. This is the case
// that would have made the feature useless: the server bumps __v on every save.
func TestClassifyIgnoresVolatileFields(t *testing.T) {
	base := json.RawMessage(`{"id":"d1","title":"T","__v":26,"lastAccess":"2025-11-21T13:11:21.879Z"}`)
	remoteLater := json.RawMessage(`{"id":"d1","title":"T","__v":41,"lastAccess":"2026-08-26T09:00:00.000Z"}`)

	got, err := Classify(unwrap.KindDashboard, base, remoteLater, canonOf(t, base))
	if err != nil {
		t.Fatal(err)
	}
	if got.State != Clean {
		t.Errorf("state = %v, want Clean — a bumped __v is not a change", got.State)
	}
}

// MovedTo is what closes the staging→production footgun.
func TestMovedTo(t *testing.T) {
	e := Entry{Host: "https://staging.example", Profile: "staging", Org: "acme", Channel: "default_channel"}

	if diffs := e.MovedTo(Target{Host: "https://staging.example", Profile: "staging", Org: "acme", Channel: "default_channel"}); len(diffs) != 0 {
		t.Errorf("same target must report nothing, got %v", diffs)
	}

	diffs := e.MovedTo(Target{Host: "https://prod.example", Profile: "prod", Org: "acme", Channel: "default_channel"})
	if len(diffs) != 2 {
		t.Fatalf("expected host and profile to differ, got %v", diffs)
	}

	// An unrecorded field must not raise a false alarm: entries written before
	// a field existed have it empty.
	partial := Entry{Org: "acme"}
	if diffs := partial.MovedTo(Target{Host: "https://prod.example", Org: "acme"}); len(diffs) != 0 {
		t.Errorf("empty recorded fields must not be compared, got %v", diffs)
	}
}

func TestDescribe(t *testing.T) {
	e := Entry{Org: "acme", Channel: "default_channel", Profile: "staging", Host: "https://staging.example"}
	got := e.Describe()
	for _, want := range []string{"org acme", "channel default_channel", "profile staging"} {
		if !contains(got, want) {
			t.Errorf("%q should mention %q", got, want)
		}
	}
	if (Entry{}).Describe() != "unknown origin" {
		t.Error("an empty entry should say the origin is unknown")
	}
}

// A rule and a connector function may share an identifier; the store must not
// alias them onto each other.
func TestEntriesAreNamespacedByKind(t *testing.T) {
	root := t.TempDir()
	store := Open(root)
	for _, kind := range []unwrap.Kind{unwrap.KindRule, unwrap.KindConnectorFunction} {
		dir := filepath.Join(root, string(kind))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := store.Record(kind, "shared-id", "X", dir, json.RawMessage(`{"identifier":"shared-id"}`), Target{}); err != nil {
			t.Fatal(err)
		}
	}
	m, err := store.LoadManifest()
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Entries) != 2 {
		t.Errorf("entries = %d, want 2 — kinds must be namespaced", len(m.Entries))
	}
}

func TestRecordRejectsMissingID(t *testing.T) {
	root := t.TempDir()
	if err := Open(root).Record(unwrap.KindRule, "", "X", root, json.RawMessage(`{}`), Target{}); err == nil {
		t.Error("expected an error when there is no identifier to key on")
	}
}

func TestManifestSurvivesReopen(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "a")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Open(root).Record(unwrap.KindRule, "r-1", "R", dir, json.RawMessage(rulePayload), Target{Org: "acme"}); err != nil {
		t.Fatal(err)
	}
	if err := Open(root).Record(unwrap.KindRule, "r-2", "R2", dir, json.RawMessage(rulePayload), Target{Org: "acme"}); err != nil {
		t.Fatal(err)
	}
	m, err := Open(root).LoadManifest()
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Entries) != 2 {
		t.Errorf("a second record must not truncate the manifest: %d entries", len(m.Entries))
	}
	if m.SchemaVersion != manifestVersion {
		t.Errorf("schema version = %d, want %d", m.SchemaVersion, manifestVersion)
	}
}

func canonOf(t *testing.T, payload json.RawMessage) []byte {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "x")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	store := Open(root)
	kind := unwrap.KindRule
	var probe map[string]any
	if json.Unmarshal(payload, &probe) == nil {
		if _, ok := probe["title"]; ok {
			kind = unwrap.KindDashboard
		}
	}
	id := "probe"
	if err := store.Record(kind, id, "", dir, payload, Target{}); err != nil {
		t.Fatal(err)
	}
	e, _ := store.Lookup(kind, id)
	base, err := store.ReadBase(e)
	if err != nil {
		t.Fatal(err)
	}
	return base
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
