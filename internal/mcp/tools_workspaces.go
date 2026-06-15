package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerWorkspaceTools(s *server.MCPServer, p *provider) {
	s.AddTool(wsListTool(), wsListHandler(p))
	s.AddTool(wsGetTool(), wsGetHandler(p))
	s.AddTool(wsExportTool(), wsExportHandler(p))
	s.AddTool(wsImportTool(), wsImportHandler(p))
	s.AddTool(wsUpdateTool(), wsUpdateHandler(p))
	s.AddTool(wsDeleteTool(), wsDeleteHandler(p))
}

func wsListTool() mcp.Tool {
	return mcp.NewTool("workspaces_list",
		mcp.WithDescription("List workspaces. Use full=true to include embedded dashboards in each workspace."),
		mcp.WithBoolean("full", mcp.Description("Include embedded dashboards (?full=1)")),
	)
}

func wsListHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		args := request.GetArguments()
		full, _ := args["full"].(bool)
		wss, err := c.ListWorkspaces(full)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("list failed: %v", err)), nil
		}
		result, _ := json.Marshal(wss)
		return mcp.NewToolResultText(string(result)), nil
	}
}

func wsGetTool() mcp.Tool {
	return mcp.NewTool("workspaces_get",
		mcp.WithDescription("Get a workspace by ID."),
		mcp.WithString("id", mcp.Description("Workspace ID"), mcp.Required()),
		mcp.WithBoolean("full", mcp.Description("Include embedded dashboards (?full=1)")),
	)
}

func wsGetHandler(p *provider) server.ToolHandlerFunc {
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
		full, _ := args["full"].(bool)
		w, err := c.GetWorkspace(id, full)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("get failed: %v", err)), nil
		}
		result, _ := json.Marshal(w)
		return mcp.NewToolResultText(string(result)), nil
	}
}

func wsExportTool() mcp.Tool {
	return mcp.NewTool("workspaces_export",
		mcp.WithDescription("Export a workspace as JSON using /workspaces/export/{id}. Result is suitable for workspaces_import on another tenant."),
		mcp.WithString("id", mcp.Description("Workspace ID"), mcp.Required()),
	)
}

func wsExportHandler(p *provider) server.ToolHandlerFunc {
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
		data, err := c.ExportWorkspace(id)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("export failed: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func wsImportTool() mcp.Tool {
	return mcp.NewTool("workspaces_import",
		mcp.WithDescription("Create a workspace from a JSON payload (typically produced by workspaces_export)."),
		mcp.WithString("body", mcp.Description("Full workspace JSON definition"), mcp.Required()),
	)
}

func wsImportHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		args := request.GetArguments()
		body, _ := args["body"].(string)
		if body == "" {
			return mcp.NewToolResultError("body is required"), nil
		}
		resp, err := c.CreateWorkspace(json.RawMessage(body))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("import failed: %v", err)), nil
		}
		if len(resp) > 0 {
			return mcp.NewToolResultText(string(resp)), nil
		}
		return mcp.NewToolResultText("Workspace imported successfully."), nil
	}
}

func wsUpdateTool() mcp.Tool {
	return mcp.NewTool("workspaces_update",
		mcp.WithDescription("Update an existing workspace."),
		mcp.WithString("id", mcp.Description("Workspace ID"), mcp.Required()),
		mcp.WithString("body", mcp.Description("Full workspace JSON definition"), mcp.Required()),
	)
}

func wsUpdateHandler(p *provider) server.ToolHandlerFunc {
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
		if err := c.UpdateWorkspace(id, json.RawMessage(body)); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("update failed: %v", err)), nil
		}
		return mcp.NewToolResultText("Workspace updated successfully."), nil
	}
}

func wsDeleteTool() mcp.Tool {
	return mcp.NewTool("workspaces_delete",
		mcp.WithDescription("Delete a workspace by ID."),
		mcp.WithString("id", mcp.Description("Workspace ID"), mcp.Required()),
	)
}

func wsDeleteHandler(p *provider) server.ToolHandlerFunc {
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
		if err := c.DeleteWorkspace(id); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("delete failed: %v", err)), nil
		}
		return mcp.NewToolResultText("Workspace deleted successfully."), nil
	}
}
