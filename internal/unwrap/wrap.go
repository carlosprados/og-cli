package unwrap

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/carlosprados/og-cli/pkg/opengate"
)

// Wrap reconstructs a workspace from a previously-unwrapped directory tree.
// It reads workspace.json, walks each dashboard sub-directory in alphabetical
// order (which equals the original grid order thanks to the NN prefix), and
// re-inflates widget JS files into their config fields.
//
// The returned Workspace has every dashboard fully populated (grid included)
// and can be re-serialized for /api/workspaces import.
func Wrap(dir string) (*opengate.Workspace, error) {
	wsPath := filepath.Join(dir, "workspace.json")
	wsRaw, err := os.ReadFile(wsPath)
	if err != nil {
		return nil, fmt.Errorf("reading workspace.json: %w", err)
	}

	var ws opengate.Workspace
	if err := json.Unmarshal(wsRaw, &ws); err != nil {
		return nil, fmt.Errorf("parsing workspace.json: %w", err)
	}

	dashEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("listing workspace dir: %w", err)
	}

	var dashboards []opengate.WorkspaceDashboard
	for _, e := range dashEntries {
		if !e.IsDir() {
			continue
		}
		wd, err := wrapDashboard(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("dashboard %s: %w", e.Name(), err)
		}
		dashboards = append(dashboards, wd)
	}
	ws.Dashboards = dashboards
	return &ws, nil
}

// WrapDashboard reconstructs a full opengate.Dashboard (with grid + widget
// configs) from a dashboard directory, plus the WorkspaceDashboard layout
// entry that wraps it inside its parent workspace.
func WrapDashboard(dir string) (*opengate.Dashboard, *opengate.WorkspaceDashboard, error) {
	wd, err := wrapDashboard(dir)
	if err != nil {
		return nil, nil, err
	}

	// Reconstruct the full Dashboard from disk by reading dashboard.json again
	// (wrapDashboard only consumes the layout/grid; we need the body fields).
	full, err := readFullDashboard(dir)
	if err != nil {
		return nil, nil, err
	}

	full.Grid = collectGrid(dir)

	return full, &wd, nil
}

// wrapDashboard reads dashboard.json + each widget folder, and returns a
// WorkspaceDashboard with the simplified body (grid included) and layout
// in place.
func wrapDashboard(dir string) (opengate.WorkspaceDashboard, error) {
	dashPath := filepath.Join(dir, "dashboard.json")
	raw, err := os.ReadFile(dashPath)
	if err != nil {
		return opengate.WorkspaceDashboard{}, fmt.Errorf("reading dashboard.json: %w", err)
	}

	// Decode into a generic map so we can split out the _workspaceLayout side
	// channel that unwrap inserted.
	var generic map[string]json.RawMessage
	if err := json.Unmarshal(raw, &generic); err != nil {
		return opengate.WorkspaceDashboard{}, fmt.Errorf("parsing dashboard.json: %w", err)
	}

	var layout opengate.WorkspaceDashboard
	if layoutRaw, ok := generic["_workspaceLayout"]; ok {
		if err := json.Unmarshal(layoutRaw, &layout); err != nil {
			return opengate.WorkspaceDashboard{}, fmt.Errorf("parsing _workspaceLayout: %w", err)
		}
		delete(generic, "_workspaceLayout")
	}

	bodyRaw, err := json.Marshal(generic)
	if err != nil {
		return opengate.WorkspaceDashboard{}, fmt.Errorf("re-encoding dashboard body: %w", err)
	}
	var simplified opengate.DashboardSimplified
	if err := json.Unmarshal(bodyRaw, &simplified); err != nil {
		return opengate.WorkspaceDashboard{}, fmt.Errorf("decoding dashboard body: %w", err)
	}

	// Reassemble the grid from the widget sub-folders. This is the inverse of
	// UnwrapDashboardFull which separates each grid item into its own folder.
	simplified.Grid = collectGrid(dir)

	layout.Dashboard = &simplified
	if layout.ID == "" {
		layout.ID = simplified.ID
	}
	return layout, nil
}

// readFullDashboard reads dashboard.json as a full opengate.Dashboard (with
// grid populated by collectGrid).
func readFullDashboard(dir string) (*opengate.Dashboard, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "dashboard.json"))
	if err != nil {
		return nil, err
	}
	var generic map[string]json.RawMessage
	if err := json.Unmarshal(raw, &generic); err != nil {
		return nil, fmt.Errorf("parsing dashboard.json: %w", err)
	}
	delete(generic, "_workspaceLayout")
	bodyRaw, err := json.Marshal(generic)
	if err != nil {
		return nil, err
	}
	var d opengate.Dashboard
	if err := json.Unmarshal(bodyRaw, &d); err != nil {
		return nil, fmt.Errorf("decoding dashboard: %w", err)
	}
	return &d, nil
}

// collectGrid walks widget sub-directories in alphabetical order and rebuilds
// each GridItem, re-injecting any .js files back into the config.
func collectGrid(dir string) []opengate.GridItem {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	var grid []opengate.GridItem
	for _, name := range names {
		item, err := wrapWidget(filepath.Join(dir, name))
		if err != nil {
			// Skip malformed widget directories rather than fail the whole wrap.
			continue
		}
		grid = append(grid, *item)
	}
	return grid
}

// wrapWidget reads widget.json + every <field>.js sibling and produces a
// GridItem ready for inclusion in a Dashboard.Grid.
func wrapWidget(dir string) (*opengate.GridItem, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "widget.json"))
	if err != nil {
		return nil, err
	}
	var item opengate.GridItem
	if err := json.Unmarshal(raw, &item); err != nil {
		return nil, fmt.Errorf("parsing widget.json: %w", err)
	}
	if item.Definition == nil {
		return &item, nil
	}

	// Load all .js files in the widget directory.
	entries, err := os.ReadDir(dir)
	if err != nil {
		return &item, nil
	}
	jsFiles := make(map[string]string)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".js") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		jsFiles[e.Name()] = string(content)
	}

	if len(jsFiles) == 0 {
		return &item, nil
	}

	// Decode the cleaned config, re-inject every .js back at its keypath, then
	// re-encode and put it back on the definition.
	var configTree any
	if len(item.Definition.Config) > 0 {
		if err := json.Unmarshal(item.Definition.Config, &configTree); err != nil {
			return nil, fmt.Errorf("decoding widget config: %w", err)
		}
	}
	if configTree == nil {
		configTree = map[string]any{}
	}
	rebuilt := ReinjectJSFields(configTree, jsFiles)
	finalRaw, err := json.Marshal(rebuilt)
	if err != nil {
		return nil, fmt.Errorf("encoding rebuilt config: %w", err)
	}
	item.Definition.Config = finalRaw
	return &item, nil
}
