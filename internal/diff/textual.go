package diff

import (
	"fmt"
	"strings"
)

// FileDiff is the textual difference of one extracted source file.
type FileDiff struct {
	File    string `json:"file"`
	Added   int    `json:"added"`
	Removed int    `json:"removed"`
	Unified string `json:"unified,omitempty"`
	OnlyIn  string `json:"onlyIn,omitempty"` // "local" | "remote" when the file exists on one side only
}

// Changed reports whether there is anything to show.
func (f FileDiff) Changed() bool { return f.Added > 0 || f.Removed > 0 || f.OnlyIn != "" }

// Text computes a unified diff between two source files.
//
// Implemented here rather than pulled in: a line-level LCS is thirty lines, and
// this is a lean single binary. If word-level or move detection is ever wanted,
// that is the point to reach for a library.
func Text(name, before, after string, contextLines int) FileDiff {
	d := FileDiff{File: name}
	if before == after {
		return d
	}

	a := splitLines(before)
	b := splitLines(after)
	ops := lcsDiff(a, b)

	for _, op := range ops {
		switch op.kind {
		case opAdd:
			d.Added++
		case opDel:
			d.Removed++
		}
	}
	d.Unified = renderUnified(ops, contextLines)
	return d
}

type opKind int

const (
	opKeep opKind = iota
	opAdd
	opDel
)

type op struct {
	kind opKind
	line string
}

// lcsDiff produces the edit script between two line slices via a longest
// common subsequence table.
func lcsDiff(a, b []string) []op {
	n, m := len(a), len(b)

	// table[i][j] = length of the LCS of a[i:] and b[j:]
	table := make([][]int, n+1)
	for i := range table {
		table[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				table[i][j] = table[i+1][j+1] + 1
			} else if table[i+1][j] >= table[i][j+1] {
				table[i][j] = table[i+1][j]
			} else {
				table[i][j] = table[i][j+1]
			}
		}
	}

	var ops []op
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case a[i] == b[j]:
			ops = append(ops, op{opKeep, a[i]})
			i++
			j++
		case table[i+1][j] >= table[i][j+1]:
			ops = append(ops, op{opDel, a[i]})
			i++
		default:
			ops = append(ops, op{opAdd, b[j]})
			j++
		}
	}
	for ; i < n; i++ {
		ops = append(ops, op{opDel, a[i]})
	}
	for ; j < m; j++ {
		ops = append(ops, op{opAdd, b[j]})
	}
	return ops
}

// renderUnified prints changed lines with up to contextLines of surrounding
// context, collapsing longer stretches of unchanged lines.
func renderUnified(ops []op, contextLines int) string {
	if contextLines < 0 {
		contextLines = 0
	}

	// Mark which unchanged lines are near a change and therefore worth showing.
	show := make([]bool, len(ops))
	for i, o := range ops {
		if o.kind == opKeep {
			continue
		}
		show[i] = true
		for d := 1; d <= contextLines; d++ {
			if i-d >= 0 {
				show[i-d] = true
			}
			if i+d < len(ops) {
				show[i+d] = true
			}
		}
	}

	var b strings.Builder
	skipped := 0
	flushSkip := func() {
		if skipped > 0 {
			fmt.Fprintf(&b, "      … %d unchanged line", skipped)
			if skipped != 1 {
				b.WriteString("s")
			}
			b.WriteString("\n")
			skipped = 0
		}
	}

	for i, o := range ops {
		if !show[i] {
			skipped++
			continue
		}
		flushSkip()
		switch o.kind {
		case opAdd:
			fmt.Fprintf(&b, "    + %s\n", o.line)
		case opDel:
			fmt.Fprintf(&b, "    - %s\n", o.line)
		default:
			fmt.Fprintf(&b, "      %s\n", o.line)
		}
	}
	flushSkip()
	return b.String()
}

// splitLines splits on newlines without inventing a trailing empty line, so a
// file with and without a final newline do not differ by a phantom line.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	s = strings.TrimSuffix(s, "\n")
	return strings.Split(s, "\n")
}
