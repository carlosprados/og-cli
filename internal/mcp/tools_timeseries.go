package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/carlosprados/og-cli/internal/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerTimeSeriesTools(s *server.MCPServer, c *client.Client) {
	s.AddTool(tsListTool(), tsListHandler(c))
	s.AddTool(tsGetTool(), tsGetHandler(c))
	s.AddTool(tsCreateTool(), tsCreateHandler(c))
	s.AddTool(tsUpdateTool(), tsUpdateHandler(c))
	s.AddTool(tsDeleteTool(), tsDeleteHandler(c))
	s.AddTool(tsDataTool(), tsDataHandler(c))
	s.AddTool(tsExportTool(), tsExportHandler(c))
}

// --- list ---

func tsListTool() mcp.Tool {
	return mcp.NewTool("timeseries_list",
		mcp.WithDescription("List all time series in an organization, including their columns and context."),
		mcp.WithString("organization",
			mcp.Description("Organization name"),
			mcp.Required(),
		),
	)
}

func tsListHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		org, _ := args["organization"].(string)
		if org == "" {
			return mcp.NewToolResultError("organization is required"), nil
		}
		resp, err := c.ListTimeSeries(org)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("list failed: %v", err)), nil
		}
		result, _ := json.Marshal(resp.Timeseries)
		return mcp.NewToolResultText(string(result)), nil
	}
}

// --- get ---

func tsGetTool() mcp.Tool {
	return mcp.NewTool("timeseries_get",
		mcp.WithDescription("Get the full definition of a time series including columns, context, and sorts."),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
		mcp.WithString("id", mcp.Description("Time series identifier"), mcp.Required()),
	)
}

func tsGetHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		org, _ := args["organization"].(string)
		id, _ := args["id"].(string)
		if org == "" || id == "" {
			return mcp.NewToolResultError("organization and id are required"), nil
		}
		ts, err := c.GetTimeSeries(org, id)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("get failed: %v", err)), nil
		}
		result, _ := json.Marshal(ts)
		return mcp.NewToolResultText(string(result)), nil
	}
}

// --- create ---

func tsCreateTool() mcp.Tool {
	return mcp.NewTool("timeseries_create",
		mcp.WithDescription("Create a new time series in an organization."),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
		mcp.WithString("body", mcp.Description("Full time series JSON definition"), mcp.Required()),
	)
}

func tsCreateHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		org, _ := args["organization"].(string)
		body, _ := args["body"].(string)
		if org == "" || body == "" {
			return mcp.NewToolResultError("organization and body are required"), nil
		}
		if err := c.CreateTimeSeries(org, json.RawMessage(body)); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("create failed: %v", err)), nil
		}
		return mcp.NewToolResultText("Time series created successfully."), nil
	}
}

// --- update ---

func tsUpdateTool() mcp.Tool {
	return mcp.NewTool("timeseries_update",
		mcp.WithDescription("Update an existing time series."),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
		mcp.WithString("id", mcp.Description("Time series identifier"), mcp.Required()),
		mcp.WithString("body", mcp.Description("Full time series JSON definition"), mcp.Required()),
	)
}

func tsUpdateHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		org, _ := args["organization"].(string)
		id, _ := args["id"].(string)
		body, _ := args["body"].(string)
		if org == "" || id == "" || body == "" {
			return mcp.NewToolResultError("organization, id, and body are required"), nil
		}
		if err := c.UpdateTimeSeries(org, id, json.RawMessage(body)); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("update failed: %v", err)), nil
		}
		return mcp.NewToolResultText("Time series updated successfully."), nil
	}
}

// --- delete ---

func tsDeleteTool() mcp.Tool {
	return mcp.NewTool("timeseries_delete",
		mcp.WithDescription("Delete a time series."),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
		mcp.WithString("id", mcp.Description("Time series identifier"), mcp.Required()),
	)
}

func tsDeleteHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		org, _ := args["organization"].(string)
		id, _ := args["id"].(string)
		if org == "" || id == "" {
			return mcp.NewToolResultError("organization and id are required"), nil
		}
		if err := c.DeleteTimeSeries(org, id); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("delete failed: %v", err)), nil
		}
		return mcp.NewToolResultText("Time series deleted successfully."), nil
	}
}

// --- data ---

func tsDataTool() mcp.Tool {
	return mcp.NewTool("timeseries_data",
		mcp.WithDescription(`Query collected data from a time series. Returns tabular data with columns and rows.
Use 'query' to filter by column names defined in the time series.`),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
		mcp.WithString("id", mcp.Description("Time series identifier"), mcp.Required()),
		mcp.WithString("query",
			mcp.Description("Filter: \"column_name op value\". Example: \"Prov Identifier eq MyDevice1\""),
		),
		mcp.WithString("sort", mcp.Description("Sort identifier defined in the time series")),
		mcp.WithNumber("limit", mcp.Description("Max number of rows")),
		mcp.WithString("filter", mcp.Description("Advanced: raw OpenGate JSON filter. Overrides 'query'.")),
	)
}

func tsDataHandler(c *client.Client) server.ToolHandlerFunc {
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

		// Inject sort if provided
		if sort, _ := args["sort"].(string); sort != "" {
			filter, err = injectSortJSON(filter, sort)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid sort: %v", err)), nil
			}
		}

		resp, err := c.QueryTimeSeriesData(org, id, filter)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("query failed: %v", err)), nil
		}

		result, _ := json.Marshal(resp)
		return mcp.NewToolResultText(string(result)), nil
	}
}

// --- export ---

func tsExportTool() mcp.Tool {
	return mcp.NewTool("timeseries_export",
		mcp.WithDescription("Trigger a Parquet export of a time series."),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
		mcp.WithString("id", mcp.Description("Time series identifier"), mcp.Required()),
	)
}

func tsExportHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		org, _ := args["organization"].(string)
		id, _ := args["id"].(string)
		if org == "" || id == "" {
			return mcp.NewToolResultError("organization and id are required"), nil
		}
		if err := c.ExportTimeSeries(org, id, nil); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("export failed: %v", err)), nil
		}
		return mcp.NewToolResultText("Export triggered successfully."), nil
	}
}

func injectSortJSON(filter json.RawMessage, sort string) (json.RawMessage, error) {
	if filter == nil {
		return json.Marshal(map[string]any{"sort": sort})
	}
	var m map[string]any
	if err := json.Unmarshal(filter, &m); err != nil {
		return nil, err
	}
	m["sort"] = sort
	return json.Marshal(m)
}
