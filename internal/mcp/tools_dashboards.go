package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/carlosprados/og-cli/v2/internal/unwrap"
	"github.com/carlosprados/og-cli/v2/pkg/opengate"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerDashboardTools(r *registrar) {
	r.tool(tsDashboards, dashListTool(), dashListHandler(r.p))
	r.tool(tsDashboards, dashGetTool(), dashGetHandler(r.p))
	r.tool(tsDashboards, dashCodeTool(), dashCodeHandler(r.p))
	r.tool(tsDashboards, dashExportTool(), dashExportHandler(r.p))
	r.tool(tsDashboardsWrite, dashImportTool(), dashImportHandler(r.p))
	r.tool(tsDashboardsWrite, dashUpdateTool(), dashUpdateHandler(r.p))
	r.tool(tsDashboardsWrite, dashDeleteTool(), dashDeleteHandler(r.p))
}

type dashListEntry struct {
	WorkspaceID   string `json:"workspaceId"`
	WorkspaceName string `json:"workspaceName"`
	DashboardID   string `json:"dashboardId"`
	Title         string `json:"title"`
	Owner         string `json:"owner"`
}

func dashListTool() mcp.Tool {
	return mcp.NewTool("dashboards_list",
		mcp.WithDescription(`List dashboards. Every dashboard belongs to exactly one workspace.
Without 'workspace_id': iterates all workspaces (with full=1) and returns every dashboard.
With 'workspace_id': returns only the dashboards of that workspace.`),
		mcp.WithString("workspace_id", mcp.Description("Optional workspace ID to filter dashboards")),
	)
}

func dashListHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		args := request.GetArguments()
		wsID, _ := args["workspace_id"].(string)

		var entries []dashListEntry
		if wsID != "" {
			w, err := c.GetWorkspace(ctx, wsID, true)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("list failed: %v", err)), nil
			}
			entries = collectDashEntries(w)
		} else {
			wss, err := c.ListWorkspaces(ctx, true)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("list failed: %v", err)), nil
			}
			for i := range wss {
				entries = append(entries, collectDashEntries(&wss[i])...)
			}
		}

		result, _ := json.Marshal(entries)
		return mcp.NewToolResultText(string(result)), nil
	}
}

func collectDashEntries(w *opengate.Workspace) []dashListEntry {
	out := make([]dashListEntry, 0, len(w.Dashboards))
	for _, wd := range w.Dashboards {
		if wd.Dashboard == nil {
			out = append(out, dashListEntry{
				WorkspaceID:   w.ID,
				WorkspaceName: w.Name,
				DashboardID:   wd.ID,
			})
			continue
		}
		out = append(out, dashListEntry{
			WorkspaceID:   w.ID,
			WorkspaceName: w.Name,
			DashboardID:   wd.Dashboard.ID,
			Title:         wd.Dashboard.Title,
			Owner:         wd.Dashboard.Owner,
		})
	}
	return out
}

func dashGetTool() mcp.Tool {
	return mcp.NewTool("dashboards_get",
		mcp.WithDescription("Get a dashboard by ID, including grid layout and widget definitions."),
		mcp.WithString("id", mcp.Description("Dashboard ID"), mcp.Required()),
	)
}

func dashGetHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		args := request.GetArguments()
		id, _ := args["id"].(string)
		if id == "" {
			return mcp.NewToolResultError("id is required"), nil
		}
		d, err := c.GetDashboard(ctx, id)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("get failed: %v", err)), nil
		}
		result, _ := json.Marshal(d)
		return mcp.NewToolResultText(string(result)), nil
	}
}

// --- code ---
//
// dashboards_get already returns the widget code, but buried: it is one string
// among the nested config of every widget in the grid, so reading one formatter
// costs the whole dashboard in context. This is the same extraction
// `og dashboard show` performs, addressed by the path `og workspace pull`
// writes on disk — so an agent and a person editing the tree name a file the
// same way.
//
// The flat families get no equivalent on purpose: a rule's code is a single
// field of a payload rules_get returns whole, and a second way to read it would
// buy nothing.

func dashCodeTool() mcp.Tool {
	return mcp.NewTool("dashboards_code",
		mcp.WithDescription(`Read a dashboard's widget JavaScript, one file at a time.
Without 'path': lists the code files the dashboard carries, with their size.
With 'path': returns that file's content and nothing else.
Paths are '<widget-dir>/<file>.js', the same ones 'og workspace pull' writes; the
NN prefix is the grid position and is ignored when matching, so a path from a
tree pulled before a reorder still resolves.`),
		mcp.WithString("id", mcp.Description("Dashboard ID"), mcp.Required()),
		mcp.WithString("path", mcp.Description("Code file to return, e.g. 01__customtable__sales/_widgetConfigCode.js")),
	)
}

