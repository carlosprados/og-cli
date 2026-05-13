// Package unwrap turns an OpenGate workspace JSON into a navigable directory
// tree (and back), separating widget configuration from JavaScript code so it
// can be edited with an IDE — and consumed by AI agents.
package unwrap

import (
	"regexp"
	"strings"
	"unicode"
)

var nonAlphanum = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify converts an arbitrary name into a filesystem-safe kebab-case slug.
// Returns "unnamed" if the input collapses to empty.
func Slugify(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
		default:
			b.WriteRune('-')
		}
	}
	s := nonAlphanum.ReplaceAllString(b.String(), "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "unnamed"
	}
	return s
}

// DedupedSlug returns a unique slug for (name, id) given a set of slugs already
// taken. When the base slug is free, it is returned as-is. On collision, it
// appends "__<shortID>" where shortID is the last 12 characters of id (or the
// full id if shorter) lowercased and slugified.
func DedupedSlug(name, id string, taken map[string]bool) string {
	base := Slugify(name)
	if !taken[base] {
		taken[base] = true
		return base
	}
	short := id
	if len(short) > 12 {
		short = short[len(short)-12:]
	}
	candidate := base + "__" + Slugify(short)
	if !taken[candidate] {
		taken[candidate] = true
		return candidate
	}
	// Last resort: append full id slug.
	candidate = base + "__" + Slugify(id)
	taken[candidate] = true
	return candidate
}
