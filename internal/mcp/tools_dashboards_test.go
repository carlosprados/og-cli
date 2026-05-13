package mcp

import (
	"testing"

	"github.com/carlosprados/og-cli/internal/client"
)

func TestCollectDashEntries_FullDashboards(t *testing.T) {
	w := &client.Workspace{
		ID:   "ws1",
		Name: "WS One",
		Dashboards: []client.WorkspaceDashboard{
			{
				ID: "d1",
				Dashboard: &client.DashboardSimplified{
					ID:    "d1",
					Title: "Dash 1",
					Owner: "alice@example.com",
				},
			},
		},
	}

	entries := collectDashEntries(w)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	got := entries[0]
	if got.WorkspaceID != "ws1" || got.WorkspaceName != "WS One" {
		t.Errorf("workspace fields wrong: %+v", got)
	}
	if got.DashboardID != "d1" || got.Title != "Dash 1" || got.Owner != "alice@example.com" {
		t.Errorf("dashboard fields wrong: %+v", got)
	}
}

func TestCollectDashEntries_NilDashboard(t *testing.T) {
	w := &client.Workspace{
		ID:   "ws1",
		Name: "WS One",
		Dashboards: []client.WorkspaceDashboard{
			{ID: "orphan-id", Dashboard: nil},
		},
	}

	entries := collectDashEntries(w)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].DashboardID != "orphan-id" {
		t.Errorf("expected fallback to WorkspaceDashboard.ID, got %q", entries[0].DashboardID)
	}
	if entries[0].Title != "" || entries[0].Owner != "" {
		t.Errorf("expected empty title/owner when Dashboard is nil, got %+v", entries[0])
	}
}

func TestCollectDashEntries_EmptyWorkspace(t *testing.T) {
	w := &client.Workspace{ID: "ws-empty"}
	entries := collectDashEntries(w)
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestCollectDashEntries_MultipleDashboards(t *testing.T) {
	w := &client.Workspace{
		ID:   "ws1",
		Name: "WS",
		Dashboards: []client.WorkspaceDashboard{
			{ID: "d1", Dashboard: &client.DashboardSimplified{ID: "d1", Title: "A"}},
			{ID: "d2", Dashboard: &client.DashboardSimplified{ID: "d2", Title: "B"}},
			{ID: "d3", Dashboard: &client.DashboardSimplified{ID: "d3", Title: "C"}},
		},
	}

	entries := collectDashEntries(w)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	titles := []string{entries[0].Title, entries[1].Title, entries[2].Title}
	want := []string{"A", "B", "C"}
	for i := range titles {
		if titles[i] != want[i] {
			t.Errorf("entry %d title: got %q, want %q", i, titles[i], want[i])
		}
	}
}