type dashCodeEntry struct {
	File  string `json:"file"`
	Bytes int    `json:"bytes"`
	Lines int    `json:"lines"`
}

func dashCodeHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		args := request.GetArguments()
		id, _ := args["id"].(string)
		if id == "" {
			return mcp.NewToolResultError("id is required"), nil
		}

		d, err := c.GetDashboard(ctx, id)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("get failed: %v", err)), nil
		}
		files, err := unwrap.DashboardCodeFiles(d)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("reading the widget code failed: %v", err)), nil
		}

		names := make([]string, 0, len(files))
		for name := range files {
			names = append(names, name)
		}
		sort.Strings(names)

		if path, _ := args["path"].(string); path != "" {
			content, ok := unwrap.ResolveCodePath(files, path)
			if !ok {
				// Naming what is there turns a wrong guess into one more call
				// rather than into a dead end.
				return mcp.NewToolResultError(fmt.Sprintf("dashboard %s carries no file %q; it has: %s",
					id, path, strings.Join(names, ", "))), nil
			}
			return mcp.NewToolResultText(content), nil
		}

		entries := make([]dashCodeEntry, 0, len(names))
		for _, name := range names {
			entries = append(entries, dashCodeEntry{
				File:  name,
				Bytes: len(files[name]),
				Lines: strings.Count(files[name], "\n") + 1,
			})
		}
		result, _ := json.Marshal(entries)
		return mcp.NewToolResultText(string(result)), nil
	}
}

func dashExportTool() mcp.Tool {
	return mcp.NewTool("dashboards_export",
		mcp.WithDescription("Export a dashboard as JSON using /dashboards/export/{id}."),
		mcp.WithString("id", mcp.Description("Dashboard ID"), mcp.Required()),
	)
}

func dashExportHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		args := request.GetArguments()
		id, _ := args["id"].(string)
		if id == "" {
			return mcp.NewToolResultError("id is required"), nil
		}
		data, err := c.ExportDashboard(ctx, id)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("export failed: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func dashImportTool() mcp.Tool {
	return mcp.NewTool("dashboards_import",
		mcp.WithDescription(`Create a dashboard from a JSON payload.
If 'workspace_id' is provided, it overrides the "workspaces" field in the payload
(useful when migrating a dashboard to a different tenant or workspace).`),
		mcp.WithString("body", mcp.Description("Full dashboard JSON definition"), mcp.Required()),
		mcp.WithString("workspace_id", mcp.Description("Optional override for the target workspace ID")),
	)
}

func dashImportHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		args := request.GetArguments()
		body, _ := args["body"].(string)
		wsID, _ := args["workspace_id"].(string)
		if body == "" {
			return mcp.NewToolResultError("body is required"), nil
		}
		resp, err := c.CreateDashboard(ctx, json.RawMessage(body), wsID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("import failed: %v", err)), nil
		}
		if len(resp) > 0 {
			return mcp.NewToolResultText(string(resp)), nil
		}
		return mcp.NewToolResultText("Dashboard imported successfully."), nil
	}
}

func dashUpdateTool() mcp.Tool {
	return mcp.NewTool("dashboards_update",
		mcp.WithDescription("Update an existing dashboard."),
		mcp.WithString("id", mcp.Description("Dashboard ID"), mcp.Required()),
		mcp.WithString("body", mcp.Description("Full dashboard JSON definition"), mcp.Required()),
	)
}

func dashUpdateHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		args := request.GetArguments()
		id, _ := args["id"].(string)
		body, _ := args["body"].(string)
		if id == "" || body == "" {
			return mcp.NewToolResultError("id and body are required"), nil
		}
		if err := c.UpdateDashboard(ctx, id, json.RawMessage(body)); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("update failed: %v", err)), nil
		}
		return mcp.NewToolResultText("Dashboard updated successfully."), nil
	}
}

func dashDeleteTool() mcp.Tool {
	return mcp.NewTool("dashboards_delete",
		mcp.WithDescription("Delete a dashboard by ID."),
		mcp.WithString("id", mcp.Description("Dashboard ID"), mcp.Required()),
	)
}

func dashDeleteHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		args := request.GetArguments()
		id, _ := args["id"].(string)
		if id == "" {
			return mcp.NewToolResultError("id is required"), nil
		}
		if err := c.DeleteDashboard(ctx, id); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("delete failed: %v", err)), nil
		}
		return mcp.NewToolResultText("Dashboard deleted successfully."), nil
	}
}
