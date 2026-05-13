package unwrap

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/carlosprados/og-cli/internal/client"
)

// Unwrap creates the workspace root directory and writes workspace.json
// (workspace metadata with the dashboards array stripped). Each dashboard is
// then written by the caller via UnwrapDashboardFull into a sibling sub-folder
// — typically named NN__<dash-slug> to preserve the array order.
//
// Final tree:
//
//	<dir>/
//	  workspace.json           — workspace metadata, dashboards array removed
//	  <NN>__<dash-slug>/
//	    dashboard.json         — dashboard metadata, grid array removed
//	    <NN>__<type>__<wid>/   — one folder per widget (NN preserves grid order)
//	      widget.json          — full grid item, JS fields removed
//	      <field>.js           — one file per extracted JS string
func Unwrap(w *client.Workspace, dir string) (string, error) {
	if w == nil {
		return "", fmt.Errorf("workspace is nil")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating root dir: %w", err)
	}

	wsClone := *w
	wsClone.Dashboards = nil
	if err := writeJSON(filepath.Join(dir, "workspace.json"), wsClone); err != nil {
		return "", err
	}
	return dir, nil
}

// UnwrapDashboardFull unwraps a full Dashboard (with grid+widgets) into the
// given dashboard directory. Use it when you have the full Dashboard struct,
// typically fetched via client.GetDashboard.
func UnwrapDashboardFull(d *client.Dashboard, layout *client.WorkspaceDashboard, dir string) error {
	if d == nil {
		return fmt.Errorf("dashboard is nil")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating dashboard dir: %w", err)
	}

	// dashboard.json — strip grid; preserve workspace layout context if given.
	dClone := *d
	dClone.Grid = nil
	type dashboardOut struct {
		*client.Dashboard
		WorkspaceLayout *client.WorkspaceDashboard `json:"_workspaceLayout,omitempty"`
	}
	out := dashboardOut{Dashboard: &dClone, WorkspaceLayout: layout}
	if err := writeJSON(filepath.Join(dir, "dashboard.json"), out); err != nil {
		return err
	}

	// Widgets: NN prefix preserves the grid order across filesystem listing.
	width := max(len(strconv.Itoa(len(d.Grid)-1)), 2)
	for i, item := range d.Grid {
		name := widgetSlug(i, item, width)
		widgetDir := filepath.Join(dir, name)
		if err := unwrapWidget(&item, widgetDir); err != nil {
			return fmt.Errorf("widget %d (%s): %w", i, name, err)
		}
	}
	return nil
}

func unwrapWidget(item *client.GridItem, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating widget dir: %w", err)
	}

	// Decode config into a generic tree, run JS extraction, then re-encode.
	var configTree any
	if item.Definition != nil && len(item.Definition.Config) > 0 {
		if err := json.Unmarshal(item.Definition.Config, &configTree); err != nil {
			return fmt.Errorf("decoding widget config: %w", err)
		}
	}
	cleaned, jsFiles := ExtractJSFields(configTree)

	// Write each JS file.
	for filename, code := range jsFiles {
		path := filepath.Join(dir, filename)
		if err := os.WriteFile(path, []byte(code), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", filename, err)
		}
	}

	// Build widget.json with cleaned config.
	itemClone := *item
	if item.Definition != nil {
		defClone := *item.Definition
		cleanedRaw, err := json.Marshal(cleaned)
		if err != nil {
			return fmt.Errorf("encoding cleaned config: %w", err)
		}
		defClone.Config = cleanedRaw
		itemClone.Definition = &defClone
	}

	return writeJSON(filepath.Join(dir, "widget.json"), itemClone)
}

// widgetSlug builds the widget folder name: "<NN>__<type>__<wid>".
func widgetSlug(index int, item client.GridItem, width int) string {
	prefix := fmt.Sprintf("%0*d", width, index)
	wtype := "Widget"
	wid := item.I
	if item.Definition != nil {
		if item.Definition.Type != "" {
			wtype = item.Definition.Type
		}
		if item.Definition.Wid != "" {
			wid = item.Definition.Wid
		}
	}
	parts := []string{prefix, Slugify(wtype)}
	if wid != "" {
		parts = append(parts, Slugify(wid))
	}
	return strings.Join(parts, "__")
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling %s: %w", filepath.Base(path), err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

// ListWorkspaceDirs returns the subdirectories of dir that contain a
// workspace.json — useful when wrap-all walks a parent directory.
func ListWorkspaceDirs(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var result []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, e.Name(), "workspace.json")); err == nil {
			result = append(result, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(result)
	return result, nil
}

