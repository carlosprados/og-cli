package unwrap

import (
	"strconv"
	"strings"
)

// keyPath represents a position in a nested JSON document. Object keys are
// stored as-is; array indices as their decimal string. Used both as a logical
// path and as the basis of the .js filename.
type keyPath []string

// Filenames join path segments with "__". A segment that itself contains two
// consecutive underscores would be indistinguishable from the separator, so
// those runs are percent-escaped (and "%" with them, to keep the escape
// unambiguous). Single underscores are left alone, which is what keeps
// _widgetConfigCode.js and columns__0___formatterCode.js spelled as they
// always were.
const (
	pathSep       = "__"
	escPercent    = "%25"
	escUnderscore = "%5f"
)

// escapeSegment makes a path segment safe to join with pathSep.
func escapeSegment(s string) string {
	s = strings.ReplaceAll(s, "%", escPercent)
	// Escape underscores only where they form a run of two or more: a lone
	// underscore cannot be confused with the separator.
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] != '_' {
			b.WriteByte(s[i])
			i++
			continue
		}
		run := 0
		for i+run < len(s) && s[i+run] == '_' {
			run++
		}
		if run == 1 {
			b.WriteByte('_')
		} else {
			b.WriteString(strings.Repeat(escUnderscore, run))
		}
		i += run
	}
	return b.String()
}

// unescapeSegment reverses escapeSegment.
func unescapeSegment(s string) string {
	s = strings.ReplaceAll(s, escUnderscore, "_")
	return strings.ReplaceAll(s, escPercent, "%")
}

// filename joins the keypath with "__" so it survives any filesystem.
// Example: ["columns", "0", "formatter"] → "columns__0__formatter.js".
func (k keyPath) filename() string {
	parts := make([]string, len(k))
	for i, seg := range k {
		parts[i] = escapeSegment(seg)
	}
	return strings.Join(parts, pathSep) + ".js"
}

// parseFilename reverses filename(): turns "columns__0__formatter.js" back
// into ["columns", "0", "formatter"]. Returns nil for non-.js names.
//
// Files written before segment escaping existed parse unchanged: they contain
// no percent escapes, so unescapeSegment is a no-op on them.
func parseFilename(name string) keyPath {
	if !strings.HasSuffix(name, ".js") {
		return nil
	}
	stem := strings.TrimSuffix(name, ".js")
	if stem == "" {
		return nil
	}
	raw := strings.Split(stem, pathSep)
	out := make(keyPath, len(raw))
	for i, seg := range raw {
		out[i] = unescapeSegment(seg)
	}
	return out
}

// setAt navigates node along path and stores value at the leaf.
//
// The container at each position decides how its segment is read: an existing
// map treats "0" as a key, an existing slice as an index. Inferring the
// container type from the segment's spelling — as this used to — silently
// turned an object keyed by number into an array on every round trip. Only
// when the container is absent must the type be guessed, and there a numeric
// segment means an array.
func setAt(node any, path keyPath, value any) any {
	if len(path) == 0 {
		return value
	}
	head, rest := path[0], path[1:]

	switch container := node.(type) {
	case map[string]any:
		container[head] = setAt(container[head], rest, value)
		return container

	case []any:
		idx, err := strconv.Atoi(head)
		if err != nil || idx < 0 {
			// A named segment against an array: the path does not match the
			// document. Leave the array untouched rather than corrupt it.
			return container
		}
		for len(container) <= idx {
			container = append(container, nil)
		}
		container[idx] = setAt(container[idx], rest, value)
		return container

	default:
		// Absent (or scalar) container: create one. Here, and only here, the
		// segment's spelling is the only signal available.
		if idx, err := strconv.Atoi(head); err == nil && idx >= 0 {
			arr := make([]any, idx+1)
			arr[idx] = setAt(nil, rest, value)
			return arr
		}
		return map[string]any{head: setAt(nil, rest, value)}
	}
}
