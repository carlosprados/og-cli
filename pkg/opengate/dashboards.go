package opengate

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	dashboardsPath = "/api/dashboards"
	// The delete endpoint answers 400 without the trailing slash.
	dashboardsDeletePath = "/api/dashboards/"
	dashboardPath        = "/api/dashboards/%s"
	dashboardExportPath  = "/api/dashboards/export/%s"
)

// Dashboard represents a full OpenGate Web API dashboard. Every dashboard
// belongs to exactly one workspace, referenced by the Workspaces field.
type Dashboard struct {
	ID              string   `json:"_id,omitempty"`
	AltID           string   `json:"id,omitempty"`
	Title           string   `json:"title"`
	Description     *string  `json:"description,omitempty"`
	Icon            string   `json:"icon,omitempty"`
	IconType        string   `json:"iconType,omitempty"`
	Owner           string   `json:"owner,omitempty"`
	Workspaces      string   `json:"workspaces,omitempty"`
	Users           []string `json:"users,omitempty"`
	Workgroups      []string `json:"workgroups,omitempty"`
	AllowedProfiles []string `json:"allowedProfiles,omitempty"`
	Domains         []string `json:"domains,omitempty"`
	LastAccess      string   `json:"lastAccess,omitempty"`
	Editable        *bool    `json:"editable,omitempty"`
	BackgroundImage *string  `json:"backgroundImage,omitempty"`
	// BackgroundColor and BackgroundImageSize sit next to BackgroundImage in
	// the platform's document and were the two the struct did not name, so a
	// dashboard read through it came back without them. Pointers, so an empty
	// string stays an empty string rather than collapsing into "absent".
	BackgroundColor     *string               `json:"backgroundColor,omitempty"`
	BackgroundImageSize *string               `json:"backgroundImageSize,omitempty"`
	BannerImage         *string               `json:"bannerImage,omitempty"`
	Version             int                   `json:"__v,omitempty"`
	ExtraConfig         *DashboardExtraConfig `json:"extraConfig,omitempty"`
	Grid                []GridItem            `json:"grid,omitempty"`
	TemplateConfig      json.RawMessage       `json:"templateConfig,omitempty"`
}

// DashboardSimplified is the dashboard payload returned inside a workspace's
// embedded dashboards array. Some endpoints omit the grid (workspaces?full=1),
// others include it (workspaces/export/{id}). Grid is therefore optional.
type DashboardSimplified struct {
	ID              string   `json:"_id,omitempty"`
	AltID           string   `json:"id,omitempty"`
	Title           string   `json:"title"`
	Description     *string  `json:"description,omitempty"`
	Icon            string   `json:"icon,omitempty"`
	IconType        string   `json:"iconType,omitempty"`
	Owner           string   `json:"owner,omitempty"`
	Workspaces      string   `json:"workspaces,omitempty"`
	Users           []string `json:"users,omitempty"`
	Workgroups      []string `json:"workgroups,omitempty"`
	AllowedProfiles []string `json:"allowedProfiles,omitempty"`
	Domains         []string `json:"domains,omitempty"`
	LastAccess      string   `json:"lastAccess,omitempty"`
	Editable        *bool    `json:"editable,omitempty"`
	BackgroundImage *string  `json:"backgroundImage,omitempty"`
	// Same two fields as on Dashboard: the platform sends them on the embedded
	// dashboard too, and this struct is what a workspace read decodes into.
	BackgroundColor     *string               `json:"backgroundColor,omitempty"`
	BackgroundImageSize *string               `json:"backgroundImageSize,omitempty"`
	BannerImage         *string               `json:"bannerImage,omitempty"`
	Version             int                   `json:"__v,omitempty"`
	ExtraConfig         *DashboardExtraConfig `json:"extraConfig,omitempty"`
	Grid                []GridItem            `json:"grid,omitempty"`
	TemplateConfig      json.RawMessage       `json:"templateConfig,omitempty"`
}

