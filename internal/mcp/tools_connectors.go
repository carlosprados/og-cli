package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/carlosprados/og-cli/v2/pkg/opengate"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerConnectorTools(r *registrar) {
	r.tool(tsConnectors, connectorsListTool(), connectorsListHandler(r.p))
	r.tool(tsConnectors, connectorsGetTool(), connectorsGetHandler(r.p))
	r.tool(tsConnectorsWrite, connectorsCreateTool(), connectorsCreateHandler(r.p))
	r.tool(tsConnectorsWrite, connectorsUpdateTool(), connectorsUpdateHandler(r.p))
	r.tool(tsConnectorsWrite, connectorsDeleteTool(), connectorsDeleteHandler(r.p))
	r.tool(tsConnectorsOps, connectorsSetStatusTool(), connectorsSetStatusHandler(r.p))
	r.tool(tsConnectors, connectorsCatalogTool(), connectorsCatalogHandler(r.p))
	r.tool(tsConnectors, connectorsLogsTool(), connectorsLogsHandler(r.p))
}

// --- catalog ---

func connectorsCatalogTool() mcp.Tool {
	return mcp.NewTool("connectors_catalog",
		mcp.WithDescription("Show the platform connector functions catalog (predefined templates). No arguments."),
	)
}

func connectorsCatalogHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		data, err := c.ConnectorFunctionsCatalog(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("catalog failed: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// --- logs (bounded) ---

func connectorsLogsTool() mcp.Tool {
	return mcp.NewTool("connectors_logs",
		mcp.WithDescription("Collect a connector function's execution logs (functions-logger), up to 'count' lines or until 'timeout_seconds'. Traces come from logger.trace/debug/info/warn/error in the connector function JS. IMPORTANT: device-generated logs are only emitted while the TARGET device's administrativeState is TESTING — with an ACTIVE device the CF still runs and collects data but emits NO logs, so set the device to TESTING first. Use level DEBUG/TRACE to see logger.debug/trace (default INFO hides them)."),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
		mcp.WithString("id", mcp.Description("Connector function identifier"), mcp.Required()),
		mcp.WithString("channel", mcp.Description("Channel name (default: default_channel)")),
		mcp.WithString("level", mcp.Description("Log level: ERROR|WARN|INFO|DEBUG|TRACE (default INFO)")),
		mcp.WithNumber("count", mcp.Description("Max log lines to collect (default 20)")),
		mcp.WithNumber("timeout_seconds", mcp.Description("Max seconds to wait (default 15)")),
	)
}

func connectorsLogsHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		apiKey, errRes := p.apiKey(ctx)
		if errRes != nil {
			return errRes, nil
		}
		args := request.GetArguments()
		return functionLogsResult(ctx, c, apiKey, opengate.LoggerConnectorFunctions, connectorChannelArg(args), args)
	}
}

func connectorChannelArg(args map[string]any) string {
	if ch, _ := args["channel"].(string); ch != "" {
		return ch
	}
	return defaultRulesChannel
}

// --- list ---

func connectorsListTool() mcp.Tool {
	return mcp.NewTool("connectors_list",
		mcp.WithDescription(`List OpenGate connector functions in a channel. Connector functions are JavaScript hooks in the device-integration pipeline:
  REQUEST    — transform an outgoing operation request before it reaches the device (matched by operationName + northCriterias)
  RESPONSE   — process an operation response from the device (matched by southCriterias)
  COLLECTION — process collected data and emit datapoints (matched by southCriterias)

The code lives in the 'javascript' field. operationalStatus is DISABLED | PRODUCTION | TEST.`),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
		mcp.WithString("channel", mcp.Description("Channel name (default: default_channel)")),
	)
}

func connectorsListHandler(p *provider) server.ToolHandlerFunc {
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
		resp, err := c.ListConnectorFunctions(ctx, org, connectorChannelArg(args))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("list failed: %v", err)), nil
		}
		result, err := json.Marshal(resp.ConnectorFunctions)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshaling result: %v", err)), nil
		}
		return mcp.NewToolResultText(string(result)), nil
	}
}

// --- get ---

func connectorsGetTool() mcp.Tool {
	return mcp.NewTool("connectors_get",
		mcp.WithDescription("Get a connector function by identifier. The connector function code is in the 'javascript' field."),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
		mcp.WithString("id", mcp.Description("Connector function identifier (from connectors_list)"), mcp.Required()),
		mcp.WithString("channel", mcp.Description("Channel name (default: default_channel)")),
	)
}

