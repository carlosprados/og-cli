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
	"strconv"
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

// fieldSubSuffixes are the projectable metadata sub-fields of a datastream's
// current value, selectable via an @-suffix on a -s field. `at` = platform
// reception time, `date` = measurement time, `source` = origin.
var fieldSubSuffixes = []string{"at", "date", "source"}

// SelectFromFields converts -s style field names into select clauses.
// An "@<sub>" suffix adds a metadata sub-field alongside the value and is
// repeatable: "wt@at" → value + at, "wt@at@date" → value + at + date.
// withAt forces the at sub-field on every field (the --at flag).
func SelectFromFields(fields []string, withAt bool) []SelectClause {
	if len(fields) == 0 {
		return nil
	}
	clauses := make([]SelectClause, len(fields))
	for i, f := range fields {
		name, subs := parseFieldSuffix(f)
		if withAt && !containsString(subs, "at") {
			subs = append(subs, "at")
		}
		clauses[i] = NewSelectClause(name, subs...)
	}
	return clauses
}

// NewSelectClause builds a clause for one datastream: always the value, plus
// any requested sub-fields (at/date/source). Aliases are auto-generated:
// value → FieldAlias(name), sub → FieldAlias(name) + "_" + sub.
func NewSelectClause(name string, subs ...string) SelectClause {
	alias := FieldAlias(name)
	fields := []SelectField{{Field: "value", Alias: alias}}
	for _, sub := range subs {
		fields = append(fields, SelectField{Field: sub, Alias: alias + "_" + sub})
	}
	return SelectClause{Name: name, Fields: fields}
}

// parseFieldSuffix strips trailing @-markers (@at, @date, @source) from a field
// name, returning the bare name and the requested sub-fields in projection
// order. "wt@at@date" → ("wt", ["at","date"]); "wt" → ("wt", nil).
func parseFieldSuffix(field string) (name string, subs []string) {
	name = field
	for {
		cut := false
		for _, sub := range fieldSubSuffixes {
			if n, found := strings.CutSuffix(name, "@"+sub); found {
				subs = append([]string{sub}, subs...)
				name = n
				cut = true
				break
			}
		}
		if !cut {
			break
		}
	}
	return name, subs
}

func containsString(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
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

// castValue interprets a filter value's JSON type so the search lake (which is
// type-strict: a string field never matches a number) receives the right type.
//
//   - A single- or double-quoted value is forced to string and unquoted — the
//     escape hatch for all-digit string identifiers, e.g. `eq '00123'`.
//   - "true"/"false" → bool.
//   - A value that is ENTIRELY an integer or float → number.
//   - Everything else → string.
//
// strconv (not fmt.Sscanf) is deliberate: Sscanf stops at the first non-numeric
// rune, so "192.168.0.1" or an ISO-8601 timestamp would be silently truncated to
// a number and never match. strconv.Parse* require the whole string to parse,
// which also makes timestamp filters (`<ds>._current.at gte <iso>`) work as
// strings.
func castValue(s string) any {
	if len(s) >= 2 {
		if q := s[0]; (q == '\'' || q == '"') && s[len(s)-1] == q {
			return s[1 : len(s)-1]
		}
	}
	switch s {
	case "true":
		return true
	case "false":
		return false
	}
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
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
