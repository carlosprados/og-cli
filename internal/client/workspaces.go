package client

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	workspacesPath      = "/api/workspaces"
	workspacesListPath  = "/api/workspaces/"
	workspacePath       = "/api/workspaces/%s"
	workspaceExportPath = "/api/workspaces/export/%s"
)

// Workspace represents an OpenGate Web API workspace. A workspace groups
// dashboards together and is the top-level container in the UI configuration
// hierarchy (workspace 1 → N dashboards).
type Workspace struct {
	ID              string                           `json:"_id,omitempty"`
	Name            string                           `json:"name"`
	Description     *string                          `json:"description,omitempty"`
	Owner           string                           `json:"owner,omitempty"`
	Image           *string                          `json:"image,omitempty"`
	Icon            string                           `json:"icon,omitempty"`
	Users           []string                         `json:"users,omitempty"`
	Domains         []string                         `json:"domains,omitempty"`
	Workgroups      []string                         `json:"workgroups,omitempty"`
	Actions         []string                         `json:"actions,omitempty"`
	Widgets         []string                         `json:"widgets,omitempty"`
	WidgetAction    map[string]string                `json:"widget_action,omitempty"`
	AllowedProfiles []string                         `json:"allowedProfiles,omitempty"`
	Dashboards      []WorkspaceDashboard             `json:"dashboards,omitempty"`
	Priority        int                              `json:"priority,omitempty"`
	Color           string                           `json:"color,omitempty"`
	LastAccess      string                           `json:"lastAccess,omitempty"`
	Editable        *bool                            `json:"editable,omitempty"`
	Version         int                              `json:"__v,omitempty"`
	EditMode        *bool                            `json:"_editMode,omitempty"`
	Others          *WorkspaceOthers                 `json:"others,omitempty"`
	Menu            []WorkspaceMenuItem              `json:"menu,omitempty"`
	MenuTree        map[string]WorkspaceMenuCategory `json:"menu_tree,omitempty"`
}

// WorkspaceOthers holds workspace display options.
type WorkspaceOthers struct {
	ShowInHome  bool    `json:"showInHome,omitempty"`
	Mode        string  `json:"mode,omitempty"`
	BannerImage *string `json:"bannerImage,omitempty"`
}

// WorkspaceDashboard is a dashboard embedded inside a workspace (with grid layout).
type WorkspaceDashboard struct {
	X         int                  `json:"x"`
	Y         int                  `json:"y"`
	Width     int                  `json:"width,omitempty"`
	Height    int                  `json:"height,omitempty"`
	W         int                  `json:"w"`
	H         int                  `json:"h"`
	I         string               `json:"i,omitempty"`
	Moved     bool                 `json:"moved,omitempty"`
	ID        string               `json:"id,omitempty"`
	MongoID   string               `json:"_id,omitempty"`
	Dashboard *DashboardSimplified `json:"dashboard,omitempty"`
}

// WorkspaceMenuItem is an action entry in the workspace side menu.
type WorkspaceMenuItem struct {
	Title      string            `json:"title,omitempty"`
	Action     string            `json:"action,omitempty"`
	Permission string            `json:"permission,omitempty"`
	Icon       string            `json:"icon,omitempty"`
	Menu       string            `json:"menu,omitempty"`
	MenuCfg    *WorkspaceMenuCfg `json:"menuCfg,omitempty"`
}

// WorkspaceMenuCfg describes the parent menu group config.
type WorkspaceMenuCfg struct {
	Icon  string `json:"icon,omitempty"`
	Title string `json:"title,omitempty"`
}

// WorkspaceMenuCategory groups menu items under a category.
type WorkspaceMenuCategory struct {
	Config  *WorkspaceMenuCfg   `json:"config,omitempty"`
	Actions []WorkspaceMenuItem `json:"actions,omitempty"`
}

// ListWorkspaces returns all workspaces accessible to the current user.
// When full is true, the response includes embedded dashboards (?full=1).
func (c *Client) ListWorkspaces(full bool) ([]Workspace, error) {
	path := workspacesListPath
	if full {
		path += "?full=1"
	}

	data, statusCode, err := c.WebGet(path)
	if err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	if IsEmptyResponse(data, statusCode) {
		return nil, nil
	}

	var ws []Workspace
	if err := json.Unmarshal(data, &ws); err != nil {
		return nil, fmt.Errorf("parsing workspaces: %w", err)
	}
	return ws, nil
}

