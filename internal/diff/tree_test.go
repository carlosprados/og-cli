package diff

import (
	"encoding/json"
	"strings"
	"testing"
)

func widget(id, name, code string, meta map[string]any) Side {
	raw, _ := json.Marshal(meta)
	return Side{Kind: "widget", ID: id, Name: name, Meta: raw, Code: map[string]string{"formatter.js": code}}
}

func tree(children ...Side) Side {
	raw, _ := json.Marshal(map[string]any{"name": "ws"})
	dash, _ := json.Marshal(map[string]any{"title": "d"})
	return Side{Kind: "workspace", Name: "ws", ID: "w1", Meta: raw, Children: []Side{
		{Kind: "dashboard", Name: "d", ID: "d1", Meta: dash, Children: children},
	}}
}

func compare(t *testing.T, remote, local Side) TreeResult {
	t.Helper()
	r, err := CompareTree("dir", remote, local, Options{ContextLines: 1})
	if err != nil {
		t.Fatalf("CompareTree: %v", err)
	}
	return r
}

func TestTreeReportsNoDifferenceOnIdenticalTrees(t *testing.T) {
	s := tree(widget("a", "00__a", "var x = 1;", map[string]any{"x": 0}))
	r := compare(t, s, s)
	if r.Changed() {
		t.Errorf("identical trees reported as changed:\n%s", r.RenderText(false))
	}
}

// The reason this renderer exists: a widget moved in the grid must be one move,
// not two rewrites.
func TestTreeMatchesByIdentitySoAReorderIsAMove(t *testing.T) {
	a := widget("a", "a", "var x = 1;", map[string]any{"x": 0})
	b := widget("b", "b", "var y = 2;", map[string]any{"x": 1})

	r := compare(t, tree(a, b), tree(b, a))
	kids := r.Root.Children[0].Children
	if len(kids) != 2 {
		t.Fatalf("got %d widgets, want 2", len(kids))
	}
	for _, k := range kids {
		if k.Moved == "" {
			t.Errorf("widget %s not reported as moved", k.ID)
		}
		if len(k.Metadata) > 0 || len(k.Code) > 0 {
			t.Errorf("widget %s reported content changes on a pure reorder: %+v %+v", k.ID, k.Metadata, k.Code)
		}
	}
}

func TestTreeReportsAddedRemovedAndChanged(t *testing.T) {
	before := tree(
		widget("keep", "keep", "var x = 1;", map[string]any{"x": 0}),
		widget("gone", "gone", "var g = 1;", map[string]any{"x": 1}),
	)
	after := tree(
		widget("keep", "keep", "var x = 2;", map[string]any{"x": 0}),
		widget("new", "new", "var n = 1;", map[string]any{"x": 1}),
	)

	r := compare(t, before, after)
	got := map[string]string{}
	for _, k := range r.Root.Children[0].Children {
		got[k.ID] = k.Status
	}
	want := map[string]string{"keep": StatusChanged, "new": StatusAdded, "gone": StatusRemoved}
	for id, status := range want {
		if got[id] != status {
			t.Errorf("widget %s is %q, want %q", id, got[id], status)
		}
	}

	// A changed leaf must make its ancestors changed too, or the reader has no
	// path to it.
	if r.Root.Status != StatusChanged || r.Root.Children[0].Status != StatusChanged {
		t.Errorf("ancestors not marked changed: workspace=%q dashboard=%q",
			r.Root.Status, r.Root.Children[0].Status)
	}
}

// Unchanged branches are pruned, but the path to a change is kept.
func TestRenderPrunesUnchangedBranches(t *testing.T) {
	before := tree(
		widget("quiet", "quiet-one", "var q = 1;", map[string]any{"x": 0}),
		widget("loud", "loud-one", "var l = 1;", map[string]any{"x": 1}),
	)
	after := tree(
		widget("quiet", "quiet-one", "var q = 1;", map[string]any{"x": 0}),
		widget("loud", "loud-one", "var l = 999;", map[string]any{"x": 1}),
	)

	out := compare(t, before, after).RenderText(false)
	if strings.Contains(out, "quiet-one") {
		t.Errorf("unchanged widget rendered:\n%s", out)
	}
	if !strings.Contains(out, "loud-one") {
		t.Errorf("changed widget missing:\n%s", out)
	}
	// The dashboard is unchanged in itself but is the path to the widget.
	if !strings.Contains(out, "dashboard d") {
		t.Errorf("path to the change missing:\n%s", out)
	}
}

// Direction is remote → local, so the report reads as what deploying would do.
func TestTreeDirectionIsWhatDeployWouldDo(t *testing.T) {
	remote := tree(widget("a", "a", "var x = 1;", map[string]any{"x": 0}))
	local := tree(widget("a", "a", "var x = 2;", map[string]any{"x": 0}))

	out := compare(t, remote, local).RenderText(false)
	if !strings.Contains(out, "- var x = 1;") || !strings.Contains(out, "+ var x = 2;") {
		t.Errorf("expected remote as the before side and local as the after:\n%s", out)
	}
}

// A node with no payload of its own must not break the comparison: the tree has
// containers that carry only children.
func TestTreeToleratesNodesWithoutMetadata(t *testing.T) {
	side := Side{Kind: "workspace", Name: "ws", Children: []Side{{Kind: "dashboard", Name: "d"}}}
	r := compare(t, side, side)
	if r.Changed() {
		t.Errorf("empty tree reported as changed:\n%s", r.RenderText(false))
	}
}
