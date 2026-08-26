package diff

import (
	"strings"
	"testing"
)

func TestStructuralDetectsEachKind(t *testing.T) {
	before := []byte(`{"name":"R","active":true,"gone":"x","n":1}`)
	after := []byte(`{"name":"R","active":false,"added":"y","n":2}`)

	changes, err := Structural(before, after)
	if err != nil {
		t.Fatal(err)
	}

	got := map[string]Change{}
	for _, c := range changes {
		got[c.Path] = c
	}

	if c := got["active"]; c.Kind != Changed || c.Before != true || c.After != false {
		t.Errorf("active: %+v", c)
	}
	if c := got["gone"]; c.Kind != Removed || c.Before != "x" {
		t.Errorf("gone: %+v", c)
	}
	if c := got["added"]; c.Kind != Added || c.After != "y" {
		t.Errorf("added: %+v", c)
	}
	if c := got["n"]; c.Kind != Changed || c.Before != "1" || c.After != "2" {
		t.Errorf("n: %+v (numbers should render without a trailing .0)", c)
	}
	if _, present := got["name"]; present {
		t.Error("an unchanged field must not be reported")
	}
}

func TestStructuralNestedPaths(t *testing.T) {
	before := []byte(`{"config":{"columns":[{"title":"A"},{"title":"B"}]}}`)
	after := []byte(`{"config":{"columns":[{"title":"A"},{"title":"Z"}]}}`)

	changes, err := Structural(before, after)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 {
		t.Fatalf("expected one change, got %+v", changes)
	}
	if changes[0].Path != "config.columns[1].title" {
		t.Errorf("path = %q, want config.columns[1].title", changes[0].Path)
	}
}

// Arrays are ordered on purpose here — a dashboard grid, a datastream list — so
// a reordering is a real change and must not be hidden.
func TestStructuralReportsReordering(t *testing.T) {
	changes, err := Structural([]byte(`{"grid":["a","b"]}`), []byte(`{"grid":["b","a"]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 2 {
		t.Errorf("a reordering must be reported, got %+v", changes)
	}
}

func TestStructuralArrayLengthChange(t *testing.T) {
	changes, err := Structural([]byte(`{"a":[1,2]}`), []byte(`{"a":[1,2,3]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 || changes[0].Kind != Added || changes[0].Path != "a[2]" {
		t.Errorf("expected one addition at a[2], got %+v", changes)
	}
}

func TestStructuralTypeChange(t *testing.T) {
	changes, err := Structural([]byte(`{"a":{"b":1}}`), []byte(`{"a":[1]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 || changes[0].Kind != Changed {
		t.Errorf("a container changing type is one change, got %+v", changes)
	}
}

// Long values are truncated: a full code string belongs in a textual diff.
func TestStructuralTruncatesLongValues(t *testing.T) {
	long := strings.Repeat("x", 200)
	changes, err := Structural([]byte(`{"code":"short"}`), []byte(`{"code":"`+long+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	after, _ := changes[0].After.(string)
	if len(after) > maxInlineLen+30 {
		t.Errorf("long value not truncated: %d chars", len(after))
	}
	if !strings.Contains(after, "200 chars") {
		t.Errorf("truncation should state the real length: %q", after)
	}
}

func TestStructuralNoChanges(t *testing.T) {
	changes, err := Structural([]byte(`{"a":1,"b":{"c":[1,2]}}`), []byte(`{"b":{"c":[1,2]},"a":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 0 {
		t.Errorf("key order must not produce changes: %+v", changes)
	}
}

func TestStructuralInvalidJSON(t *testing.T) {
	if _, err := Structural([]byte(`{`), []byte(`{}`)); err == nil {
		t.Error("expected an error for malformed input")
	}
}

func TestRenderMarkers(t *testing.T) {
	out := Render([]Change{
		{Kind: Added, Path: "a", After: "1"},
		{Kind: Removed, Path: "b", Before: "2"},
		{Kind: Changed, Path: "c", Before: "3", After: "4"},
	}, "  ")

	for _, want := range []string{"+ a: 1", "- b: 2", "~ c: 3 → 4"} {
		if !strings.Contains(out, want) {
			t.Errorf("render missing %q:\n%s", want, out)
		}
	}
}