func connectorsGetHandler(p *provider) server.ToolHandlerFunc {
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
		data, err := c.GetConnectorFunction(ctx, org, connectorChannelArg(args), id)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("get failed: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// --- create ---

func connectorsCreateTool() mcp.Tool {
	return mcp.NewTool("connectors_create",
		mcp.WithDescription(`Create a connector function.

REQUEST connector function body shape (transforms an outgoing operation request):
  {"name":"refreshInfo","type":"REQUEST","operationalStatus":"PRODUCTION","payloadType":"JSON",
   "operationName":"REFRESH_INFO",
   "northCriterias":[{"path":"provision.device.model._current.value.manufacturer","value":"Acme"}],
   "javascript":"// build the south payload here"}

COLLECTION/RESPONSE body shape (matched by southCriterias, operationName must be null):
  {"name":"collectData","type":"COLLECTION","operationalStatus":"TEST","payloadType":"JSON",
   "southCriterias":["mqtt://iot/collected"],
   "javascript":"collection.addDatapoint('device.name', 'value', Date.now()/1000); collection.send();"}`),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
		mcp.WithString("body", mcp.Description("Connector function JSON definition"), mcp.Required()),
		mcp.WithString("channel", mcp.Description("Channel name (default: default_channel)")),
	)
}

func connectorsCreateHandler(p *provider) server.ToolHandlerFunc {
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
		if _, err := c.CreateConnectorFunction(ctx, org, connectorChannelArg(args), json.RawMessage(body)); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("create failed: %v", err)), nil
		}
		return mcp.NewToolResultText("Connector function created successfully."), nil
	}
}

// --- update ---

func connectorsUpdateTool() mcp.Tool {
	return mcp.NewTool("connectors_update",
		mcp.WithDescription("Update an existing connector function. Fetch it first with connectors_get, modify, and send the FULL body back. The 'type' field cannot be changed."),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
		mcp.WithString("id", mcp.Description("Connector function identifier"), mcp.Required()),
		mcp.WithString("body", mcp.Description("Full connector function JSON definition"), mcp.Required()),
		mcp.WithString("channel", mcp.Description("Channel name (default: default_channel)")),
	)
}

func connectorsUpdateHandler(p *provider) server.ToolHandlerFunc {
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
		if err := c.UpdateConnectorFunction(ctx, org, connectorChannelArg(args), id, json.RawMessage(body)); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("update failed: %v", err)), nil
		}
		return mcp.NewToolResultText("Connector function updated successfully."), nil
	}
}

// --- delete ---

func connectorsDeleteTool() mcp.Tool {
	return mcp.NewTool("connectors_delete",
		mcp.WithDescription("Delete a connector function by identifier."),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
		mcp.WithString("id", mcp.Description("Connector function identifier"), mcp.Required()),
		mcp.WithString("channel", mcp.Description("Channel name (default: default_channel)")),
	)
}

func connectorsDeleteHandler(p *provider) server.ToolHandlerFunc {
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
		if err := c.DeleteConnectorFunction(ctx, org, connectorChannelArg(args), id); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("delete failed: %v", err)), nil
		}
		return mcp.NewToolResultText("Connector function deleted successfully."), nil
	}
}

// --- set status ---

func connectorsSetStatusTool() mcp.Tool {
	return mcp.NewTool("connectors_set_status",
		mcp.WithDescription("Change a connector function's operationalStatus (DISABLED, PRODUCTION, or TEST). Use this instead of connectors_update for simple status changes."),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
		mcp.WithString("id", mcp.Description("Connector function identifier"), mcp.Required()),
		mcp.WithString("status", mcp.Description("New operationalStatus: DISABLED | PRODUCTION | TEST"), mcp.Required()),
		mcp.WithString("channel", mcp.Description("Channel name (default: default_channel)")),
	)
}

func connectorsSetStatusHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		args := request.GetArguments()
		org, _ := args["organization"].(string)
		id, _ := args["id"].(string)
		status, _ := args["status"].(string)
		if org == "" || id == "" || status == "" {
			return mcp.NewToolResultError("organization, id, and status are required"), nil
		}
		switch status {
		case "DISABLED", "PRODUCTION", "TEST":
		default:
			return mcp.NewToolResultError("status must be DISABLED, PRODUCTION, or TEST"), nil
		}
		if err := c.SetConnectorFunctionStatus(ctx, org, connectorChannelArg(args), id, status); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("set status failed: %v", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Connector function operationalStatus=%s applied.", status)), nil
	}
}
