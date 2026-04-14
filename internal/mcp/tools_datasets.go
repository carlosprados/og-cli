package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/carlosprados/og-cli/internal/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerDatasetTools(s *server.MCPServer, c *client.Client) {
	s.AddTool(dsListTool(), dsListHandler(c))
	s.AddTool(dsGetTool(), dsGetHandler(c))
	s.AddTool(dsCreateTool(), dsCreateHandler(c))
	s.AddTool(dsUpdateTool(), dsUpdateHandler(c))
	s.AddTool(dsDeleteTool(), dsDeleteHandler(c))
	s.AddTool(dsDataTool(), dsDataHandler(c))
}

func dsListTool() mcp.Tool {
	return mcp.NewTool("datasets_list",
		mcp.WithDescription("List all datasets in an organization."),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
	)
}

func dsListHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		org, _ := args["organization"].(string)
		if org == "" {
			return mcp.NewToolResultError("organization is required"), nil
		}
		resp, err := c.ListDatasets(org)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("list failed: %v", err)), nil
		}
		result, _ := json.Marshal(resp.Datasets)
		return mcp.NewToolResultText(string(result)), nil
	}
}

func dsGetTool() mcp.Tool {
	return mcp.NewTool("datasets_get",
		mcp.WithDescription("Get the full definition of a dataset including columns."),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
		mcp.WithString("id", mcp.Description("Dataset identifier"), mcp.Required()),
	)
}

func dsGetHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		org, _ := args["organization"].(string)
		id, _ := args["id"].(string)
		if org == "" || id == "" {
			return mcp.NewToolResultError("organization and id are required"), nil
		}
		ds, err := c.GetDataset(org, id)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("get failed: %v", err)), nil
		}
		result, _ := json.Marshal(ds)
		return mcp.NewToolResultText(string(result)), nil
	}
}

func dsCreateTool() mcp.Tool {
	return mcp.NewTool("datasets_create",
		mcp.WithDescription("Create a new dataset in an organization."),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
		mcp.WithString("body", mcp.Description("Full dataset JSON definition"), mcp.Required()),
	)
}

func dsCreateHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		org, _ := args["organization"].(string)
		body, _ := args["body"].(string)
		if org == "" || body == "" {
			return mcp.NewToolResultError("organization and body are required"), nil
		}
		if err := c.CreateDataset(org, json.RawMessage(body)); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("create failed: %v", err)), nil
		}
		return mcp.NewToolResultText("Dataset created successfully."), nil
	}
}

func dsUpdateTool() mcp.Tool {
	return mcp.NewTool("datasets_update",
		mcp.WithDescription("Update an existing dataset."),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
		mcp.WithString("id", mcp.Description("Dataset identifier"), mcp.Required()),
		mcp.WithString("body", mcp.Description("Full dataset JSON definition"), mcp.Required()),
	)
}

func dsUpdateHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		org, _ := args["organization"].(string)
		id, _ := args["id"].(string)
		body, _ := args["body"].(string)
		if org == "" || id == "" || body == "" {
			return mcp.NewToolResultError("organization, id, and body are required"), nil
		}
		if err := c.UpdateDataset(org, id, json.RawMessage(body)); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("update failed: %v", err)), nil
		}
		return mcp.NewToolResultText("Dataset updated successfully."), nil
	}
}

func dsDeleteTool() mcp.Tool {
	return mcp.NewTool("datasets_delete",
		mcp.WithDescription("Delete a dataset."),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
		mcp.WithString("id", mcp.Description("Dataset identifier"), mcp.Required()),
	)
}

func dsDeleteHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		org, _ := args["organization"].(string)
		id, _ := args["id"].(string)
		if org == "" || id == "" {
			return mcp.NewToolResultError("organization and id are required"), nil
		}
		if err := c.DeleteDataset(org, id); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("delete failed: %v", err)), nil
		}
		return mcp.NewToolResultText("Dataset deleted successfully."), nil
	}
}

func dsDataTool() mcp.Tool {
	return mcp.NewTool("datasets_data",
		mcp.WithDescription(`Query data from a dataset. Returns tabular data with columns and rows.
Use 'query' to filter by column names defined in the dataset.`),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
		mcp.WithString("id", mcp.Description("Dataset identifier"), mcp.Required()),
		mcp.WithString("query",
			mcp.Description("Filter: \"column_name op value\". Example: \"Prov Identifier eq MyDevice1\""),
		),
		mcp.WithNumber("limit", mcp.Description("Max number of rows")),
		mcp.WithString("filter", mcp.Description("Advanced: raw OpenGate JSON filter. Overrides 'query'.")),
	)
}

func dsDataHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		org, _ := args["organization"].(string)
		id, _ := args["id"].(string)
		if org == "" || id == "" {
			return mcp.NewToolResultError("organization and id are required"), nil
		}

		filter, err := mcpBuildFilter(args)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid query: %v", err)), nil
		}

		resp, err := c.QueryDatasetData(org, id, filter)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("query failed: %v", err)), nil
		}

		result, _ := json.Marshal(resp)
		return mcp.NewToolResultText(string(result)), nil
	}
}
