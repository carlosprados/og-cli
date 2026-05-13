package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/carlosprados/og-cli/internal/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerDashboardTools(s *server.MCPServer, c *client.Client) {
	s.AddTool(dashListTool(), dashListHandler(c))
	s.AddTool(dashGetTool(), dashGetHandler(c))
	s.AddTool(dashExportTool(), dashExportHandler(c))
	s.AddTool(dashImportTool(), dashImportHandler(c))
	s.AddTool(dashUpdateTool(), dashUpdateHandler(c))
	s.AddTool(dashDeleteTool(), dashDeleteHandler(c))
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

func dashListHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		wsID, _ := args["workspace_id"].(string)

		var entries []dashListEntry
		if wsID != "" {
			w, err := c.GetWorkspace(wsID, true)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("list failed: %v", err)), nil
			}
			entries = collectDashEntries(w)
		} else {
			wss, err := c.ListWorkspaces(true)
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

func collectDashEntries(w *client.Workspace) []dashListEntry {
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

func dashGetHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		id, _ := args["id"].(string)
		if id == "" {
			return mcp.NewToolResultError("id is required"), nil
		}
		d, err := c.GetDashboard(id)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("get failed: %v", err)), nil
		}
		result, _ := json.Marshal(d)
		return mcp.NewToolResultText(string(result)), nil
	}
}

func dashExportTool() mcp.Tool {
	return mcp.NewTool("dashboards_export",
		mcp.WithDescription("Export a dashboard as JSON using /dashboards/export/{id}."),
		mcp.WithString("id", mcp.Description("Dashboard ID"), mcp.Required()),
	)
}

func dashExportHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		id, _ := args["id"].(string)
		if id == "" {
			return mcp.NewToolResultError("id is required"), nil
		}
		data, err := c.ExportDashboard(id)
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

func dashImportHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		body, _ := args["body"].(string)
		wsID, _ := args["workspace_id"].(string)
		if body == "" {
			return mcp.NewToolResultError("body is required"), nil
		}
		resp, err := c.CreateDashboard(json.RawMessage(body), wsID)
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

func dashUpdateHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		id, _ := args["id"].(string)
		body, _ := args["body"].(string)
		if id == "" || body == "" {
			return mcp.NewToolResultError("id and body are required"), nil
		}
		if err := c.UpdateDashboard(id, json.RawMessage(body)); err != nil {
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

func dashDeleteHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		id, _ := args["id"].(string)
		if id == "" {
			return mcp.NewToolResultError("id is required"), nil
		}
		if err := c.DeleteDashboard(id); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("delete failed: %v", err)), nil
		}
		return mcp.NewToolResultText("Dashboard deleted successfully."), nil
	}
}
