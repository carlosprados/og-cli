package views

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosprados/og-cli/v2/pkg/query"
)

func builtinRegistry(t *testing.T) *Registry {
	t.Helper()
	defs, err := parseFile(builtinYAML, SourceBuiltin)
	if err != nil {
		t.Fatalf("parsing builtin views: %v", err)
	}
	return &Registry{views: defs}
}

func TestBuiltinViewsParse(t *testing.T) {
	r := builtinRegistry(t)

	for _, want := range []string{"summary", "power", "location", "status", "identifier"} {
		if _, ok := r.Get(want); !ok {
			t.Errorf("builtin view %q missing", want)
		}
	}

	summary, _ := r.Get("summary")
	if len(summary.Fields) != 8 {
		t.Errorf("summary: expected 8 fields, got %d", len(summary.Fields))
	}
	last := summary.Fields[len(summary.Fields)-1]
	if last.Name != "device.operationalStatus" || !last.At {
		t.Errorf("summary last field = %+v, want device.operationalStatus with at", last)
	}
}

func TestParseFileShorthandAndLongForm(t *testing.T) {
	data := []byte(`
views:
  water:
    description: Water sensor readings
    fields:
      - wt@at
      - wp
      - name: device.temperature.value
        at: true
        alias: temp
`)
	defs, err := parseFile(data, "test.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	v := defs["water"]
	want := []Field{
		{Name: "wt", At: true},
		{Name: "wp"},
		{Name: "device.temperature.value", At: true, Alias: "temp"},
	}
	for i, f := range v.Fields {
		if f != want[i] {
			t.Errorf("field[%d] = %+v, want %+v", i, f, want[i])
		}
	}
	if v.Source != "test.yaml" {
		t.Errorf("source = %q", v.Source)
	}
}

func TestParseFileErrors(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{"no views key", "foo: bar", "no views defined"},
		{"empty fields", "views:\n  x:\n    fields: []", "has no fields"},
		{"missing name in long form", "views:\n  x:\n    fields:\n      - at: true", "missing 'name'"},
		{"invalid yaml", "views: [", "parsing"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseFile([]byte(tt.data), "test.yaml")
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("expected error containing %q, got %v", tt.want, err)
			}
		})
	}
}

func TestClausesAliases(t *testing.T) {
	d := Definition{Fields: []Field{
		{Name: "device.powersupply.battery.charge", At: true},
		{Name: "device.temperature.value", At: true, Alias: "temp"},
	}}
	got, _ := json.Marshal(d.Clauses())
	want := `[{"name":"device.powersupply.battery.charge","fields":[{"field":"value","alias":"charge"},{"field":"at","alias":"charge_at"}]},` +
		`{"name":"device.temperature.value","fields":[{"field":"value","alias":"temp"},{"field":"at","alias":"temp_at"}]}]`
	if string(got) != want {
		t.Errorf("Clauses() = %s\nwant %s", got, want)
	}
}

func TestResolveSelect(t *testing.T) {
	r := builtinRegistry(t)

	tests := []struct {
		name      string
		views     []string
		explicit  []query.SelectClause
		wantNames []string // expected clause names, in order
		wantErr   string
	}{
		{
			name:      "single view",
			views:     []string{"location"},
			wantNames: []string{"provision.device.location", "provision.asset.location"},
		},
		{
			name:     "explicit wins and comes first",
			views:    []string{"organization"},
			explicit: query.SelectFromFields([]string{"wt"}, false),
			wantNames: []string{
				"wt",
				"provision.administration.organization",
			},
		},
		{
			name:     "dedupe across view and explicit",
			views:    []string{"organization"},
			explicit: query.SelectFromFields([]string{"provision.administration.organization@at"}, false),
			wantNames: []string{
				"provision.administration.organization",
			},
		},
		{
			name:      "dedupe across views",
			views:     []string{"summary", "identifier"},
			wantNames: nil, // checked separately below
		},
		{
			name:    "unknown view with suggestion",
			views:   []string{"sumary"},
			wantErr: `did you mean "summary"`,
		},
		{
			name:    "unknown view without suggestion",
			views:   []string{"zzzzzzz"},
			wantErr: "unknown view",
		},
		{
			name:      "no views no explicit",
			wantNames: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.ResolveSelect(tt.views, tt.explicit)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantNames == nil {
				return
			}
			if len(got) != len(tt.wantNames) {
				t.Fatalf("got %d clauses, want %d: %+v", len(got), len(tt.wantNames), got)
			}
			for i, name := range tt.wantNames {
				if got[i].Name != name {
					t.Errorf("clause[%d].Name = %q, want %q", i, got[i].Name, name)
				}
			}
		})
	}
}

func TestResolveSelectDedupeAcrossViews(t *testing.T) {
	r := builtinRegistry(t)
	got, err := r.ResolveSelect([]string{"summary", "identifier"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	seen := make(map[string]bool)
	for _, c := range got {
		if seen[c.Name] {
			t.Errorf("duplicated clause %q", c.Name)
		}
		seen[c.Name] = true
	}
	// summary already projects provision.device.identifier; identifier view
	// must only add the ones summary lacks.
	if !seen["provision.administration.identifier"] || !seen["device.identifier"] {
		t.Errorf("identifier view fields missing: %v", got)
	}
}

func TestExplicitPrecedenceOverView(t *testing.T) {
	r := builtinRegistry(t)
	// summary projects device.operationalStatus with at; an explicit plain
	// select for the same datastream must win (value only).
	explicit := query.SelectFromFields([]string{"device.operationalStatus"}, false)
	got, err := r.ResolveSelect([]string{"summary"}, explicit)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, c := range got {
		if c.Name == "device.operationalStatus" {
			if len(c.Fields) != 1 || c.Fields[0].Field != "value" {
				t.Errorf("explicit clause overridden by view: %+v", c)
			}
			return
		}
	}
	t.Fatal("device.operationalStatus clause not found")
}

func TestLoadDirLayering(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("a.yaml", "views:\n  water:\n    fields: [wt@at]")
	write("b.yaml", "views:\n  fleet:\n    fields: [provision.device.identifier]")
	write("ignored.txt", "not yaml")

	layer, err := loadDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(layer) != 2 {
		t.Fatalf("expected 2 views, got %d", len(layer))
	}
	if layer["water"].Fields[0] != (Field{Name: "wt", At: true}) {
		t.Errorf("water field = %+v", layer["water"].Fields[0])
	}

	// Same view in two files of the same layer → error
	write("c.yaml", "views:\n  water:\n    fields: [other]")
	if _, err := loadDir(dir); err == nil || !strings.Contains(err.Error(), "defined twice") {
		t.Errorf("expected collision error, got %v", err)
	}
}

func TestLoadDirMissingIsOK(t *testing.T) {
	layer, err := loadDir(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("missing dir must not error, got %v", err)
	}
	if layer != nil {
		t.Errorf("expected nil layer, got %v", layer)
	}
}
