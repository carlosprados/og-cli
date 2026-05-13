package unwrap

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestExtractJSFields_TopLevelByName(t *testing.T) {
	raw := `{
		"title": "T",
		"formatter": "function(v){return v;}",
		"script": "doStuff();",
		"reloadPeriod": "60"
	}`
	var node any
	if err := json.Unmarshal([]byte(raw), &node); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	cleaned, js := ExtractJSFields(node)

	cleanedMap := cleaned.(map[string]any)
	if _, present := cleanedMap["formatter"]; present {
		t.Error("formatter should have been removed from cleaned config")
	}
	if _, present := cleanedMap["script"]; present {
		t.Error("script should have been removed from cleaned config")
	}
	if cleanedMap["title"] != "T" {
		t.Errorf("non-JS field corrupted: title=%v", cleanedMap["title"])
	}
	if cleanedMap["reloadPeriod"] != "60" {
		t.Errorf("short string with no JS keywords must stay: reloadPeriod=%v", cleanedMap["reloadPeriod"])
	}

	if js["formatter.js"] != "function(v){return v;}" {
		t.Errorf("formatter.js wrong content: %q", js["formatter.js"])
	}
	if js["script.js"] != "doStuff();" {
		t.Errorf("script.js wrong content: %q", js["script.js"])
	}
}

func TestExtractJSFields_Nested(t *testing.T) {
	raw := `{
		"columns": [
			{"path": "wt", "formatter": "function(v){return v + ' C';}"},
			{"path": "wp", "title": "Pressure"}
		]
	}`
	var node any
	json.Unmarshal([]byte(raw), &node)

	_, js := ExtractJSFields(node)

	if got, want := js["columns__0__formatter.js"], "function(v){return v + ' C';}"; got != want {
		t.Errorf("nested formatter wrong: got %q want %q", got, want)
	}
	if _, present := js["columns__1__formatter.js"]; present {
		t.Error("there is no formatter in columns[1] — should not appear")
	}
}

func TestExtractJSFields_Heuristic(t *testing.T) {
	// Field name is NOT in the known list, but content looks like JS.
	longJS := "let total = 0; for (const x of items) { total += x; } return total;"
	raw := `{"customLogic": ` + jsonString(longJS) + `, "shortField": "ok"}`

	var node any
	json.Unmarshal([]byte(raw), &node)
	cleaned, js := ExtractJSFields(node)

	if got := js["customLogic.js"]; got != longJS {
		t.Errorf("heuristic should have caught customLogic: got %q", got)
	}
	if v := cleaned.(map[string]any)["customLogic"]; v != nil {
		t.Errorf("customLogic should have been removed: %v", v)
	}
	if v := cleaned.(map[string]any)["shortField"]; v != "ok" {
		t.Errorf("shortField must remain: %v", v)
	}
}

func TestExtractJSFields_Roundtrip(t *testing.T) {
	raw := `{
		"title": "T",
		"formatter": "function(v){return v;}",
		"columns": [
			{"path": "wt", "formatter": "function(v){return v + ' C';}"},
			{"path": "wp"}
		]
	}`
	var original, want any
	json.Unmarshal([]byte(raw), &original)
	json.Unmarshal([]byte(raw), &want)

	cleaned, js := ExtractJSFields(original)
	rebuilt := ReinjectJSFields(cleaned, js)

	if !reflect.DeepEqual(rebuilt, want) {
		t.Errorf("roundtrip mismatch.\n  got:  %#v\n  want: %#v", rebuilt, want)
	}
}

func TestLooksLikeJS(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		// Short strings always fail the heuristic; they are caught by field name only.
		{"function(){ return 1; }", false}, // 23 chars — under threshold
		{"const x = items.filter(i => i > 0); return x;", true},
		{"60", false},
		{"yes this is a long string but no keywords here at all really", false},
		{"abc def ghi jkl mno pqr stu vwx yz function this should be long enough", true},
	}
	for _, c := range cases {
		got := looksLikeJS(c.in)
		if got != c.want {
			t.Errorf("looksLikeJS(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// jsonString returns s as a properly-escaped JSON string literal (with quotes).
func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
