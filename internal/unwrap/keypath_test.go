package unwrap

import (
	"encoding/json"
	"reflect"
	"testing"
)

// roundtripTree mirrors what pull/wrap do to a config: extract the code, then
// reinject it from the files on disk.
func roundtripTree(t *testing.T, in string) any {
	t.Helper()
	var node any
	if err := json.Unmarshal([]byte(in), &node); err != nil {
		t.Fatal(err)
	}
	cleaned, files := ExtractJSFields(node)
	return ReinjectJSFields(cleaned, files)
}

func assertLossless(t *testing.T, label, in string) {
	t.Helper()
	var want any
	if err := json.Unmarshal([]byte(in), &want); err != nil {
		t.Fatal(err)
	}
	got := roundtripTree(t, in)
	if !reflect.DeepEqual(want, got) {
		gotJSON, _ := json.Marshal(got)
		t.Errorf("%s: round trip not lossless\n  in : %s\n  out: %s", label, in, gotJSON)
	}
}

// An object keyed by number must stay an object. Inferring the container type
// from the segment's spelling turned it into an array.
func TestRoundtripNumericObjectKeys(t *testing.T) {
	assertLossless(t, "numeric key holding code",
		`{"series":{"0":{"formatter":"function(v){return v;}"}}}`)
	assertLossless(t, "numeric keys, several",
		`{"axes":{"0":{"formatter":"function(a){return a;}"},"1":{"formatter":"function(b){return b;}"}}}`)
	assertLossless(t, "numeric key beside a real array",
		`{"m":{"2":{"code":"function(){return 1;}"}},"a":[{"code":"function(){return 2;}"}]}`)
}

// A key containing two consecutive underscores collided with the separator:
// both encoded to the same filename, so one overwrote the other and the loser
// came back empty.
func TestRoundtripSeparatorCollision(t *testing.T) {
	assertLossless(t, "colliding key and nested path",
		`{"a__b":{"code":"function(){return 1;}"},"a":{"b":{"code":"function(){return 2;}"}}}`)
	assertLossless(t, "underscore run of three",
		`{"a___b":{"code":"function(){return 3;}"}}`)
	assertLossless(t, "percent in a key",
		`{"a%b":{"code":"function(){return 4;}"}}`)
}

// Arrays and single-underscore names must keep the filenames they always had —
// no existing tree gets renamed by the escaping.
func TestFilenameStability(t *testing.T) {
	cases := map[string]keyPath{
		"columns__0__formatter.js":      {"columns", "0", "formatter"},
		"columns__2___formatterCode.js": {"columns", "2", "_formatterCode"},
		"_widgetConfigCode.js":          {"_widgetConfigCode"},
		"javascript.js":                 {"javascript"},
		"scriptProcessor__script.js":    {"scriptProcessor", "script"},
	}
	for want, path := range cases {
		if got := path.filename(); got != want {
			t.Errorf("filename(%v) = %q, want %q", path, got, want)
		}
		if got := parseFilename(want); !reflect.DeepEqual(got, path) {
			t.Errorf("parseFilename(%q) = %v, want %v", want, got, path)
		}
	}
}

// Filenames written before escaping existed must still parse.
func TestParseFilenameBackwardCompatible(t *testing.T) {
	got := parseFilename("columns__0___formatterCode.js")
	want := keyPath{"columns", "0", "_formatterCode"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
