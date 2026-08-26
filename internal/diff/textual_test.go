package diff

import (
	"strings"
	"testing"
)

func TestTextIdenticalIsNoDiff(t *testing.T) {
	d := Text("javascript.js", "line one\nline two\n", "line one\nline two\n", 3)
	if d.Changed() {
		t.Errorf("identical files must not differ: %+v", d)
	}
}

// A trailing newline must not read as a changed line.
func TestTextTrailingNewlineIgnored(t *testing.T) {
	d := Text("javascript.js", "a\nb", "a\nb\n", 3)
	if d.Changed() {
		t.Errorf("a trailing newline is not a change: %+v", d)
	}
}

func TestTextCountsAndRenders(t *testing.T) {
	before := "one\ntwo\nthree\n"
	after := "one\nTWO\nthree\nfour\n"

	d := Text("javascript.js", before, after, 3)
	if d.Added != 2 || d.Removed != 1 {
		t.Errorf("added=%d removed=%d, want 2 and 1", d.Added, d.Removed)
	}
	if !strings.Contains(d.Unified, "- two") || !strings.Contains(d.Unified, "+ TWO") || !strings.Contains(d.Unified, "+ four") {
		t.Errorf("unified diff missing lines:\n%s", d.Unified)
	}
	if !strings.Contains(d.Unified, "  one") {
		t.Errorf("context line missing:\n%s", d.Unified)
	}
}

// Long unchanged stretches are collapsed, or a one-line change in a 500-line
// file prints the whole file.
func TestTextCollapsesUnchangedRuns(t *testing.T) {
	var before strings.Builder
	for i := 0; i < 40; i++ {
		before.WriteString("same\n")
	}
	after := before.String() + "new line\n"

	d := Text("javascript.js", before.String(), after, 2)
	if !strings.Contains(d.Unified, "unchanged line") {
		t.Errorf("long unchanged run should be collapsed:\n%s", d.Unified)
	}
	if lines := strings.Count(d.Unified, "\n"); lines > 8 {
		t.Errorf("collapsed diff should be short, got %d lines:\n%s", lines, d.Unified)
	}
}

func TestTextZeroContext(t *testing.T) {
	d := Text("f.js", "a\nb\nc\n", "a\nX\nc\n", 0)
	if strings.Contains(d.Unified, "  a") {
		t.Errorf("with zero context no unchanged line should print:\n%s", d.Unified)
	}
	if !strings.Contains(d.Unified, "- b") || !strings.Contains(d.Unified, "+ X") {
		t.Errorf("the change itself must still print:\n%s", d.Unified)
	}
}

func TestTextEmptySides(t *testing.T) {
	d := Text("f.js", "", "a\nb\n", 3)
	if d.Added != 2 || d.Removed != 0 {
		t.Errorf("added=%d removed=%d, want 2 and 0", d.Added, d.Removed)
	}
	d = Text("f.js", "a\nb\n", "", 3)
	if d.Added != 0 || d.Removed != 2 {
		t.Errorf("added=%d removed=%d, want 0 and 2", d.Added, d.Removed)
	}
}

// A moved line is one deletion and one addition, not a rewrite of everything
// in between: that is what the LCS buys.
func TestTextMovedLine(t *testing.T) {
	d := Text("f.js", "a\nb\nc\nd\n", "b\nc\nd\na\n", 3)
	if d.Added != 1 || d.Removed != 1 {
		t.Errorf("a moved line should be +1/-1, got +%d/-%d:\n%s", d.Added, d.Removed, d.Unified)
	}
}
