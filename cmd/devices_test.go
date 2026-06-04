package cmd

import (
	"encoding/json"
	"testing"
)

func TestBuildSearchFilterANDInsideWhere(t *testing.T) {
	// A single -w with "cond AND cond" must produce BOTH conditions,
	// matching the MCP query parameter semantics (regression: the second
	// condition used to be silently dropped).
	filter, err := buildSearchFilter([]string{"wt gte 10 AND wt lte 15"}, 0, nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var body struct {
		Filter struct {
			And []json.RawMessage `json:"and"`
		} `json:"filter"`
	}
	if err := json.Unmarshal(filter, &body); err != nil {
		t.Fatalf("unmarshaling filter: %v", err)
	}
	if len(body.Filter.And) != 2 {
		t.Fatalf("expected 2 AND clauses, got %d: %s", len(body.Filter.And), filter)
	}
}

func TestBuildSearchFilterMultipleWhere(t *testing.T) {
	filter, err := buildSearchFilter([]string{"wt gte 10", "wt lte 15"}, 0, nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var body struct {
		Filter struct {
			And []json.RawMessage `json:"and"`
		} `json:"filter"`
	}
	if err := json.Unmarshal(filter, &body); err != nil {
		t.Fatalf("unmarshaling filter: %v", err)
	}
	if len(body.Filter.And) != 2 {
		t.Fatalf("expected 2 AND clauses, got %d: %s", len(body.Filter.And), filter)
	}
}
