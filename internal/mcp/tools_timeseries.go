package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerTimeSeriesTools(r *registrar) {
	r.tool(tsTimeseries, tsListTool(), tsListHandler(r.p))
	r.tool(tsTimeseries, tsGetTool(), tsGetHandler(r.p))
	r.tool(tsTimeseriesWrite, tsCreateTool(), tsCreateHandler(r.p))
	r.tool(tsTimeseriesWrite, tsUpdateTool(), tsUpdateHandler(r.p))
	r.tool(tsTimeseriesWrite, tsDeleteTool(), tsDeleteHandler(r.p))
	r.tool(tsTimeseries, tsDataTool(), tsDataHandler(r.p))
	r.tool(tsTimeseries, tsExportTool(), tsExportHandler(r.p))
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

func tsListHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		args := request.GetArguments()
		org, _ := args["organization"].(string)
		if org == "" {
			return mcp.NewToolResultError("organization is required"), nil
		}
		resp, err := c.ListTimeSeries(ctx, org)
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

func tsGetHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		args := request.GetArguments()
		org, _ := args["organization"].(string)
		id, _ := args["id"].(string)
		if org == "" || id == "" {
			return mcp.NewToolResultError("organization and id are required"), nil
		}
		ts, err := c.GetTimeSeries(ctx, org, id)
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

func tsCreateHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		args := request.GetArguments()
		org, _ := args["organization"].(string)
		body, _ := args["body"].(string)
		if org == "" || body == "" {
			return mcp.NewToolResultError("organization and body are required"), nil
		}
		if err := c.CreateTimeSeries(ctx, org, json.RawMessage(body)); err != nil {
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

func tsUpdateHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		args := request.GetArguments()
		org, _ := args["organization"].(string)
		id, _ := args["id"].(string)
		body, _ := args["body"].(string)
		if org == "" || id == "" || body == "" {
			return mcp.NewToolResultError("organization, id, and body are required"), nil
		}
		if err := c.UpdateTimeSeries(ctx, org, id, json.RawMessage(body)); err != nil {
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

func tsDeleteHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		args := request.GetArguments()
		org, _ := args["organization"].(string)
		id, _ := args["id"].(string)
		if org == "" || id == "" {
			return mcp.NewToolResultError("organization and id are required"), nil
		}
		if err := c.DeleteTimeSeries(ctx, org, id); err != nil {
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

func tsDataHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
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

		resp, err := c.QueryTimeSeriesData(ctx, org, id, filter)
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

func tsExportHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		args := request.GetArguments()
		org, _ := args["organization"].(string)
		id, _ := args["id"].(string)
		if org == "" || id == "" {
			return mcp.NewToolResultError("organization and id are required"), nil
		}
		if err := c.ExportTimeSeries(ctx, org, id, nil); err != nil {
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
