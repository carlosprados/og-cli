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
		Select: SelectFromFields([]string{"provision.device.identifier", "wt"}, false),
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

func TestSelectFromFields(t *testing.T) {
	tests := []struct {
		name   string
		fields []string
		withAt bool
		want   []SelectClause
	}{
		{
			name:   "plain field",
			fields: []string{"wt"},
			want: []SelectClause{
				{Name: "wt", Fields: []SelectField{{Field: "value", Alias: "wt"}}},
			},
		},
		{
			name:   "dotted path alias",
			fields: []string{"provision.device.identifier"},
			want: []SelectClause{
				{Name: "provision.device.identifier", Fields: []SelectField{{Field: "value", Alias: "identifier"}}},
			},
		},
		{
			name:   "at suffix",
			fields: []string{"wt@at"},
			want: []SelectClause{
				{Name: "wt", Fields: []SelectField{
					{Field: "value", Alias: "wt"},
					{Field: "at", Alias: "wt_at"},
				}},
			},
		},
		{
			name:   "date suffix",
			fields: []string{"wt@date"},
			want: []SelectClause{
				{Name: "wt", Fields: []SelectField{
					{Field: "value", Alias: "wt"},
					{Field: "date", Alias: "wt_date"},
				}},
			},
		},
		{
			name:   "at and date suffixes combined",
			fields: []string{"device.identifier@at@date"},
			want: []SelectClause{
				{Name: "device.identifier", Fields: []SelectField{
					{Field: "value", Alias: "identifier"},
					{Field: "at", Alias: "identifier_at"},
					{Field: "date", Alias: "identifier_date"},
				}},
			},
		},
		{
			name:   "source suffix",
			fields: []string{"wt@source"},
			want: []SelectClause{
				{Name: "wt", Fields: []SelectField{
					{Field: "value", Alias: "wt"},
					{Field: "source", Alias: "wt_source"},
				}},
			},
		},
		{
			name:   "withAt does not duplicate an explicit @at",
			fields: []string{"wt@at"},
			withAt: true,
			want: []SelectClause{
				{Name: "wt", Fields: []SelectField{
					{Field: "value", Alias: "wt"},
					{Field: "at", Alias: "wt_at"},
				}},
			},
		},
		{
			name:   "withAt forces at on all fields",
			fields: []string{"wt", "device.temperature.value"},
			withAt: true,
			want: []SelectClause{
				{Name: "wt", Fields: []SelectField{
					{Field: "value", Alias: "wt"},
					{Field: "at", Alias: "wt_at"},
				}},
				{Name: "device.temperature.value", Fields: []SelectField{
					{Field: "value", Alias: "value"},
					{Field: "at", Alias: "value_at"},
				}},
			},
		},
		{
			name:   "empty",
			fields: nil,
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SelectFromFields(tt.fields, tt.withAt)
			gotJSON, _ := json.Marshal(got)
			wantJSON, _ := json.Marshal(tt.want)
			if string(gotJSON) != string(wantJSON) {
				t.Errorf("SelectFromFields(%v, %v) = %s, want %s", tt.fields, tt.withAt, gotJSON, wantJSON)
			}
		})
	}
}

func TestBuildFilterSelectShapeUnchanged(t *testing.T) {
	// Plain -s fields must produce the exact same select JSON as before
	// the rich-select refactor.
	p := SearchParams{Select: SelectFromFields([]string{"wt"}, false)}
	data, err := BuildFilter(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"select":[{"name":"wt","fields":[{"field":"value","alias":"wt"}]}]}`
	if string(data) != want {
		t.Errorf("BuildFilter select = %s, want %s", data, want)
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
	tests := []struct {
		in   string
		want any
	}{
		{"true", true},
		{"false", false},
		{"42", int64(42)},
		{"3.14", 3.14},
		{"hello", "hello"},
		// Regression: numeric-prefixed strings must NOT be truncated to numbers.
		{"192.168.0.1-04.42.1A.B7.D7.D0", "192.168.0.1-04.42.1A.B7.D7.D0"},
		{"1.2.3", "1.2.3"},
		{"2026-06-21T18:00:00.000+02:00", "2026-06-21T18:00:00.000+02:00"},
		// Quoting forces string (escape hatch for all-digit identifiers).
		{"'00123'", "00123"},
		{`"00123"`, "00123"},
		{"'42'", "42"},
		{"'true'", "true"},
		{"'hello world'", "hello world"},
	}
	for _, tt := range tests {
		if v := castValue(tt.in); v != tt.want {
			t.Errorf("castValue(%q) = %v (%T), want %v (%T)", tt.in, v, v, tt.want, tt.want)
		}
	}
}

func TestBuildFilterPagination(t *testing.T) {
	tests := []struct {
		name  string
		limit int
		start int
		want  string // the limit block, or "" when absent
	}{
		{name: "size only", limit: 100, want: `{"size":100}`},
		{name: "start only", start: 3, want: `{"start":3}`},
		{name: "size and start", limit: 50, start: 2, want: `{"size":50,"start":2}`},
		{name: "neither", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := BuildFilter(SearchParams{
				Conditions: []Condition{{Field: "a", Op: "eq", Value: "1"}},
				Limit:      tc.limit,
				Start:      tc.start,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var result map[string]any
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatal(err)
			}

			raw, present := result["limit"]
			if tc.want == "" {
				if present {
					t.Errorf("limit should be absent, got %v", raw)
				}
				return
			}
			if !present {
				t.Fatalf("limit missing from %s", data)
			}

			var want map[string]any
			if err := json.Unmarshal([]byte(tc.want), &want); err != nil {
				t.Fatal(err)
			}
			got, _ := json.Marshal(raw)
			wantJSON, _ := json.Marshal(want)
			if string(got) != string(wantJSON) {
				t.Errorf("limit = %s, want %s", got, wantJSON)
			}
		})
	}
}

// TestBuildFilterStartAloneIsNotDroppedFromEmptyParams guards the early-return
// shortcut: asking for page 3 with no other criterion must still emit a limit.
func TestBuildFilterStartAloneIsNotDroppedFromEmptyParams(t *testing.T) {
	data, err := BuildFilter(SearchParams{Start: 3})
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	limit, ok := result["limit"].(map[string]any)
	if !ok {
		t.Fatalf("limit missing from %s", data)
	}
	if limit["start"] != float64(3) {
		t.Errorf("limit.start = %v, want 3", limit["start"])
	}
}
