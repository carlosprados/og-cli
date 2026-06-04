package mcp

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMcpBuildFilterSelectAt(t *testing.T) {
	filter, err := mcpBuildFilter(map[string]any{"select": "wt@at,wp"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var body struct {
		Select []struct {
			Name   string `json:"name"`
			Fields []struct {
				Field string `json:"field"`
				Alias string `json:"alias"`
			} `json:"fields"`
		} `json:"select"`
	}
	if err := json.Unmarshal(filter, &body); err != nil {
		t.Fatalf("unmarshaling filter: %v", err)
	}
	if len(body.Select) != 2 {
		t.Fatalf("expected 2 select clauses, got %d", len(body.Select))
	}
	wt := body.Select[0]
	if wt.Name != "wt" || len(wt.Fields) != 2 || wt.Fields[1].Field != "at" || wt.Fields[1].Alias != "wt_at" {
		t.Errorf("wt clause = %+v", wt)
	}
	wp := body.Select[1]
	if wp.Name != "wp" || len(wp.Fields) != 1 {
		t.Errorf("wp clause = %+v", wp)
	}
}

func TestMcpBuildFilterWithView(t *testing.T) {
	filter, err := mcpBuildFilter(map[string]any{"view": "organization", "select": "wt"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(filter)
	// Explicit select first, then the view expansion.
	if !strings.Contains(s, `"name":"wt"`) || !strings.Contains(s, `"name":"provision.administration.organization"`) {
		t.Errorf("filter missing expected clauses: %s", s)
	}
}

func TestMcpBuildFilterUnknownView(t *testing.T) {
	_, err := mcpBuildFilter(map[string]any{"view": "sumary"})
	if err == nil || !strings.Contains(err.Error(), "unknown view") {
		t.Errorf("expected unknown view error, got %v", err)
	}
}
