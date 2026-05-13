package unwrap

import (
	"regexp"
	"strconv"
	"strings"
)

// knownJSFieldNames is the set of widget config field names that OpenGate uses
// to store JavaScript code. Top-level or nested, any string-typed entry with
// one of these names is extracted to a .js file.
//
// `_widgetConfigCode` is the canonical field for chart widget code in
// OpenGate (seen in customChart widgets); the leading underscore indicates
// internal/metadata in OG's conventions.
var knownJSFieldNames = map[string]bool{
	"formatter":         true,
	"script":            true,
	"operation":         true,
	"code":              true,
	"fn":                true,
	"expression":        true,
	"_widgetconfigcode": true,
}

// jsKeywordPattern matches strings that look like JavaScript code by content
// rather than by field name. We require a minimum length and at least one
// recognisable JS construct.
var jsKeywordPattern = regexp.MustCompile(`\b(function\b|return\b|=>|const\s|let\s|var\s)`)

const minHeuristicLen = 40

// looksLikeJS returns true when the string is long enough and contains a
// pattern characteristic of JavaScript.
func looksLikeJS(s string) bool {
	if len(s) < minHeuristicLen {
		return false
	}
	return jsKeywordPattern.MatchString(s)
}

// keyPath represents a position in a nested JSON document. Object keys are
// stored as-is; array indices as their decimal string. Used both as a logical
// path and as the basis of the .js filename.
type keyPath []string

// filename joins the keypath with "__" so it survives any filesystem.
// Example: ["columns", "0", "formatter"] → "columns__0__formatter.js".
func (k keyPath) filename() string {
	return strings.Join(k, "__") + ".js"
}

// parseFilename reverses filename(): turns "columns__0__formatter.js" back
// into ["columns", "0", "formatter"]. Returns nil for non-.js names.
func parseFilename(name string) keyPath {
	if !strings.HasSuffix(name, ".js") {
		return nil
	}
	stem := strings.TrimSuffix(name, ".js")
	if stem == "" {
		return nil
	}
	return strings.Split(stem, "__")
}

// ExtractJSFields walks an arbitrary JSON-decoded value, recursively pulling
// out any string-typed entry that is either named like a JS field or whose
// content matches the JS heuristic. Extracted strings are removed from the
// returned (cleaned) value, and their keypath → code is returned in jsFiles.
//
// node may be a map[string]any, []any, or any other JSON primitive. The
// returned cleaned value mirrors node's structure with JS fields removed.
func ExtractJSFields(node any) (cleaned any, jsFiles map[string]string) {
	jsFiles = make(map[string]string)
	cleaned = walkExtract(node, nil, jsFiles)
	return cleaned, jsFiles
}

func walkExtract(node any, path keyPath, out map[string]string) any {
	switch v := node.(type) {
	case map[string]any:
		result := make(map[string]any, len(v))
		for k, child := range v {
			childPath := append(append(keyPath{}, path...), k)
			if s, ok := child.(string); ok && shouldExtract(k, s) {
				out[childPath.filename()] = s
				continue
			}
			result[k] = walkExtract(child, childPath, out)
		}
		return result
	case []any:
		result := make([]any, len(v))
		for i, child := range v {
			childPath := append(append(keyPath{}, path...), strconv.Itoa(i))
			result[i] = walkExtract(child, childPath, out)
		}
		return result
	default:
		return v
	}
}

// shouldExtract decides whether the string value at key should be moved out
// to a .js file. Either the key is one we know carries code, or the value
// itself looks like JS by content.
func shouldExtract(key, value string) bool {
	if knownJSFieldNames[strings.ToLower(key)] {
		return true
	}
	return looksLikeJS(value)
}

// ReinjectJSFields takes a cleaned JSON-decoded value and a map of keypath
// filenames → code, and reinserts each code string into the value at its
// original location. Returns the modified value.
func ReinjectJSFields(node any, jsFiles map[string]string) any {
	for filename, code := range jsFiles {
		path := parseFilename(filename)
		if path == nil {
			continue
		}
		node = setAt(node, path, code)
	}
	return node
}

// setAt navigates node along path, creating intermediate maps if needed, and
// stores value at the leaf. Numeric path segments are treated as array indices.
func setAt(node any, path keyPath, value any) any {
	if len(path) == 0 {
		return value
	}
	head, rest := path[0], path[1:]

	if idx, err := strconv.Atoi(head); err == nil {
		arr, _ := node.([]any)
		for len(arr) <= idx {
			arr = append(arr, nil)
		}
		arr[idx] = setAt(arr[idx], rest, value)
		return arr
	}

	m, _ := node.(map[string]any)
	if m == nil {
		m = map[string]any{}
	}
	m[head] = setAt(m[head], rest, value)
	return m
}
