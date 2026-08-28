package typegen

import (
	"strings"
	"testing"
)

func TestWrapWidgetPutsTheCodeInsideThePlatformsFunction(t *testing.T) {
	code := "var a = await thing();\nreturn { series: [] };\n"
	wrapped, offset := WrapWidget(ContextWidgetChart, code)

	if offset != 1 {
		t.Errorf("offset = %d, want 1", offset)
	}
	if !strings.HasPrefix(wrapped, "async function __ogWidget(") {
		t.Errorf("wrapper does not open with the async function:\n%s", wrapped)
	}
	// The author's first line must be the line right after the wrapper's, or the
	// arithmetic that maps diagnostics back is wrong.
	lines := strings.Split(wrapped, "\n")
	if lines[offset] != "var a = await thing();" {
		t.Errorf("line %d is %q, want the author's first line", offset, lines[offset])
	}

	// A customTable is paged and has no return contract; a customChart is not.
	table, _ := WrapWidget(ContextWidgetTable, code)
	if !strings.Contains(table, "pageElements, page, callback") {
		t.Errorf("customTable wrapper missing its paging parameters:\n%s", table)
	}
	if strings.Contains(wrapped, "pageElements") {
		t.Error("customChart wrapper declares the table's paging parameters")
	}
}

func TestParseDiagnosticsMapsBackToTheAuthorsFile(t *testing.T) {
	out := strings.Join([]string{
		"../.og-widget-check.js(18,15): error TS2551: Property 'x' does not exist. Did you mean 'y'?",
		"../.og-widget-check.js(1,1): error TS1005: ';' expected.",
		"noise that is not a diagnostic",
	}, "\n")

	diags := ParseDiagnostics(out, 1)
	if len(diags) != 1 {
		t.Fatalf("got %d diagnostics, want 1 (the wrapper's own line must be dropped): %+v", len(diags), diags)
	}
	d := diags[0]
	if d.Line != 17 || d.Column != 15 {
		t.Errorf("position = %d:%d, want 17:15", d.Line, d.Column)
	}
	if d.Code != "TS2551" {
		t.Errorf("code = %q, want TS2551", d.Code)
	}
	if !strings.HasPrefix(d.Message, "Property 'x' does not exist") {
		t.Errorf("message = %q", d.Message)
	}
}

// The command is only worth having if it keeps the $api findings and drops the
// two idioms TypeScript disagrees with. Both live sensehat widgets contain the
// latter, and neither should report anything.
func TestClassifyKeepsRealFindingsAndSetsAsideFriction(t *testing.T) {
	code := strings.Join([]string{
		"var latest = null;",          // 1
		"if (latest.value) { go(); }", // 2
		"pts.sort(function (a, b) { return new Date(a[0]) - new Date(b[0]); });", // 3
		"$api.datapointsSearchBuildr();",                                         // 4
		"var n = obj - other;",                                                   // 5
	}, "\n")

	diags := []Diagnostic{
		{Line: 2, Code: "TS2339", Message: "Property 'value' does not exist on type 'never'."},
		{Line: 3, Code: "TS2362", Message: "The left-hand side of an arithmetic operation must be of type 'any', 'number', 'bigint' or an enum type."},
		{Line: 4, Code: "TS2551", Message: "Property 'datapointsSearchBuildr' does not exist on type 'OpenGateAPI'. Did you mean 'datapointsSearchBuilder'?"},
		{Line: 4, Code: "TS2339", Message: "Property 'nope' does not exist on type 'OpenGateAPI'."},
		{Line: 5, Code: "TS2362", Message: "The left-hand side of an arithmetic operation must be of type 'any', 'number', 'bigint' or an enum type."},
	}

	real, friction := Classify(diags, code)

	if len(friction) != 2 {
		t.Errorf("set aside %d, want 2 (the never-narrowing and the date subtraction): %+v", len(friction), friction)
	}
	if len(real) != 3 {
		t.Fatalf("kept %d, want 3: %+v", len(real), real)
	}
	// A misspelled $api member is the whole point: TS2339 must survive when it is
	// not the `never` form, and arithmetic must survive when no date is involved.
	var codes []string
	for _, d := range real {
		codes = append(codes, d.Code)
	}
	want := "TS2551 TS2339 TS2362"
	if got := strings.Join(codes, " "); got != want {
		t.Errorf("kept codes = %q, want %q", got, want)
	}
}
