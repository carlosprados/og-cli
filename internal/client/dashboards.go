package client

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	dashboardsPath      = "/api/dashboards"
	dashboardPath       = "/api/dashboards/%s"
	dashboardExportPath = "/api/dashboards/export/%s"
)

// Dashboard represents a full OpenGate Web API dashboard. Every dashboard
// belongs to exactly one workspace, referenced by the Workspaces field.
type Dashboard struct {
	ID              string                `json:"_id,omitempty"`
	AltID           string                `json:"id,omitempty"`
	Title           string                `json:"title"`
	Description     *string               `json:"description,omitempty"`
	Icon            string                `json:"icon,omitempty"`
	IconType        string                `json:"iconType,omitempty"`
	Owner           string                `json:"owner,omitempty"`
	Workspaces      string                `json:"workspaces,omitempty"`
	Users           []string              `json:"users,omitempty"`
	Workgroups      []string              `json:"workgroups,omitempty"`
	AllowedProfiles []string              `json:"allowedProfiles,omitempty"`
	Domains         []string              `json:"domains,omitempty"`
	LastAccess      string                `json:"lastAccess,omitempty"`
	Editable        *bool                 `json:"editable,omitempty"`
	BackgroundImage *string               `json:"backgroundImage,omitempty"`
	BannerImage     *string               `json:"bannerImage,omitempty"`
	Version         int                   `json:"__v,omitempty"`
	ExtraConfig     *DashboardExtraConfig `json:"extraConfig,omitempty"`
	Grid            []GridItem            `json:"grid,omitempty"`
	TemplateConfig  json.RawMessage       `json:"templateConfig,omitempty"`
}

// DashboardSimplified is the dashboard payload returned inside a workspace's
// embedded dashboards array. Some endpoints omit the grid (workspaces?full=1),
// others include it (workspaces/export/{id}). Grid is therefore optional.
type DashboardSimplified struct {
	ID              string                `json:"_id,omitempty"`
	AltID           string                `json:"id,omitempty"`
	Title           string                `json:"title"`
	Description     *string               `json:"description,omitempty"`
	Icon            string                `json:"icon,omitempty"`
	IconType        string                `json:"iconType,omitempty"`
	Owner           string                `json:"owner,omitempty"`
	Workspaces      string                `json:"workspaces,omitempty"`
	Users           []string              `json:"users,omitempty"`
	Workgroups      []string              `json:"workgroups,omitempty"`
	AllowedProfiles []string              `json:"allowedProfiles,omitempty"`
	Domains         []string              `json:"domains,omitempty"`
	LastAccess      string                `json:"lastAccess,omitempty"`
	Editable        *bool                 `json:"editable,omitempty"`
	BackgroundImage *string               `json:"backgroundImage,omitempty"`
	BannerImage     *string               `json:"bannerImage,omitempty"`
	Version         int                   `json:"__v,omitempty"`
	ExtraConfig     *DashboardExtraConfig `json:"extraConfig,omitempty"`
	Grid            []GridItem            `json:"grid,omitempty"`
	TemplateConfig  json.RawMessage       `json:"templateConfig,omitempty"`
}

// DashboardExtraConfig holds display options for a dashboard.
type DashboardExtraConfig struct {
	CellsWidth               string `json:"cellsWidth,omitempty"`
	CellHeight               int    `json:"cellHeight,omitempty"`
	DashboardRefreshInterval string `json:"dashboardRefreshInterval,omitempty"`
	ShowBanner               bool   `json:"showBanner,omitempty"`
	Favourite                bool   `json:"favourite,omitempty"`
}

// GridItem is a single cell in the dashboard grid, holding one widget.
type GridItem struct {
	Width      int               `json:"width,omitempty"`
	Height     int               `json:"height,omitempty"`
	X          int               `json:"x"`
	Y          int               `json:"y"`
	W          int               `json:"w"`
	H          int               `json:"h"`
	I          string            `json:"i,omitempty"`
	Moved      bool              `json:"moved,omitempty"`
	Definition *WidgetDefinition `json:"definition,omitempty"`
}

// WidgetDefinition describes the widget rendered in a grid cell.
type WidgetDefinition struct {
	Type   string          `json:"type,omitempty"`
	Wid    string          `json:"wid,omitempty"`
	Config json.RawMessage `json:"config,omitempty"`
}

// GetDashboard retrieves a single dashboard by ID.
func (c *Client) GetDashboard(id string) (*Dashboard, error) {
	path := fmt.Sprintf(dashboardPath, id)

	data, statusCode, err := c.WebGet(path)
	if err != nil {
		return nil, fmt.Errorf("get dashboard: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}

	var d Dashboard
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, fmt.Errorf("parsing dashboard: %w", err)
	}
	return &d, nil
}

// ExportDashboard fetches the export payload for a dashboard as raw JSON.
func (c *Client) ExportDashboard(id string) ([]byte, error) {
	path := fmt.Sprintf(dashboardExportPath, id)

	data, statusCode, err := c.WebGet(path)
	if err != nil {
		return nil, fmt.Errorf("export dashboard: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	return data, nil
}

// CreateDashboard posts a dashboard definition. If workspaceOverride is
// non-empty, the "workspaces" field of the body is replaced with that value
// before sending — useful for cross-tenant migrations.
func (c *Client) CreateDashboard(body json.RawMessage, workspaceOverride string) ([]byte, error) {
	payload, err := applyWorkspaceOverride(body, workspaceOverride)
	if err != nil {
		return nil, fmt.Errorf("create dashboard: %w", err)
	}

	data, statusCode, err := c.WebPost(dashboardsPath, strings.NewReader(string(payload)))
	if err != nil {
		return nil, fmt.Errorf("create dashboard: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	return data, nil
}

// UpdateDashboard updates an existing dashboard.
func (c *Client) UpdateDashboard(id string, body json.RawMessage) error {
	path := fmt.Sprintf(dashboardPath, id)

	data, statusCode, err := c.WebPut(path, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("update dashboard: %w", err)
	}
	return CheckResponse(data, statusCode)
}

// DeleteDashboard deletes a dashboard by ID. The Web API exposes DELETE
// /dashboards with the id in the body, so we send a minimal payload.
func (c *Client) DeleteDashboard(id string) error {
	body := fmt.Sprintf(`{"_id":%q}`, id)

	data, statusCode, err := c.webDoRequest("DELETE", dashboardsPath, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("delete dashboard: %w", err)
	}
	return CheckResponse(data, statusCode)
}

// applyWorkspaceOverride rewrites the "workspaces" field of a dashboard JSON
// payload when override is non-empty.
func applyWorkspaceOverride(body json.RawMessage, override string) (json.RawMessage, error) {
	if override == "" {
		return body, nil
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, fmt.Errorf("parsing dashboard payload: %w", err)
	}

	wsJSON, err := json.Marshal(override)
	if err != nil {
		return nil, err
	}
	m["workspaces"] = wsJSON

	out, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return out, nil
}
