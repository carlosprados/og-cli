package unwrap

import (
	"encoding/json"
	"testing"

	"github.com/carlosprados/og-cli/v2/pkg/opengate"
)

func widget(wtype, wid, config string) opengate.GridItem {
	return opengate.GridItem{
		I: wid,
		Definition: &opengate.WidgetDefinition{
			Type:   wtype,
			Wid:    wid,
			Config: json.RawMessage(config),
		},
	}
}

func demoDashboard() *opengate.Dashboard {
	return &opengate.Dashboard{
		ID:    "d1",
		Title: "Demo",
		Grid: []opengate.GridItem{
			widget("markdown", "intro", `{"text":"hello"}`),
			widget("customTable", "table", `{"_widgetConfigCode":"return 1;"}`),
			widget("FullDevicesList", "list", `{"columns":[{"_formatterCode":"return 'a';"},{"_formatterCode":"return 'b';"}]}`),
		},
	}
}

// The keys have to be the paths a pull writes, or an editor sending the path it
// has on disk addresses nothing.
func TestDashboardCodeFilesUsesThePullsOwnPaths(t *testing.T) {
	files, err := DashboardCodeFiles(demoDashboard())
	if err != nil {
		t.Fatalf("DashboardCodeFiles: %v", err)
	}

	want := map[string]string{
		"01__customtable__table/_widgetConfigCode.js":             "return 1;",
		"02__fulldeviceslist__list/columns__0___formatterCode.js": "return 'a';",
		"02__fulldeviceslist__list/columns__1___formatterCode.js": "return 'b';",
	}
	if len(files) != len(want) {
		t.Fatalf("got %d files, want %d: %v", len(files), len(want), sortedKeys(files))
	}
	for path, code := range want {
		got, ok := files[path]
		if !ok {
			t.Errorf("missing %s; got %v", path, sortedKeys(files))
			continue
		}
		if got != code {
			t.Errorf("%s = %q, want %q", path, got, code)
		}
	}

	// A widget with no code contributes nothing rather than an empty entry.
	for path := range files {
		if path == "00__markdown__intro" {
			t.Errorf("a widget with no code produced %s", path)
		}
	}
}

func TestDashboardCodeFilesRejectsNil(t *testing.T) {
	if _, err := DashboardCodeFiles(nil); err == nil {
		t.Error("expected an error for a nil dashboard")
	}
}

func TestDashboardCodeFilesReportsAMalformedConfig(t *testing.T) {
	d := &opengate.Dashboard{Grid: []opengate.GridItem{
		{I: "w", Definition: &opengate.WidgetDefinition{Type: "customTable", Wid: "w", Config: json.RawMessage(`not json`)}},
	}}
	if _, err := DashboardCodeFiles(d); err == nil {
		t.Error("expected an error for a config that is not JSON")
	}
}

// The index in a widget's folder name is the remote grid order at the time of
// the pull. A reorder on the platform must not make a local path unresolvable.
func TestResolveCodePathIgnoresTheGridIndex(t *testing.T) {
	files, err := DashboardCodeFiles(demoDashboard())
	if err != nil {
		t.Fatalf("DashboardCodeFiles: %v", err)
	}

	cases := []struct {
		name      string
		requested string
	}{
		{"exact", "01__customtable__table/_widgetConfigCode.js"},
		{"moved in the grid", "07__customtable__table/_widgetConfigCode.js"},
		{"no index at all", "customtable__table/_widgetConfigCode.js"},
		{"windows separators", `01__customtable__table\_widgetConfigCode.js`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ResolveCodePath(files, tc.requested)
			if !ok {
				t.Fatalf("%q did not resolve", tc.requested)
			}
			if got != "return 1;" {
				t.Errorf("content = %q", got)
			}
		})
	}
}

func TestResolveCodePathRejectsWhatItCannotName(t *testing.T) {
	files, err := DashboardCodeFiles(demoDashboard())
	if err != nil {
		t.Fatalf("DashboardCodeFiles: %v", err)
	}
	for _, requested := range []string{
		"01__customtable__table/nope.js",
		"01__somethingelse__x/_widgetConfigCode.js",
		"_widgetConfigCode.js", // no widget directory at all
		"",
	} {
		if _, ok := ResolveCodePath(files, requested); ok {
			t.Errorf("%q resolved and should not have", requested)
		}
	}
}

// Two widgets of the same type with no id of their own slug to the same name
// once the index is dropped. Serving one of them at random would be worse than
// saying the path could not be resolved.
func TestResolveCodePathRefusesAnAmbiguousMatch(t *testing.T) {
	d := &opengate.Dashboard{Grid: []opengate.GridItem{
		{Definition: &opengate.WidgetDefinition{Type: "customTable", Config: json.RawMessage(`{"_widgetConfigCode":"first"}`)}},
		{Definition: &opengate.WidgetDefinition{Type: "customTable", Config: json.RawMessage(`{"_widgetConfigCode":"second"}`)}},
	}}
	files, err := DashboardCodeFiles(d)
	if err != nil {
		t.Fatalf("DashboardCodeFiles: %v", err)
	}

	// The exact path still works: it names one folder and only one.
	if got, ok := ResolveCodePath(files, "00__customtable/_widgetConfigCode.js"); !ok || got != "first" {
		t.Errorf("exact path = %q, %v", got, ok)
	}
	// Stripped of its index it names both, so it names neither.
	if _, ok := ResolveCodePath(files, "99__customtable/_widgetConfigCode.js"); ok {
		t.Error("an ambiguous path resolved; it must not")
	}
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
