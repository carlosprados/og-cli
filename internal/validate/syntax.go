package validate

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// checkBalance reports unbalanced brackets in JavaScript source.
//
// This is not a parser and does not pretend to be one. It is the check that
// catches what actually goes wrong when someone edits a script by hand — a brace
// left unclosed, one too many — and it costs nothing. Anything subtler is the
// job of the generated typings, which a real type-checker validates in the
// editor.
//
// It must skip string literals, template literals, comments AND regular
// expression literals: a bracket inside any of those is not a bracket. Regexes
// are the awkward one, because `/` is also division, and telling them apart
// needs the preceding token — a real connector function in production contains
// `message.replace(/\'/g, ”)`, whose escaped quote inside the regex desynced an
// earlier version of this check and reported a false error on working code.
func checkBalance(file, code string) []Finding {
	type frame struct {
		ch   byte
		line int
	}
	var stack []frame
	line := 1

	closers := map[byte]byte{')': '(', ']': '[', '}': '{'}

	// lastSignificant is the previous non-space, non-comment byte, which is what
	// decides whether a '/' opens a regex or divides.
	var lastSignificant byte
	// lastWord is the identifier or keyword immediately before, for `return /re/`.
	var lastWord []byte

	for i := 0; i < len(code); i++ {
		c := code[i]

		switch {
		case c == '\n':
			line++
			lastSignificant = '\n'
			lastWord = nil

		case c == ' ' || c == '\t' || c == '\r':
			// Whitespace ends a word but does not change what precedes.
			if len(lastWord) > 0 {
				lastWord = nil
			}

		case c == '/' && i+1 < len(code) && code[i+1] == '/':
			for i < len(code) && code[i] != '\n' {
				i++
			}
			line++
			lastSignificant = '\n'

		case c == '/' && i+1 < len(code) && code[i+1] == '*':
			i += 2
			for i+1 < len(code) && (code[i] != '*' || code[i+1] != '/') {
				if code[i] == '\n' {
					line++
				}
				i++
			}
			i++

		case c == '/' && startsRegex(lastSignificant, string(lastWord)):
			// A regex literal: consume to the closing '/', honouring escapes and
			// character classes, where '/' is literal.
			i++
			inClass := false
			for i < len(code) {
				switch code[i] {
				case '\\':
					i++
				case '[':
					inClass = true
				case ']':
					inClass = false
				case '/':
					if !inClass {
						goto regexDone
					}
				case '\n':
					// An unterminated regex; stop rather than run to EOF.
					line++
					goto regexDone
				}
				i++
			}
		regexDone:
			lastSignificant = '/'
			lastWord = nil

		case c == '"' || c == '\'' || c == '`':
			quote := c
			i++
			for i < len(code) && code[i] != quote {
				switch code[i] {
				case '\\':
					i++
				case '\n':
					line++
				}
				i++
			}
			lastSignificant = quote
			lastWord = nil

		case c == '(' || c == '[' || c == '{':
			stack = append(stack, frame{ch: c, line: line})
			lastSignificant = c
			lastWord = nil

		case c == ')' || c == ']' || c == '}':
			want := closers[c]
			if len(stack) == 0 {
				return []Finding{{Severity: Error, File: file, Line: line,
					Message: fmt.Sprintf("unexpected %q — nothing is open here", string(c))}}
			}
			top := stack[len(stack)-1]
			if top.ch != want {
				return []Finding{{Severity: Error, File: file, Line: line,
					Message: fmt.Sprintf("%q closes a %q opened on line %d", string(c), string(top.ch), top.line)}}
			}
			stack = stack[:len(stack)-1]
			lastSignificant = c
			lastWord = nil

		default:
			lastSignificant = c
			if isWordByte(c) {
				lastWord = append(lastWord, c)
			} else {
				lastWord = nil
			}
		}
	}

	if len(stack) > 0 {
		open := stack[len(stack)-1]
		return []Finding{{Severity: Error, File: file, Line: open.line,
			Message: fmt.Sprintf("%q is never closed", string(open.ch))}}
	}
	return nil
}

// regexPrefixKeywords are the keywords after which a '/' starts a regex rather
// than dividing.
var regexPrefixKeywords = map[string]bool{
	"return": true, "typeof": true, "instanceof": true, "in": true, "of": true,
	"new": true, "delete": true, "void": true, "throw": true, "case": true,
	"do": true, "else": true, "yield": true, "await": true,
}

// startsRegex decides whether a '/' opens a regex literal, from the token before
// it. After a value — an identifier, a number, a closing bracket, a string — it
// is division; after an operator, a separator or one of the keywords above, it
// is a regex.
func startsRegex(prev byte, word string) bool {
	if word != "" {
		return regexPrefixKeywords[word]
	}
	switch prev {
	case 0, '\n', '(', ',', '=', ':', '[', '!', '&', '|', '?', '{', '}', ';', '+', '-', '*', '%', '<', '>', '~', '^':
		return true
	default:
		return false
	}
}

func isWordByte(c byte) bool {
	return c == '_' || c == '$' ||
		(c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// jsonErrorLine turns a json.SyntaxError offset into a line number, so a
// malformed metadata file points somewhere useful.
func jsonErrorLine(data []byte, err error) int {
	var syn *json.SyntaxError
	if !errors.As(err, &syn) {
		return 0
	}
	offset := int(syn.Offset)
	if offset > len(data) {
		offset = len(data)
	}
	return strings.Count(string(data[:offset]), "\n") + 1
}
