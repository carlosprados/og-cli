// Package diff compares two versions of an artifact and renders the result.
//
// Two kinds of difference are reported separately, because mixing them is what
// makes generated diffs unreadable:
//
//   - metadata, as a STRUCTURAL diff over the canonical form: added, removed
//     and changed fields, each named by its path;
//   - extracted source code, as a TEXTUAL unified diff.
//
// A structural diff of a JavaScript field would report one enormous string
// changing, and a textual diff of reordered JSON reports the whole document.
package diff

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// ChangeKind classifies one structural difference.
type ChangeKind string

const (
	Added   ChangeKind = "added"
	Removed ChangeKind = "removed"
	Changed ChangeKind = "changed"
)

// Marker is the character used in text output.
func (k ChangeKind) Marker() string {
	switch k {
	case Added:
		return "+"
	case Removed:
		return "-"
	default:
		return "~"
	}
}

// Change is one structural difference at one path.
type Change struct {
	Kind ChangeKind `json:"kind"`
	// Path is dotted, with array indices in brackets:
	// config.columns[2].title
	Path string `json:"path"`
	// Before and After are rendered compactly; nil where not applicable.
	Before any `json:"before,omitempty"`
	After  any `json:"after,omitempty"`
}

// Structural compares two canonical payloads field by field.
//
// Both sides must already be canonical: this reports what differs, it does not
// decide what counts as a difference. Volatile fields are gone by then, which
// is why a bumped __v does not show up here.
func Structural(before, after []byte) ([]Change, error) {
	var a, b any
	if err := json.Unmarshal(before, &a); err != nil {
		return nil, fmt.Errorf("parsing the before side: %w", err)
	}
	if err := json.Unmarshal(after, &b); err != nil {
		return nil, fmt.Errorf("parsing the after side: %w", err)
	}
	var changes []Change
	walk("", a, b, &changes)
	sort.Slice(changes, func(i, j int) bool { return changes[i].Path < changes[j].Path })
	return changes, nil
}

func walk(path string, before, after any, out *[]Change) {
	switch b := before.(type) {
	case map[string]any:
		a, ok := after.(map[string]any)
		if !ok {
			*out = append(*out, Change{Kind: Changed, Path: path, Before: summarize(before), After: summarize(after)})
			return
		}
		for _, key := range sortedKeys(b, a) {
			bv, inBefore := b[key]
			av, inAfter := a[key]
			child := join(path, key)
			switch {
			case inBefore && !inAfter:
				*out = append(*out, Change{Kind: Removed, Path: child, Before: summarize(bv)})
			case !inBefore && inAfter:
				*out = append(*out, Change{Kind: Added, Path: child, After: summarize(av)})
			default:
				walk(child, bv, av, out)
			}
		}
	case []any:
		a, ok := after.([]any)
		if !ok {
			*out = append(*out, Change{Kind: Changed, Path: path, Before: summarize(before), After: summarize(after)})
			return
		}
		// Compare positionally. Arrays here are ordered on purpose — a grid, a
		// list of datastreams — so element 2 becoming element 3 IS a change,
		// and pretending otherwise would hide a reordering.
		for i := 0; i < len(b) || i < len(a); i++ {
			child := fmt.Sprintf("%s[%d]", path, i)
			switch {
			case i >= len(a):
				*out = append(*out, Change{Kind: Removed, Path: child, Before: summarize(b[i])})
			case i >= len(b):
				*out = append(*out, Change{Kind: Added, Path: child, After: summarize(a[i])})
			default:
				walk(child, b[i], a[i], out)
			}
		}
	default:
		if !equalScalar(before, after) {
			*out = append(*out, Change{Kind: Changed, Path: path, Before: summarize(before), After: summarize(after)})
		}
	}
}

func sortedKeys(a, b map[string]any) []string {
	seen := make(map[string]bool, len(a)+len(b))
	keys := make([]string, 0, len(a)+len(b))
	for k := range a {
		if !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}
	for k := range b {
		if !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys
}

func join(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

func equalScalar(a, b any) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a == b
}

// maxInlineLen is where a value stops being quoted in full. Long code strings
// belong in a textual diff, not inline in a structural one.
const maxInlineLen = 60

// summarize renders a value compactly enough to sit on one line.
func summarize(v any) any {
	switch t := v.(type) {
	case string:
		if len(t) <= maxInlineLen {
			return t
		}
		return t[:maxInlineLen] + fmt.Sprintf("… (%d chars)", len(t))
	case map[string]any:
		return fmt.Sprintf("{%d fields}", len(t))
	case []any:
		return fmt.Sprintf("[%d items]", len(t))
	case float64:
		// Render whole numbers without a trailing .0, the way the payload had them.
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return t
	default:
		return v
	}
}

// Render writes the structural changes as indented lines.
func Render(changes []Change, indent string) string {
	var b strings.Builder
	for _, c := range changes {
		switch c.Kind {
		case Added:
			fmt.Fprintf(&b, "%s+ %s: %v\n", indent, c.Path, c.After)
		case Removed:
			fmt.Fprintf(&b, "%s- %s: %v\n", indent, c.Path, c.Before)
		default:
			fmt.Fprintf(&b, "%s~ %s: %v → %v\n", indent, c.Path, c.Before, c.After)
		}
	}
	return b.String()
}
