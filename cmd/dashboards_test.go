package cmd

import (
	"testing"

	"github.com/carlosprados/og-cli/internal/client"
)

func TestCollectDashboardRows_FullDashboards(t *testing.T) {
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
			{
				ID: "d2",
				Dashboard: &client.DashboardSimplified{
					ID:    "d2",
					Title: "Dash 2",
					Owner: "bob@example.com",
				},
			},
		},
	}

	rows := collectDashboardRows(w)
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	if rows[0].WorkspaceID != "ws1" || rows[0].WorkspaceName != "WS One" {
		t.Errorf("row 0 workspace fields wrong: %+v", rows[0])
	}
	if rows[0].DashboardID != "d1" || rows[0].Title != "Dash 1" || rows[0].Owner != "alice@example.com" {
		t.Errorf("row 0 dashboard fields wrong: %+v", rows[0])
	}
	if rows[1].DashboardID != "d2" || rows[1].Title != "Dash 2" {
		t.Errorf("row 1 dashboard fields wrong: %+v", rows[1])
	}
}

func TestCollectDashboardRows_NilDashboard(t *testing.T) {
	w := &client.Workspace{
		ID:   "ws1",
		Name: "WS One",
		Dashboards: []client.WorkspaceDashboard{
			{ID: "orphan-id", Dashboard: nil},
		},
	}

	rows := collectDashboardRows(w)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	if rows[0].DashboardID != "orphan-id" {
		t.Errorf("expected fallback to WorkspaceDashboard.ID, got %q", rows[0].DashboardID)
	}
	if rows[0].Title != "" || rows[0].Owner != "" {
		t.Errorf("expected empty title/owner when Dashboard is nil, got %+v", rows[0])
	}
}

func TestCollectDashboardRows_EmptyWorkspace(t *testing.T) {
	w := &client.Workspace{ID: "ws-empty", Name: "Empty"}
	rows := collectDashboardRows(w)
	if len(rows) != 0 {
		t.Errorf("expected 0 rows for empty workspace, got %d", len(rows))
	}
}

func TestCollectDashboardRows_PrefersDashboardIDOverOuterID(t *testing.T) {
	w := &client.Workspace{
		ID: "ws1",
		Dashboards: []client.WorkspaceDashboard{
			{
				ID: "outer-id",
				Dashboard: &client.DashboardSimplified{
					ID:    "inner-id",
					Title: "T",
				},
			},
		},
	}

	rows := collectDashboardRows(w)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].DashboardID != "inner-id" {
		t.Errorf("expected dashboard.ID to take precedence, got %q", rows[0].DashboardID)
	}
}
