package query

import (
	"encoding/json"
	"testing"
)

func TestParseCondition(t *testing.T) {
	tests := []struct {
		input string
		want  Condition
		err   bool
	}{
		{"provision.device.identifier eq sense-001", Condition{"provision.device.identifier", "eq", "sense-001"}, false},
		{"provision.device.identifier like sense", Condition{"provision.device.identifier", "like", "sense"}, false},
		{"provision.device.administrativeState neq BANNED", Condition{"provision.device.administrativeState", "neq", "BANNED"}, false},
		{"field exists", Condition{"field", "exists", "true"}, false},
		{"field gt 100", Condition{"field", "gt", "100"}, false},
		{"bad", Condition{}, true},
		{"field badop value", Condition{}, true},
	}

	for _, tt := range tests {
		c, err := ParseCondition(tt.input)
		if tt.err {
			if err == nil {
				t.Errorf("ParseCondition(%q) expected error", tt.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseCondition(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if c != tt.want {
			t.Errorf("ParseCondition(%q) = %+v, want %+v", tt.input, c, tt.want)
		}
	}
}

func TestParseQuery(t *testing.T) {
	conditions, err := ParseQuery("provision.device.identifier like sense AND provision.device.administrativeState eq ACTIVE")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conditions) != 2 {
		t.Fatalf("expected 2 conditions, got %d", len(conditions))
	}
	if conditions[0].Field != "provision.device.identifier" || conditions[0].Op != "like" {
		t.Errorf("condition[0] = %+v", conditions[0])
	}
	if conditions[1].Field != "provision.device.administrativeState" || conditions[1].Op != "eq" {
		t.Errorf("condition[1] = %+v", conditions[1])
	}
}

func TestParseQueryEmpty(t *testing.T) {
	conditions, err := ParseQuery("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conditions != nil {
		t.Errorf("expected nil, got %+v", conditions)
	}
}

func TestBuildFilterSingle(t *testing.T) {
	p := SearchParams{
		Conditions: []Condition{{Field: "provision.device.identifier", Op: "eq", Value: "sense-001"}},
	}
	data, err := BuildFilter(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	json.Unmarshal(data, &result)

	filter, ok := result["filter"].(map[string]any)
	if !ok {
		t.Fatalf("expected filter object, got %v", result)
	}
	eqClause, ok := filter["eq"].(map[string]any)
	if !ok {
		t.Fatalf("expected eq clause, got %v", filter)
	}
	if eqClause["provision.device.identifier"] != "sense-001" {
		t.Errorf("unexpected value: %v", eqClause)
	}
}

func TestBuildFilterMultiple(t *testing.T) {
	p := SearchParams{
		Conditions: []Condition{
			{Field: "a", Op: "eq", Value: "1"},
			{Field: "b", Op: "like", Value: "x"},
		},
		Limit: 10,
	}
	data, err := BuildFilter(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	json.Unmarshal(data, &result)

	filter := result["filter"].(map[string]any)
	andClauses, ok := filter["and"].([]any)
	if !ok {
		t.Fatalf("expected and array, got %v", filter)
	}
	if len(andClauses) != 2 {
		t.Fatalf("expected 2 clauses, got %d", len(andClauses))
	}

	limit := result["limit"].(map[string]any)
	if limit["size"] != float64(10) {
		t.Errorf("expected limit 10, got %v", limit["size"])
	}
}

func TestBuildFilterWithSelect(t *testing.T) {
	p := SearchParams{
		Select: []string{"provision.device.identifier", "wt"},
	}
	data, err := BuildFilter(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	json.Unmarshal(data, &result)

	sel, ok := result["select"].([]any)
	if !ok {
		t.Fatalf("expected select array, got %v", result)
	}
	if len(sel) != 2 {
		t.Fatalf("expected 2 select entries, got %d", len(sel))
	}

	first := sel[0].(map[string]any)
	if first["name"] != "provision.device.identifier" {
		t.Errorf("expected provision.device.identifier, got %v", first["name"])
	}
}

func TestFieldAlias(t *testing.T) {
	if a := FieldAlias("provision.device.identifier"); a != "identifier" {
		t.Errorf("expected identifier, got %s", a)
	}
	if a := FieldAlias("wt"); a != "wt" {
		t.Errorf("expected wt, got %s", a)
	}
}

func TestCastValue(t *testing.T) {
	if v := castValue("true"); v != true {
		t.Errorf("expected true, got %v", v)
	}
	if v := castValue("42"); v != int64(42) {
		t.Errorf("expected 42, got %v (%T)", v, v)
	}
	if v := castValue("hello"); v != "hello" {
		t.Errorf("expected hello, got %v", v)
	}
}
