package client

import (
	"encoding/json"
	"testing"
)

// TestUnmarshalDashboardGet validates that a real /api/dashboards/{id} response
// unmarshals into Dashboard with grid and widget definitions populated.
func TestUnmarshalDashboardGet(t *testing.T) {
	data := loadFixture(t, "dashboard_get.json")

	var d Dashboard
	if err := json.Unmarshal(data, &d); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if d.ID == "" || d.Title == "" {
		t.Errorf("dashboard missing id/title: %+v", d)
	}
	if d.Workspaces == "" {
		t.Error("dashboard should reference a parent workspace (workspaces field)")
	}
	if len(d.Grid) == 0 {
		t.Error("expected grid items in dashboard fixture")
	}

	g := d.Grid[0]
	if g.Definition == nil {
		t.Fatal("grid item missing widget definition")
	}
	if g.Definition.Type == "" {
		t.Error("widget definition missing Type")
	}
}

func TestApplyWorkspaceOverride_NoOverride(t *testing.T) {
	body := json.RawMessage(`{"_id":"d1","title":"Test","workspaces":"original"}`)
	got, err := applyWorkspaceOverride(body, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != string(body) {
		t.Errorf("expected payload to be untouched, got %s", string(got))
	}
}

func TestApplyWorkspaceOverride_ReplacesWorkspace(t *testing.T) {
	body := json.RawMessage(`{"_id":"d1","title":"Test","workspaces":"original"}`)
	got, err := applyWorkspaceOverride(body, "new-ws")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}

	if parsed["workspaces"] != "new-ws" {
		t.Errorf("expected workspaces=new-ws, got %v", parsed["workspaces"])
	}
	if parsed["_id"] != "d1" {
		t.Errorf("expected _id preserved, got %v", parsed["_id"])
	}
	if parsed["title"] != "Test" {
		t.Errorf("expected title preserved, got %v", parsed["title"])
	}
}

func TestApplyWorkspaceOverride_AddsWorkspaceWhenMissing(t *testing.T) {
	body := json.RawMessage(`{"_id":"d1","title":"Test"}`)
	got, err := applyWorkspaceOverride(body, "new-ws")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}

	if parsed["workspaces"] != "new-ws" {
		t.Errorf("expected workspaces=new-ws added, got %v", parsed["workspaces"])
	}
}

func TestApplyWorkspaceOverride_InvalidJSON(t *testing.T) {
	body := json.RawMessage(`not json`)
	_, err := applyWorkspaceOverride(body, "new-ws")
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestApplyWorkspaceOverride_PreservesNestedFields(t *testing.T) {
	body := json.RawMessage(`{"_id":"d1","grid":[{"x":0,"y":0,"definition":{"type":"widget1"}}],"workspaces":"original"}`)
	got, err := applyWorkspaceOverride(body, "new-ws")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}

	gridStr := string(parsed["grid"])
	expectedGrid := `[{"x":0,"y":0,"definition":{"type":"widget1"}}]`
	if gridStr != expectedGrid {
		t.Errorf("grid field not preserved verbatim:\n  got:  %s\n  want: %s", gridStr, expectedGrid)
	}
}
