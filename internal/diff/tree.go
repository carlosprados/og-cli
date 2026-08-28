package diff

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/carlosprados/og-cli/v2/internal/canon"
	"github.com/carlosprados/og-cli/v2/internal/unwrap"
)

// A workspace is not one artifact but a tree of them: a workspace holds
// dashboards, a dashboard holds widgets, and each level carries both metadata
// and — at the leaves — code.
//
// Compare() flattens everything into one payload, which works but reads badly:
// a changed widget formatter comes out as `dashboards[0].dashboard.grid[2].
// definition.config.columns[4]._formatterCode`, and a widget moved from one
// cell to another looks like every field of two widgets changed at once. The
// engine underneath is family-agnostic and the payloads canonicalize fine; what
// this adds is the shape.
//
// Nodes are matched by identity rather than by position, so reordering a
// dashboard's widgets reports two moves instead of a rewrite of both.

// Node is one level of the tree: the workspace, a dashboard, or a widget.
//
// Emitted under -o json, so it is a contract: additive changes only.
type Node struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
	ID   string `json:"id,omitempty"`

	// Status is added (local only), removed (remote only), changed, or
	// unchanged. A node whose own fields match but whose children differ is
	// still "changed": it is the path to something that moved.
	Status string `json:"status"`

	// Moved records a change of position among its siblings, which for a grid
	// is a real edit and not a diff artefact.
	Moved string `json:"moved,omitempty"`

	Metadata []Change   `json:"metadata,omitempty"`
	Code     []FileDiff `json:"code,omitempty"`
	Children []Node     `json:"children,omitempty"`
}

// Status values.
const (
	StatusAdded     = "added"
	StatusRemoved   = "removed"
	StatusChanged   = "changed"
	StatusUnchanged = "unchanged"
)

// Changed reports whether this node or anything under it differs.
func (n Node) Changed() bool {
	if n.Status != StatusUnchanged || n.Moved != "" {
		return true
	}
	for _, c := range n.Children {
		if c.Changed() {
			return true
		}
	}
	return false
}

// TreeResult is a hierarchical comparison, the counterpart of Result.
type TreeResult struct {
	Kind    string `json:"kind"`
	Dir     string `json:"dir"`
	State   string `json:"state"`
	Root    Node   `json:"root"`
	Origin  string `json:"origin,omitempty"`
	Ignored string `json:"ignored,omitempty"`
}

// Changed reports whether anything in the tree differs.
func (t TreeResult) Changed() bool { return t.Root.Changed() }

// Side is one side of a hierarchical comparison, built by the caller from
// whatever it has — a local directory tree or a set of API responses.
//
// The engine takes this rather than the client's types so that it stays
// testable without a platform and reusable if another family ever grows a
// hierarchy.
type Side struct {
	Kind     string
	Name     string
	ID       string
	Meta     json.RawMessage
	Children []Side
	// Code is the extracted source of a leaf, keyed by the filename the
	// unwrapped tree gives it.
	Code map[string]string
}

// CompareTree compares two hierarchies, remote as the "before" side and local
// as the "after", so the report reads as what deploying would do — the same
// direction as every other diff in og.
func CompareTree(dir string, remote, local Side, opts Options) (TreeResult, error) {
	root, err := compareSide(remote, local, opts)
	if err != nil {
		return TreeResult{}, err
	}
	return TreeResult{Kind: local.Kind, Dir: dir, Root: root}, nil
}

func compareSide(remote, local Side, opts Options) (Node, error) {
	n := Node{Kind: local.Kind, Name: local.Name, ID: local.ID, Status: StatusUnchanged}
	if n.Kind == "" {
		n.Kind = remote.Kind
	}
	if n.Name == "" {
		n.Name = remote.Name
	}
	if n.ID == "" {
		n.ID = remote.ID
	}

	canonOpts := canon.Options{Kind: unwrap.Kind(n.Kind), Scope: opts.Scope}
	remoteMeta, err := canonicalOrEmpty(remote.Meta, canonOpts)
	if err != nil {
		return n, err
	}
	localMeta, err := canonicalOrEmpty(local.Meta, canonOpts)
	if err != nil {
		return n, err
	}
	if n.Metadata, err = Structural(remoteMeta, localMeta); err != nil {
		return n, err
	}

	n.Code = compareCodeMaps(remote.Code, local.Code, opts.ContextLines)

	children, err := matchChildren(remote.Children, local.Children, opts)
	if err != nil {
		return n, err
	}
	n.Children = children

	if len(n.Metadata) > 0 || len(n.Code) > 0 {
		n.Status = StatusChanged
	} else {
		for _, c := range n.Children {
			if c.Changed() {
				n.Status = StatusChanged
				break
			}
		}
	}
	return n, nil
}

// matchChildren pairs children by identity, not by position.
//
// Position is reported separately, as a move. Matching by index instead would
// turn one widget inserted at the top of a dashboard into "every widget
// changed", which is exactly the output that makes a diff useless.
func matchChildren(remote, local []Side, opts Options) ([]Node, error) {
	remoteAt := map[string]int{}
	for i, c := range remote {
		remoteAt[childKey(c, i)] = i
	}
	seen := map[string]bool{}

	var out []Node
	for i, l := range local {
		key := childKey(l, i)
		seen[key] = true

		j, ok := remoteAt[key]
		if !ok {
			out = append(out, addedNode(l, opts))
			continue
		}
		node, err := compareSide(remote[j], l, opts)
		if err != nil {
			return nil, err
		}
		if j != i {
			// A move is a change: the deploy would reorder the grid. Saying
			// "unchanged" here and putting the fact in a separate field would
			// trap anyone filtering the JSON on status.
			node.Moved = fmt.Sprintf("position %d → %d", j, i)
			node.Status = StatusChanged
		}
		out = append(out, node)
	}

	// Anything the remote has and the local does not is a deletion, and a
	// deployment would perform it. Reporting it last keeps the local order
	// readable.
	for i, r := range remote {
		if seen[childKey(r, i)] {
			continue
		}
		out = append(out, removedNode(r, opts))
	}
	return out, nil
}