// DashboardExtraConfig holds display options for a dashboard.
type DashboardExtraConfig struct {
	CellsWidth               string `json:"cellsWidth,omitempty"`
	CellHeight               int    `json:"cellHeight,omitempty"`
	DashboardRefreshInterval string `json:"dashboardRefreshInterval,omitempty"`
	// ShowBanner and Favourite carry no omitempty. The platform states both on
	// every extraConfig it sends (3 of 3 in sensehat), so omitting the false
	// half turned "banner off" into "unset" on every read.
	ShowBanner bool `json:"showBanner"`
	Favourite  bool `json:"favourite"`
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
//
// Ftype is an OpenGate platform field paired with Type: Type is the visual
// component (e.g. FullDevicesList, OperationsList) and Ftype is its data domain
// (entities, jobs, operations, alarms). og does not interpret it — it is
// preserved verbatim so the unwrap → wrap → import round-trip keeps the
// widget's data-source binding intact.
type WidgetDefinition struct {
	Type   string          `json:"type,omitempty"`
	Ftype  string          `json:"Ftype,omitempty"`
	Wid    string          `json:"wid,omitempty"`
	Config json.RawMessage `json:"config,omitempty"`
}

// GetDashboard retrieves a single dashboard by ID.
func (c *Client) GetDashboard(ctx context.Context, id string) (*Dashboard, error) {
	path := fmt.Sprintf(dashboardPath, id)

	data, statusCode, err := c.WebGet(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("get dashboard: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	if err := notFoundIfEmpty(data, statusCode, "dashboard", id); err != nil {
		return nil, err
	}

	var d Dashboard
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, fmt.Errorf("parsing dashboard: %w", err)
	}
	return &d, nil
}

// ExportDashboard fetches the export payload for a dashboard as raw JSON.
func (c *Client) ExportDashboard(ctx context.Context, id string) ([]byte, error) {
	path := fmt.Sprintf(dashboardExportPath, id)

	data, statusCode, err := c.WebGet(ctx, path)
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
func (c *Client) CreateDashboard(ctx context.Context, body json.RawMessage, workspaceOverride string) ([]byte, error) {
	payload, err := applyWorkspaceOverride(body, workspaceOverride)
	if err != nil {
		return nil, fmt.Errorf("create dashboard: %w", err)
	}

	data, statusCode, err := c.WebPost(ctx, dashboardsPath, strings.NewReader(string(payload)))
	if err != nil {
		return nil, fmt.Errorf("create dashboard: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	return data, nil
}

// UpdateDashboard updates an existing dashboard.
func (c *Client) UpdateDashboard(ctx context.Context, id string, body json.RawMessage) error {
	path := fmt.Sprintf(dashboardPath, id)

	data, statusCode, err := c.WebPut(ctx, path, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("update dashboard: %w", err)
	}
	return CheckResponse(data, statusCode)
}

// DeleteDashboard deletes a dashboard by ID.
//
// The endpoint takes a list under the key "dasboardsDelete" — the platform's
// own spelling, missing the "h", which is why every reasonable guess at this
// body ({"_id":…}, {"ids":[…]}, the document itself) came back 400 with an
// empty message. Confirmed by reading the web client's bundle, which calls
// delete("/api/dashboards/", {data:{dasboardsDelete:[…]}}) with dashboard ids.
//
// Two details the 400s hid: the trailing slash on the path is required, and the
// ids are the dashboard's "id", not its "_id" (they hold the same value on
// every dashboard seen so far, but the field the UI reads is "id").
func (c *Client) DeleteDashboard(ctx context.Context, id string) error {
	payload, err := json.Marshal(map[string][]string{"dasboardsDelete": {id}})
	if err != nil {
		return fmt.Errorf("delete dashboard: %w", err)
	}

	data, statusCode, err := c.webDoRequest(ctx, "DELETE", dashboardsDeletePath, strings.NewReader(string(payload)))
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
