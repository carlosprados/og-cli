package unwrap

import (
	"encoding/json"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/carlosprados/og-cli/v2/pkg/opengate"
)

// The remote side of a dashboard's code, addressed the way the local tree
// addresses it.
//
// `og <family> show --path` was written for the flat families, where a
// descriptor says which field holds the code and there is one document to read
// it from. A dashboard is a tree, so the same question — "give me the current
// content of this one file" — needs the tree walked instead. This is that walk,
// and it produces exactly the paths `og workspace pull` writes: an editor can
// send the path it has on disk and get back the file it names.

// DashboardCodeFiles returns every code file a dashboard carries, keyed by the
// path a pull writes it to: "<widget-dir>/<file>".
//
// The extraction is unwrapWidget's, without touching disk. Sharing it is the
// point: a second implementation would drift, and the names would stop matching
// the tree the editor is working in.
func DashboardCodeFiles(d *opengate.Dashboard) (map[string]string, error) {
	if d == nil {
		return nil, fmt.Errorf("dashboard is nil")
	}

	out := make(map[string]string)
	width := max(len(strconv.Itoa(len(d.Grid)-1)), 2)
	for i, item := range d.Grid {
		var configTree any
		if item.Definition != nil && len(item.Definition.Config) > 0 {
			if err := json.Unmarshal(item.Definition.Config, &configTree); err != nil {
				return nil, fmt.Errorf("widget %d (%s): decoding config: %w", i, item.I, err)
			}
		}
		_, files, _ := WidgetContract().Extract(configTree, nil)

		dir := widgetSlug(i, item, width)
		for name, code := range files {
			out[path.Join(dir, name)] = code
		}
	}
	return out, nil
}

// ResolveCodePath finds a requested path among a dashboard's code files,
// matching the widget by identity rather than by its position in the grid.
//
// The directory name a pull writes carries the widget's index — "01__type__wid"
// — and that index is the remote grid order at the moment of the pull. Reorder
// the dashboard on the platform and every path in the local tree names a folder
// that no longer exists, so an editor asking for the file it is looking at
// would be told there is no such file. og already matches widgets by identity
// rather than by position in `og workspace diff`, for the same reason; here the
// index is treated as decoration.
//
// An exact hit always wins. Otherwise the index prefix is dropped from both
// sides, and the match has to be unambiguous: a widget with no id of its own
// slugs to its type alone, and two of those would be indistinguishable. Serving
// the wrong file silently is worse than reporting that the path was not found.
func ResolveCodePath(files map[string]string, requested string) (string, bool) {
	requested = strings.ReplaceAll(requested, "\\", "/")
	if content, ok := files[requested]; ok {
		return content, true
	}

	want := stripIndexPrefix(requested)
	var found string
	matches := 0
	for name, content := range files {
		if stripIndexPrefix(name) == want {
			found, matches = content, matches+1
		}
	}
	if matches != 1 {
		return "", false
	}
	return found, true
}

// stripIndexPrefix drops the "NN__" ordering prefix from a path's first
// segment, leaving the widget's identity.
func stripIndexPrefix(p string) string {
	dir, rest, ok := strings.Cut(p, "/")
	if !ok {
		return p
	}
	if i := strings.Index(dir, "__"); i > 0 && allDigits(dir[:i]) {
		dir = dir[i+len("__"):]
	}
	return dir + "/" + rest
}

func allDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}
