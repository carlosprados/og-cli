package unwrap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// The three flat families differ only in these literals. Asserting the table
// directly is cheaper than asserting it through three near-identical cycles.
func TestDescriptorTable(t *testing.T) {
	cases := []struct {
		d        Descriptor
		kind     Kind
		metaFile string
		payload  string
		wantName string
		wantID   string
	}{
		{
			d: RuleDescriptor(), kind: KindRule, metaFile: "rule.json",
			payload: `{"name":"Env anomaly","identifier":"r-1"}`, wantName: "Env anomaly", wantID: "r-1",
		},
		{
			d: ConnectorFunctionDescriptor(), kind: KindConnectorFunction, metaFile: "connectorfunction.json",
			payload: `{"name":"weather","identifier":"cf-1"}`, wantName: "weather", wantID: "cf-1",
		},
		{
			// The API echoes connectorFunctionName instead of name on some reads.
			d: ConnectorFunctionDescriptor(), kind: KindConnectorFunction, metaFile: "connectorfunction.json",
			payload: `{"connectorFunctionName":"weather","identifier":"cf-1"}`, wantName: "weather", wantID: "cf-1",
		},
		{
			d: ProvisionFunctionDescriptor(), kind: KindProvisionFunction, metaFile: "provisionfunction.json",
			payload: `{"name":"createUpdate","provisionProcessorId":"pp-1"}`, wantName: "createUpdate", wantID: "pp-1",
		},
	}

	for _, c := range cases {
		if c.d.Kind != c.kind {
			t.Errorf("kind = %q, want %q", c.d.Kind, c.kind)
		}
		if c.d.MetaFile != c.metaFile {
			t.Errorf("%s: meta file = %q, want %q", c.kind, c.d.MetaFile, c.metaFile)
		}
		if got := c.d.NameOf(json.RawMessage(c.payload)); got != c.wantName {
			t.Errorf("%s: name = %q, want %q", c.kind, got, c.wantName)
		}

		var node any
		if err := json.Unmarshal([]byte(c.payload), &node); err != nil {
			t.Fatal(err)
		}
		meta, _ := node.(map[string]any)
		if got := c.d.idOf(meta); got != c.wantID {
			t.Errorf("%s: id = %q, want %q", c.kind, got, c.wantID)
		}
	}
}

// A connector function's contract depends on the payload, because its
// execution context follows its type field.
func TestDescriptorContractDependsOnPayload(t *testing.T) {
	d := ConnectorFunctionDescriptor()
	for cfType, want := range map[string]ExecCtx{
		"REQUEST":    CtxCFRequest,
		"COLLECTION": CtxCFCollection,
		"":           CtxUnknown,
	} {
		raw := `{"identifier":"cf-1","name":"x","type":"` + cfType + `","javascript":"function f(){return 1;}"}`
		dir := t.TempDir()
		if _, err := d.Unwrap(json.RawMessage(raw), dir, &Options{}); err != nil {
			t.Fatal(err)
		}
		var node any
		if err := json.Unmarshal([]byte(raw), &node); err != nil {
			t.Fatal(err)
		}
		meta, _ := node.(map[string]any)
		_, _, ctxs := d.Contract(meta).Extract(node, nil)
		if got := ctxs["javascript.js"]; got != want {
			t.Errorf("type %q: context = %q, want %q", cfType, got, want)
		}
	}
}

// The generic cycle must round-trip every flat family.
func TestDescriptorRoundTrip(t *testing.T) {
	cases := map[string]struct {
		d       Descriptor
		payload string
		file    string
	}{
		"rule":               {RuleDescriptor(), `{"name":"r","identifier":"r-1","mode":"ADVANCED","javascript":"function f(){return 1;}"}`, "javascript.js"},
		"connector function": {ConnectorFunctionDescriptor(), `{"name":"c","identifier":"c-1","type":"REQUEST","javascript":"function f(){return 2;}"}`, "javascript.js"},
		"provision function": {ProvisionFunctionDescriptor(), `{"name":"p","provisionProcessorId":"p-1","scriptProcessor":{"script":"function normalizeRawObject(o){return o;}"}}`, "scriptProcessor__script.js"},
	}

	for label, c := range cases {
		dir := t.TempDir()
		artifactDir, err := c.d.Unwrap(json.RawMessage(c.payload), dir, &Options{})
		if err != nil {
			t.Fatalf("%s: unwrap: %v", label, err)
		}
		if _, err := os.Stat(filepath.Join(artifactDir, c.d.MetaFile)); err != nil {
			t.Errorf("%s: %s missing", label, c.d.MetaFile)
		}
		if _, err := os.Stat(filepath.Join(artifactDir, c.file)); err != nil {
			t.Errorf("%s: %s missing", label, c.file)
		}

		out, err := c.d.Wrap(artifactDir, nil)
		if err != nil {
			t.Fatalf("%s: wrap: %v", label, err)
		}
		var want, got any
		if err := json.Unmarshal([]byte(c.payload), &want); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(out, &got); err != nil {
			t.Fatal(err)
		}
		if !jsonEqual(want, got) {
			t.Errorf("%s: round trip changed the payload\n  want: %s\n  got:  %s", label, c.payload, out)
		}
	}
}

func jsonEqual(a, b any) bool {
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	return string(ja) == string(jb)
}

// An unnamed artifact still gets a directory rather than failing.
func TestDescriptorUnnamedArtifact(t *testing.T) {
	dir := t.TempDir()
	got, err := RuleDescriptor().Unwrap(json.RawMessage(`{"identifier":"r-1"}`), dir, &Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filepath.Base(got) != "unnamed" {
		t.Errorf("directory = %q, want the 'unnamed' fallback", filepath.Base(got))
	}
}