// GetWorkspace retrieves a single workspace by ID. When full is true, embedded
// dashboards are included (?full=1).
func (c *Client) GetWorkspace(id string, full bool) (*Workspace, error) {
	path := fmt.Sprintf(workspacePath, id)
	if full {
		path += "?full=1"
	}

	data, statusCode, err := c.WebGet(path)
	if err != nil {
		return nil, fmt.Errorf("get workspace: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}

	var w Workspace
	if err := json.Unmarshal(data, &w); err != nil {
		return nil, fmt.Errorf("parsing workspace: %w", err)
	}
	return &w, nil
}

// ExportWorkspace fetches the export payload for a workspace as raw JSON.
// Use this for backups or migrations; the returned bytes can be passed back
// to ImportWorkspace on a different tenant.
func (c *Client) ExportWorkspace(id string) ([]byte, error) {
	path := fmt.Sprintf(workspaceExportPath, id)

	data, statusCode, err := c.WebGet(path)
	if err != nil {
		return nil, fmt.Errorf("export workspace: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	return data, nil
}

// CreateWorkspace posts a workspace definition. The body is the full JSON
// (typically produced by ExportWorkspace).
func (c *Client) CreateWorkspace(body json.RawMessage) ([]byte, error) {
	data, statusCode, err := c.WebPost(workspacesPath, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("create workspace: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	return data, nil
}

// UpdateWorkspace updates an existing workspace.
func (c *Client) UpdateWorkspace(id string, body json.RawMessage) error {
	path := fmt.Sprintf(workspacePath, id)

	data, statusCode, err := c.WebPut(path, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("update workspace: %w", err)
	}
	return CheckResponse(data, statusCode)
}

// DeleteWorkspace deletes a workspace by ID.
func (c *Client) DeleteWorkspace(id string) error {
	path := fmt.Sprintf(workspacePath, id)

	data, statusCode, err := c.WebDelete(path)
	if err != nil {
		return fmt.Errorf("delete workspace: %w", err)
	}
	return CheckResponse(data, statusCode)
}

// ImportWorkspaceDeep replays the OpenGate web UI's import-wizard flow:
//
//  1. POST /api/workspaces with the workspace shell (no dashboards inline)
//  2. POST /api/dashboards once per dashboard (full body with grid + widgets)
//  3. PUT  /api/workspaces/{id} with the shell + dashboards[] as layout refs
//
// Single-shot POST /api/workspaces persists the shell but discards the
// dashboard bodies that ride inside the array, so workspace import has to be
// done in this multi-phase manner.
func (c *Client) ImportWorkspaceDeep(w *Workspace) error {
	if w == nil {
		return fmt.Errorf("workspace is nil")
	}
	if w.ID == "" {
		return fmt.Errorf("workspace must have _id set")
	}

	shellRaw, err := workspaceShellJSON(w)
	if err != nil {
		return err
	}
	if _, err := c.CreateWorkspace(shellRaw); err != nil {
		return fmt.Errorf("creating workspace shell: %w", err)
	}

	for i := range w.Dashboards {
		dash := w.Dashboards[i].Dashboard
		if dash == nil {
			continue
		}
		dashRaw, err := json.Marshal(dash)
		if err != nil {
			return fmt.Errorf("marshaling dashboard %s: %w", dash.ID, err)
		}
		if _, err := c.CreateDashboard(dashRaw, ""); err != nil {
			return fmt.Errorf("creating dashboard %s: %w", dash.ID, err)
		}
	}

	return c.putWorkspaceWithLayoutRefs(w)
}

// UpdateWorkspaceDeep is the symmetric in-place variant for re-deploying an
// existing workspace (post-edit cycle):
//
//  1. PUT /api/dashboards/{id} for every dashboard with its current body
//  2. PUT /api/workspaces/{id} with the shell + dashboards[] as layout refs
func (c *Client) UpdateWorkspaceDeep(w *Workspace) error {
	if w == nil {
		return fmt.Errorf("workspace is nil")
	}
	if w.ID == "" {
		return fmt.Errorf("workspace must have _id set")
	}

	for i := range w.Dashboards {
		dash := w.Dashboards[i].Dashboard
		if dash == nil {
			continue
		}
		dashRaw, err := json.Marshal(dash)
		if err != nil {
			return fmt.Errorf("marshaling dashboard %s: %w", dash.ID, err)
		}
		if err := c.UpdateDashboard(dash.ID, dashRaw); err != nil {
			return fmt.Errorf("updating dashboard %s: %w", dash.ID, err)
		}
	}

	return c.putWorkspaceWithLayoutRefs(w)
}

// putWorkspaceWithLayoutRefs performs the final PUT of the workspace with the
// dashboards[] array reduced to grid layout references (no inline body).
func (c *Client) putWorkspaceWithLayoutRefs(w *Workspace) error {
	wsForPut := *w
	wsForPut.Dashboards = make([]WorkspaceDashboard, len(w.Dashboards))
	for i, wd := range w.Dashboards {
		layout := wd
		layout.Dashboard = nil // strip body — only the grid layout ref remains
		if layout.MongoID == "" && wd.Dashboard != nil {
			layout.MongoID = wd.Dashboard.ID
		}
		if layout.ID == "" && wd.Dashboard != nil {
			layout.ID = wd.Dashboard.ID
		}
		wsForPut.Dashboards[i] = layout
	}

	body, err := json.Marshal(wsForPut)
	if err != nil {
		return fmt.Errorf("marshaling workspace PUT body: %w", err)
	}
	return c.UpdateWorkspace(w.ID, body)
}

// workspaceShellJSON returns the workspace JSON with the dashboards field
// removed entirely — matches the POST /api/workspaces body the UI wizard
// sends in phase 1.
func workspaceShellJSON(w *Workspace) ([]byte, error) {
	shell := *w
	shell.Dashboards = nil
	return json.Marshal(shell)
}
