package unwrap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A malformed widget directory must fail the wrap. Skipping it produces a
// successful deploy with that widget missing from the dashboard — a silent
// partial write, which is worse than refusing.
func TestWrapDashboardFailsOnMalformedWidget(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "dashboard.json"), `{"id":"d1","title":"Dash"}`)

	// A widget directory whose widget.json is not valid JSON.
	bad := filepath.Join(dir, "00__testwidget__w1")
	if err := os.MkdirAll(bad, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(bad, "widget.json"), `{"i":"w1",`)

	_, _, err := WrapDashboard(dir, nil)
	if err == nil {
		t.Fatal("expected an error for a malformed widget directory, got nil")
	}
	if !strings.Contains(err.Error(), "00__testwidget__w1") {
		t.Errorf("error should name the offending widget, got: %v", err)
	}
}

// A widget directory with no widget.json at all is equally an error.
func TestWrapDashboardFailsOnMissingWidgetJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "dashboard.json"), `{"id":"d1","title":"Dash"}`)
	if err := os.MkdirAll(filepath.Join(dir, "00__orphan"), 0o755); err != nil {
		t.Fatal(err)
	}

	if _, _, err := WrapDashboard(dir, nil); err == nil {
		t.Error("expected an error for a widget directory without widget.json")
	}
}

// Dot-directories are metadata (the .og/ sync cache, .git, editor state) and
// must never be mistaken for artifact content.
func TestWrapIgnoresDotDirectories(t *testing.T) {
	t.Run("dashboard widgets", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "dashboard.json"), `{"id":"d1","title":"Dash"}`)

		w := filepath.Join(dir, "00__testwidget__w1")
		if err := os.MkdirAll(w, 0o755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(w, "widget.json"), `{"i":"w1","definition":{"type":"TestWidget","wid":"w1"}}`)

		// A sync cache alongside the widgets must be ignored, not wrapped.
		cache := filepath.Join(dir, ".og", "base")
		if err := os.MkdirAll(cache, 0o755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(cache, "d1.canon.json"), `{}`)

		full, _, err := WrapDashboard(dir, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(full.Grid) != 1 {
			t.Errorf("grid has %d items, want 1 (the .og/ cache must be skipped)", len(full.Grid))
		}
	})

	t.Run("workspace dashboards", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "workspace.json"), `{"id":"ws1","name":"WS"}`)

		d := filepath.Join(dir, "00__dash")
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(d, "dashboard.json"), `{"id":"d1","title":"Dash"}`)

		if err := os.MkdirAll(filepath.Join(dir, ".og"), 0o755); err != nil {
			t.Fatal(err)
		}

		ws, err := Wrap(dir, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(ws.Dashboards) != 1 {
			t.Errorf("workspace has %d dashboards, want 1 (the .og/ cache must be skipped)", len(ws.Dashboards))
		}
	})
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