// childKey identifies a child for matching: its own identifier when it has one,
// otherwise its name, otherwise its position. Falling back to position is what
// keeps an unidentified child from being reported as one deletion plus one
// addition.
func childKey(s Side, index int) string {
	switch {
	case s.ID != "":
		return "id:" + s.ID
	case s.Name != "":
		return "name:" + s.Name
	default:
		return fmt.Sprintf("index:%d", index)
	}
}

func addedNode(s Side, opts Options) Node {
	n := Node{Kind: s.Kind, Name: s.Name, ID: s.ID, Status: StatusAdded}
	for _, c := range s.Children {
		n.Children = append(n.Children, addedNode(c, opts))
	}
	for _, name := range sortedCodeNames(s.Code) {
		n.Code = append(n.Code, FileDiff{File: name, OnlyIn: "local", Added: len(splitLines(s.Code[name]))})
	}
	return n
}

func removedNode(s Side, opts Options) Node {
	n := Node{Kind: s.Kind, Name: s.Name, ID: s.ID, Status: StatusRemoved}
	for _, c := range s.Children {
		n.Children = append(n.Children, removedNode(c, opts))
	}
	for _, name := range sortedCodeNames(s.Code) {
		n.Code = append(n.Code, FileDiff{File: name, OnlyIn: "remote", Removed: len(splitLines(s.Code[name]))})
	}
	return n
}

// canonicalOrEmpty canonicalises a payload, tolerating a node that carries none
// — a container in the tree with nothing of its own to compare.
func canonicalOrEmpty(raw json.RawMessage, opts canon.Options) ([]byte, error) {
	if len(raw) == 0 {
		return []byte("{}"), nil
	}
	return canon.Canonicalize(raw, opts)
}

// compareCodeMaps diffs two sets of extracted source files.
func compareCodeMaps(before, after map[string]string, contextLines int) []FileDiff {
	names := map[string]bool{}
	for n := range before {
		names[n] = true
	}
	for n := range after {
		names[n] = true
	}

	var out []FileDiff
	for _, name := range sortedSet(names) {
		b, inBefore := before[name]
		a, inAfter := after[name]
		switch {
		case inBefore && !inAfter:
			out = append(out, FileDiff{File: name, OnlyIn: "remote", Removed: len(splitLines(b))})
		case !inBefore && inAfter:
			out = append(out, FileDiff{File: name, OnlyIn: "local", Added: len(splitLines(a))})
		default:
			if fd := Text(name, b, a, contextLines); fd.Changed() {
				out = append(out, fd)
			}
		}
	}
	return out
}

// sortedCodeNames lists a leaf's source files in a stable order.
func sortedCodeNames(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedSet(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// ── rendering ────────────────────────────────────────────────────────────────

// RenderText writes the tree for a human.
//
// Unchanged branches are pruned: on a workspace of seven widgets where one
// formatter moved, printing the other six is noise that hides the answer. What
// is kept is the path — a dashboard whose own fields are identical still prints,
// because it is how the reader gets to the widget underneath.
func (t TreeResult) RenderText(nameOnly bool) string {
	var b strings.Builder
	renderNode(&b, t.Root, "", nameOnly)
	if t.Origin != "" {
		fmt.Fprintf(&b, "  pulled from %s\n", t.Origin)
	}
	if t.Ignored != "" {
		fmt.Fprintf(&b, "  %s\n", t.Ignored)
	}
	return b.String()
}

func renderNode(b *strings.Builder, n Node, indent string, nameOnly bool) {
	fmt.Fprintf(b, "%s%s %s %s", indent, n.marker(), n.Kind, n.Name)
	if n.ID != "" {
		fmt.Fprintf(b, "  (%s)", n.ID)
	}
	if n.Moved != "" {
		fmt.Fprintf(b, "  [%s]", n.Moved)
	}
	b.WriteString("\n")

	inner := indent + "  "
	if !nameOnly {
		if len(n.Metadata) > 0 {
			fmt.Fprintf(b, "%smetadata:\n", inner)
			b.WriteString(Render(n.Metadata, inner+"  "))
		}
		for _, f := range n.Code {
			switch f.OnlyIn {
			case "local":
				fmt.Fprintf(b, "%s%s  (local only, +%d lines)\n", inner, f.File, f.Added)
				continue
			case "remote":
				fmt.Fprintf(b, "%s%s  (remote only, -%d lines)\n", inner, f.File, f.Removed)
				continue
			}
			fmt.Fprintf(b, "%s%s  +%d −%d\n", inner, f.File, f.Added, f.Removed)
			b.WriteString(indentBlock(f.Unified, inner))
		}
	}

	for _, c := range n.Children {
		if !c.Changed() {
			continue
		}
		renderNode(b, c, inner, nameOnly)
	}
}

// marker keeps the vocabulary of the flat diff: the same characters mean the
// same things, so a reader does not need two legends.
func (n Node) marker() string {
	switch n.Status {
	case StatusAdded:
		return "+"
	case StatusRemoved:
		return "−"
	case StatusChanged:
		return "~"
	default:
		return " "
	}
}

func indentBlock(s, indent string) string {
	if s == "" {
		return ""
	}
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	var b strings.Builder
	for _, l := range lines {
		fmt.Fprintf(&b, "%s%s\n", indent, l)
	}
	return b.String()
}
