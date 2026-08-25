package unwrap

import (
	"regexp"
)

// jsKeywordPattern matches strings that look like JavaScript code by content
// rather than by field name. We require a minimum length and at least one
// recognisable JS construct.
//
// Content sniffing no longer decides the on-disk layout — CodeContract does
// (see contract.go). It survives as a warning signal, and as the transitional
// fallback for widget configs, whose code fields are widget-type specific and
// not yet fully enumerated.
var jsKeywordPattern = regexp.MustCompile(`\b(function\b|return\b|=>|const\s|let\s|var\s)`)

const minHeuristicLen = 40

// looksLikeJS returns true when the string is long enough and contains a
// pattern characteristic of JavaScript.
//
// Known to be wrong in both directions: it misses a formatter written as a
// bare assignment ("cellFormatter.customValue = value;"), and it fires on
// prose containing "return" or "let". That is precisely why it is no longer
// authoritative.
func looksLikeJS(s string) bool {
	if len(s) < minHeuristicLen {
		return false
	}
	return jsKeywordPattern.MatchString(s)
}

// ExtractJSFields extracts the code fields of a widget config, the default
// contract for a document whose family is not known.
//
// Deprecated: call CodeContract.Extract with the contract of the artifact
// family you are handling, so the execution context of each file is recorded.
func ExtractJSFields(node any) (cleaned any, jsFiles map[string]string) {
	cleaned, jsFiles, _ = WidgetContract().Extract(node, nil)
	return cleaned, jsFiles
}

// ReinjectJSFields reinserts widget code files at their original keypaths.
//
// Deprecated: call CodeContract.Reinject, which also reports the .js files it
// declines to inject.
func ReinjectJSFields(node any, jsFiles map[string]string) any {
	return WidgetContract().Reinject(node, jsFiles, nil)
}
