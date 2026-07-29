package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerShareTools(r *registrar) {
	r.tool(tsWorkspacesShare, workspacesShareTool(), workspacesShareHandler(r.p))
	r.tool(tsDashboardsShare, dashboardsShareTool(), dashboardsShareHandler(r.p))
}

func splitCSV(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func workspacesShareTool() mcp.Tool {
	return mcp.NewTool("workspaces_share",
		mcp.WithDescription(`Share a workspace with other users/domains so it appears in THEIR web UI. This is the ONLY way to grant visibility (setting users[] via workspaces_update does NOT work).

The lists REPLACE the current sharing on every call. Pass empty users and domains to unshare completely.`),
		mcp.WithString("id", mcp.Description("Workspace ID"), mcp.Required()),
		mcp.WithString("users", mcp.Description("Comma-separated user emails to share with (empty = none)")),
		mcp.WithString("domains", mcp.Description("Comma-separated domains to share with (empty = none)")),
	)
}

func workspacesShareHandler(p *provider) server.ToolHandlerFunc {
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
		users := splitCSV(getString(args, "users"))
		domains := splitCSV(getString(args, "domains"))
		if _, err := c.ShareWorkspace(ctx, id, users, domains); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("share failed: %v", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Workspace %s sharing set: users=%v domains=%v", id, users, domains)), nil
	}
}

func dashboardsShareTool() mcp.Tool {
	return mcp.NewTool("dashboards_share",
		mcp.WithDescription("Share a single dashboard with other users/domains (lists REPLACE current sharing; empty = unshare). For whole workspaces use workspaces_share."),
		mcp.WithString("id", mcp.Description("Dashboard ID"), mcp.Required()),
		mcp.WithString("users", mcp.Description("Comma-separated user emails to share with (empty = none)")),
		mcp.WithString("domains", mcp.Description("Comma-separated domains to share with (empty = none)")),
	)
}

func dashboardsShareHandler(p *provider) server.ToolHandlerFunc {
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
		users := splitCSV(getString(args, "users"))
		domains := splitCSV(getString(args, "domains"))
		if _, err := c.ShareDashboard(ctx, id, users, domains); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("share failed: %v", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Dashboard %s sharing set: users=%v domains=%v", id, users, domains)), nil
	}
}

func getString(args map[string]any, key string) string {
	s, _ := args[key].(string)
	return s
}
