// Package query translates simple filter expressions into OpenGate search JSON.
//
// Supported syntax:
//
//	Single condition:  "field op value"
//	Query string:      "field op value AND field op value"
//
// Operators: eq, neq, like, gt, lt, gte, lte, in, exists
//
// Multiple conditions are composed with AND.
package query

import (
	"encoding/json"
	"fmt"
	"strings"
)

var validOps = map[string]bool{
	"eq": true, "neq": true, "like": true,
	"gt": true, "lt": true, "gte": true, "lte": true,
	"in": true, "exists": true,
}

// Condition represents a single "field op value" filter.
type Condition struct {
	Field string
	Op    string
	Value string
}

// ParseCondition parses a single "field op value" string.
func ParseCondition(s string) (Condition, error) {
	s = strings.TrimSpace(s)
	parts := strings.SplitN(s, " ", 3)
	if len(parts) < 3 {
		// "exists" operator only needs field + op
		if len(parts) == 2 && parts[1] == "exists" {
			return Condition{Field: parts[0], Op: "exists", Value: "true"}, nil
		}
		return Condition{}, fmt.Errorf("invalid condition %q: expected \"field op value\"", s)
	}

	field := parts[0]
	op := strings.ToLower(parts[1])
	value := parts[2]

	if !validOps[op] {
		return Condition{}, fmt.Errorf("unknown operator %q (valid: %s)", op, validOpsList())
	}

	return Condition{Field: field, Op: op, Value: value}, nil
}

// ParseQuery parses a query string with conditions joined by AND.
// Example: "field1 eq value1 AND field2 like value2"
func ParseQuery(q string) ([]Condition, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil, nil
	}

	// Split on " AND " (case-insensitive)
	segments := splitAND(q)
	conditions := make([]Condition, 0, len(segments))

	for _, seg := range segments {
		c, err := ParseCondition(seg)
		if err != nil {
			return nil, err
		}
		conditions = append(conditions, c)
	}

	return conditions, nil
}

// SelectField is one projected sub-field of a datastream ("value", "at").
type SelectField struct {
	Field string `json:"field"`
	Alias string `json:"alias,omitempty"`
}

// SelectClause projects one datastream with its sub-fields, matching the
// OpenGate search select JSON shape.
type SelectClause struct {
	Name   string        `json:"name"`
	Fields []SelectField `json:"fields"`
}

// SelectFromFields converts -s style field names into select clauses.
// A "@at" suffix on a field adds the timestamp sub-field alongside the
// value: "wt@at" → value + at. withAt forces the at sub-field on every field.
func SelectFromFields(fields []string, withAt bool) []SelectClause {
	if len(fields) == 0 {
		return nil
	}
	clauses := make([]SelectClause, len(fields))
	for i, f := range fields {
		name, at := parseAtSuffix(f)
		clauses[i] = NewSelectClause(name, at || withAt)
	}
	return clauses
}

// NewSelectClause builds a clause for one datastream with auto-generated
// aliases: value → FieldAlias(name), at → FieldAlias(name) + "_at".
func NewSelectClause(name string, withAt bool) SelectClause {
	alias := FieldAlias(name)
	fields := []SelectField{{Field: "value", Alias: alias}}
	if withAt {
		fields = append(fields, SelectField{Field: "at", Alias: alias + "_at"})
	}
	return SelectClause{Name: name, Fields: fields}
}

// parseAtSuffix splits the "@at" marker from a field name.
func parseAtSuffix(field string) (name string, at bool) {
	if n, found := strings.CutSuffix(field, "@at"); found {
		return n, true
	}
	return field, false
}

// SearchParams groups all parameters for building a search request.
type SearchParams struct {
	Conditions []Condition
	Limit      int
	Select     []SelectClause
}

// BuildFilter converts SearchParams into the OpenGate search JSON body.
func BuildFilter(p SearchParams) (json.RawMessage, error) {
	if len(p.Conditions) == 0 && p.Limit == 0 && len(p.Select) == 0 {
		return json.RawMessage("{}"), nil
	}

	body := make(map[string]any)

	if len(p.Conditions) == 1 {
		body["filter"] = conditionToMap(p.Conditions[0])
	} else if len(p.Conditions) > 1 {
		clauses := make([]map[string]any, len(p.Conditions))
		for i, c := range p.Conditions {
			clauses[i] = conditionToMap(c)
		}
		body["filter"] = map[string]any{"and": clauses}
	}

	if p.Limit > 0 {
		body["limit"] = map[string]any{"size": p.Limit}
	}

	if len(p.Select) > 0 {
		body["select"] = p.Select
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("building filter JSON: %w", err)
	}
	return data, nil
}

// MergeWithRaw builds a filter from SearchParams, or returns raw JSON if provided.
// Raw filter takes precedence.
func MergeWithRaw(p SearchParams, raw string) (json.RawMessage, error) {
	if raw != "" {
		return json.RawMessage(raw), nil
	}
	if len(p.Conditions) == 0 && p.Limit == 0 && len(p.Select) == 0 {
		return nil, nil
	}
	return BuildFilter(p)
}

// FieldAlias returns a short column name from a dotted field path.
// "provision.device.identifier" → "identifier"
// "wt" → "wt"
func FieldAlias(field string) string {
	parts := strings.Split(field, ".")
	return parts[len(parts)-1]
}

func conditionToMap(c Condition) map[string]any {
	return map[string]any{
		c.Op: map[string]any{
			c.Field: castValue(c.Value),
		},
	}
}

// castValue tries to interpret the value as number or bool, falls back to string.
func castValue(s string) any {
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}
	// Try integer
	var i int64
	if _, err := fmt.Sscanf(s, "%d", &i); err == nil && fmt.Sprintf("%d", i) == s {
		return i
	}
	// Try float
	var f float64
	if _, err := fmt.Sscanf(s, "%f", &f); err == nil {
		return f
	}
	return s
}

// splitAND splits a query string on " AND " boundaries (case-insensitive).
func splitAND(q string) []string {
	var result []string
	upper := strings.ToUpper(q)
	for {
		idx := strings.Index(upper, " AND ")
		if idx < 0 {
			result = append(result, strings.TrimSpace(q))
			break
		}
		result = append(result, strings.TrimSpace(q[:idx]))
		q = q[idx+5:]
		upper = upper[idx+5:]
	}
	return result
}

func validOpsList() string {
	ops := make([]string, 0, len(validOps))
	for op := range validOps {
		ops = append(ops, op)
	}
	return strings.Join(ops, ", ")
}
