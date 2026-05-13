package client

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// loadFixture reads a JSON file under testdata/.
func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("loading fixture %s: %v", name, err)
	}
	return data
}

// TestUnmarshalWorkspacesList validates that a real /api/workspaces/ response
// (a JSON array) unmarshals into []Workspace without errors. This is the test
// that would have caught the "object vs array" shape mismatch.
func TestUnmarshalWorkspacesList(t *testing.T) {
	data := loadFixture(t, "workspaces_list.json")

	var wss []Workspace
	if err := json.Unmarshal(data, &wss); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(wss) == 0 {
		t.Fatal("expected at least one workspace in fixture")
	}

	first := wss[0]
	if first.ID == "" {
		t.Error("first workspace has empty ID")
	}
	if first.Name == "" {
		t.Error("first workspace has empty Name")
	}
}

// TestUnmarshalWorkspacesListFull verifies the ?full=1 response includes
// dashboards nested inside each workspace.
func TestUnmarshalWorkspacesListFull(t *testing.T) {
	data := loadFixture(t, "workspaces_list_full.json")

	var wss []Workspace
	if err := json.Unmarshal(data, &wss); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	var withDash *Workspace
	for i := range wss {
		if len(wss[i].Dashboards) > 0 {
			withDash = &wss[i]
			break
		}
	}
	if withDash == nil {
		t.Fatal("expected at least one workspace with embedded dashboards in fixture")
	}

	d := withDash.Dashboards[0]
	if d.Dashboard == nil {
		t.Fatal("expected embedded Dashboard struct, got nil")
	}
	if d.Dashboard.ID == "" {
		t.Error("embedded dashboard has empty ID")
	}
	if d.Dashboard.Title == "" {
		t.Error("embedded dashboard has empty Title")
	}
}

// TestUnmarshalWorkspaceGet verifies a single-workspace GET response.
func TestUnmarshalWorkspaceGet(t *testing.T) {
	data := loadFixture(t, "workspace_get.json")

	var w Workspace
	if err := json.Unmarshal(data, &w); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if w.ID == "" || w.Name == "" {
		t.Errorf("workspace missing id/name: %+v", w)
	}
}

// TestWorkspaceExportShape verifies the /workspaces/export/{id} response uses
// a {"workspaces":[...]} wrapper (different from the list endpoint).
func TestWorkspaceExportShape(t *testing.T) {
	data := loadFixture(t, "workspace_export.json")

	var wrapper struct {
		Workspaces []Workspace `json:"workspaces"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if len(wrapper.Workspaces) == 0 {
		t.Fatal("expected at least one workspace inside the export wrapper")
	}
	if wrapper.Workspaces[0].Name == "" {
		t.Errorf("exported workspace missing Name: %+v", wrapper.Workspaces[0])
	}
}
