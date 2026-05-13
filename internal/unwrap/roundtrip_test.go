package unwrap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/carlosprados/og-cli/internal/client"
)

// TestRoundtrip_DashboardFull verifies that UnwrapDashboardFull → WrapDashboard
// produces a JSON-equivalent dashboard, using a real /api/dashboards/{id}
// response as fixture.
func TestRoundtrip_DashboardFull(t *testing.T) {
	fixture := filepath.Join("..", "client", "testdata", "dashboard_get.json")
	raw, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	var d client.Dashboard
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	if len(d.Grid) == 0 {
		t.Fatal("fixture has no grid items — cannot roundtrip widgets")
	}

	tmp := t.TempDir()
	dashDir := filepath.Join(tmp, "dash")

	if err := UnwrapDashboardFull(&d, nil, dashDir); err != nil {
		t.Fatalf("UnwrapDashboardFull: %v", err)
	}

	// dashboard.json must exist
	if _, err := os.Stat(filepath.Join(dashDir, "dashboard.json")); err != nil {
		t.Fatalf("dashboard.json missing: %v", err)
	}

	// Exactly len(d.Grid) widget directories must exist.
	entries, err := os.ReadDir(dashDir)
	if err != nil {
		t.Fatalf("reading dashDir: %v", err)
	}
	var widgetDirs int
	for _, e := range entries {
		if e.IsDir() {
			widgetDirs++
		}
	}
	if widgetDirs != len(d.Grid) {
		t.Errorf("got %d widget dirs, want %d", widgetDirs, len(d.Grid))
	}

	// Rebuild and compare.
	rebuilt, _, err := WrapDashboard(dashDir)
	if err != nil {
		t.Fatalf("WrapDashboard: %v", err)
	}

	if rebuilt.ID != d.ID {
		t.Errorf("ID mismatch: got %q want %q", rebuilt.ID, d.ID)
	}
	if rebuilt.Title != d.Title {
		t.Errorf("Title mismatch: got %q want %q", rebuilt.Title, d.Title)
	}
	if len(rebuilt.Grid) != len(d.Grid) {
		t.Fatalf("grid length mismatch: got %d want %d", len(rebuilt.Grid), len(d.Grid))
	}

	// Compare widget configs as decoded trees (order-independent) for each grid item.
	for i := range d.Grid {
		var origCfg, rebuiltCfg any
		_ = json.Unmarshal(d.Grid[i].Definition.Config, &origCfg)
		_ = json.Unmarshal(rebuilt.Grid[i].Definition.Config, &rebuiltCfg)
		if !reflect.DeepEqual(origCfg, rebuiltCfg) {
			t.Errorf("widget %d config mismatch", i)
		}

		if d.Grid[i].Definition.Type != rebuilt.Grid[i].Definition.Type {
			t.Errorf("widget %d type mismatch", i)
		}
	}
}

// TestRoundtrip_WithJSExtraction verifies the full unwrap → wrap cycle
// preserves a synthetic widget config containing JavaScript code.
func TestRoundtrip_WithJSExtraction(t *testing.T) {
	configJSON := `{
		"title": "Custom",
		"formatter": "function(v){return v.toFixed(2) + ' C';}",
		"columns": [
			{"path": "wt", "formatter": "function(v){return v + ' C';}"},
			{"path": "wp"}
		]
	}`

	d := &client.Dashboard{
		ID:    "synthetic",
		Title: "Synthetic",
		Grid: []client.GridItem{
			{
				X: 0, Y: 0, W: 4, H: 2, I: "w1",
				Definition: &client.WidgetDefinition{
					Type:   "TestWidget",
					Wid:    "1234-1",
					Config: json.RawMessage(configJSON),
				},
			},
		},
	}

	tmp := t.TempDir()
	dashDir := filepath.Join(tmp, "dash")
	if err := UnwrapDashboardFull(d, nil, dashDir); err != nil {
		t.Fatalf("UnwrapDashboardFull: %v", err)
	}

	// Verify formatter.js was created on disk.
	widgetDir := filepath.Join(dashDir, "00__testwidget__1234-1")
	if _, err := os.Stat(filepath.Join(widgetDir, "formatter.js")); err != nil {
		t.Errorf("formatter.js not extracted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(widgetDir, "columns__0__formatter.js")); err != nil {
		t.Errorf("columns__0__formatter.js not extracted: %v", err)
	}

	// Rebuild and compare configs by tree equality.
	rebuilt, _, err := WrapDashboard(dashDir)
	if err != nil {
		t.Fatalf("WrapDashboard: %v", err)
	}
	if len(rebuilt.Grid) != 1 {
		t.Fatalf("expected 1 grid item, got %d", len(rebuilt.Grid))
	}

	var origCfg, rebuiltCfg any
	_ = json.Unmarshal(d.Grid[0].Definition.Config, &origCfg)
	_ = json.Unmarshal(rebuilt.Grid[0].Definition.Config, &rebuiltCfg)
	if !reflect.DeepEqual(origCfg, rebuiltCfg) {
		t.Errorf("config roundtrip mismatch.\n  orig:    %v\n  rebuilt: %v", origCfg, rebuiltCfg)
	}
}
