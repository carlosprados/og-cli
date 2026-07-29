package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/carlosprados/og-cli/v2/pkg/opengate"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const defaultRulesChannel = "default_channel"

func registerRuleTools(r *registrar) {
	r.tool(tsRules, rulesSearchTool(), rulesSearchHandler(r.p))
	r.tool(tsRules, rulesGetTool(), rulesGetHandler(r.p))
	r.tool(tsRulesWrite, rulesCreateTool(), rulesCreateHandler(r.p))
	r.tool(tsRulesWrite, rulesUpdateTool(), rulesUpdateHandler(r.p))
	r.tool(tsRulesWrite, rulesDeleteTool(), rulesDeleteHandler(r.p))
	r.tool(tsRulesOps, rulesSetActiveTool(), rulesSetActiveHandler(r.p))
	r.tool(tsRules, rulesCatalogTool(), rulesCatalogHandler(r.p))
	r.tool(tsRules, rulesLogsTool(), rulesLogsHandler(r.p))
}

// --- catalog ---

func rulesCatalogTool() mcp.Tool {
	return mcp.NewTool("rules_catalog",
		mcp.WithDescription("Show the platform rules catalog (predefined rule templates). No arguments."),
	)
}

func rulesCatalogHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		data, err := c.RulesCatalog(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("catalog failed: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// --- logs (bounded) ---

func rulesLogsTool() mcp.Tool {
	return mcp.NewTool("rules_logs",
		mcp.WithDescription("Collect a rule's execution logs (functions-logger), up to 'count' lines or until 'timeout_seconds'. Traces come from logger.trace/debug/info/warn/error in ADVANCED rule JS. IMPORTANT: device-generated logs are only emitted while the device that triggers the rule has administrativeState TESTING — with ACTIVE devices the rule still runs but emits NO logs. Use level DEBUG/TRACE to see logger.debug/trace (default INFO hides them)."),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
		mcp.WithString("id", mcp.Description("Rule identifier"), mcp.Required()),
		mcp.WithString("channel", mcp.Description("Channel name (default: default_channel)")),
		mcp.WithString("level", mcp.Description("Log level: ERROR|WARN|INFO|DEBUG|TRACE (default INFO)")),
		mcp.WithNumber("count", mcp.Description("Max log lines to collect (default 20)")),
		mcp.WithNumber("timeout_seconds", mcp.Description("Max seconds to wait (default 15)")),
	)
}

func rulesLogsHandler(p *provider) server.ToolHandlerFunc {
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
		return functionLogsResult(ctx, c, apiKey, opengate.LoggerRules, ruleChannelArg(args), args)
	}
}

func ruleChannelArg(args map[string]any) string {
	if ch, _ := args["channel"].(string); ch != "" {
		return ch
	}
	return defaultRulesChannel
}

// --- search ---

func rulesSearchTool() mcp.Tool {
	return mcp.NewTool("rules_search",
		mcp.WithDescription(`Search OpenGate automation rules. Rules trigger on datastream values, events, or operations and run actions (open/close alarms, email, HTTP, launch operations).

Two modes: EASY (declarative condition + actions) and ADVANCED (a JavaScript function decides).
Filterable fields use the rule. prefix: rule.name, rule.mode (EASY|ADVANCED), rule.active (true|false).

Examples:
  query: "rule.active eq true"
  query: "rule.mode eq ADVANCED"
  query: "rule.name like Battery"`),
		mcp.WithString("query",
			mcp.Description("Filter using: \"field op value\" joined with AND. Omit to list all rules."),
		),
		mcp.WithNumber("limit", mcp.Description("Max number of results")),
	)
}

func rulesSearchHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		filter, err := mcpBuildFilter(request.GetArguments())
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid query: %v", err)), nil
		}
		resp, err := c.SearchRules(ctx, filter)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
		}
		result, err := json.Marshal(resp.Rules)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshaling result: %v", err)), nil
		}
		return mcp.NewToolResultText(string(result)), nil
	}
}

// --- get ---

func rulesGetTool() mcp.Tool {
	return mcp.NewTool("rules_get",
		mcp.WithDescription("Get a rule by identifier. ADVANCED rules include their JavaScript code in the 'javascript' field."),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
		mcp.WithString("id", mcp.Description("Rule identifier (UUID, from rules_search)"), mcp.Required()),
		mcp.WithString("channel", mcp.Description("Channel name (default: default_channel)")),
	)
}

func rulesGetHandler(p *provider) server.ToolHandlerFunc {
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
		data, err := c.GetRule(ctx, org, ruleChannelArg(args), id)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("get failed: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// --- create ---

func rulesCreateTool() mcp.Tool {
	return mcp.NewTool("rules_create",
		mcp.WithDescription(`Create an automation rule.

EASY rule body shape:
  {"name":"battery-low","mode":"EASY","active":true,
   "type":{"name":"DATASTREAM","datastreams":["power.battery"]},
   "condition":{"filter":{"lt":{"power.battery._current.value":20}}},
   "actions":{"open":[{"name":"batteryLow","enabled":true,"severity":"CRITICAL","priority":"HIGH"}]}}

ADVANCED rule body shape (JavaScript decides; use openAlarm()/closeAlarm() helpers):
  {"name":"env-anomaly","mode":"ADVANCED","active":true,
   "type":{"name":"DATASTREAM","datastreams":["sensor.temperature"]},
   "javascript":"if (entity['sensor.temperature']._value._current.value > 28) { openAlarm(null, 'envAnomaly', ruleName, 'URGENT', 'HIGH', 'Temperature anomaly'); }"}`),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
		mcp.WithString("body", mcp.Description("Rule JSON definition"), mcp.Required()),
		mcp.WithString("channel", mcp.Description("Channel name (default: default_channel)")),
	)
}

func rulesCreateHandler(p *provider) server.ToolHandlerFunc {
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
		if _, err := c.CreateRule(ctx, org, ruleChannelArg(args), json.RawMessage(body)); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("create failed: %v", err)), nil
		}
		return mcp.NewToolResultText("Rule created successfully."), nil
	}
}

// --- update ---

func rulesUpdateTool() mcp.Tool {
	return mcp.NewTool("rules_update",
		mcp.WithDescription("Update an existing rule. Fetch it first with rules_get, modify, and send the FULL rule body back."),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
		mcp.WithString("id", mcp.Description("Rule identifier"), mcp.Required()),
		mcp.WithString("body", mcp.Description("Full rule JSON definition"), mcp.Required()),
		mcp.WithString("channel", mcp.Description("Channel name (default: default_channel)")),
	)
}

func rulesUpdateHandler(p *provider) server.ToolHandlerFunc {
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
		if err := c.UpdateRule(ctx, org, ruleChannelArg(args), id, json.RawMessage(body)); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("update failed: %v", err)), nil
		}
		return mcp.NewToolResultText("Rule updated successfully."), nil
	}
}

// --- delete ---

func rulesDeleteTool() mcp.Tool {
	return mcp.NewTool("rules_delete",
		mcp.WithDescription("Delete a rule by identifier."),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
		mcp.WithString("id", mcp.Description("Rule identifier"), mcp.Required()),
		mcp.WithString("channel", mcp.Description("Channel name (default: default_channel)")),
	)
}

func rulesDeleteHandler(p *provider) server.ToolHandlerFunc {
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
		if err := c.DeleteRule(ctx, org, ruleChannelArg(args), id); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("delete failed: %v", err)), nil
		}
		return mcp.NewToolResultText("Rule deleted successfully."), nil
	}
}

// --- enable/disable ---

func rulesSetActiveTool() mcp.Tool {
	return mcp.NewTool("rules_set_active",
		mcp.WithDescription("Enable or disable a rule (sets the 'active' field). Use this instead of rules_update for simple on/off changes."),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
		mcp.WithString("id", mcp.Description("Rule identifier"), mcp.Required()),
		mcp.WithBoolean("active", mcp.Description("true to enable, false to disable"), mcp.Required()),
		mcp.WithString("channel", mcp.Description("Channel name (default: default_channel)")),
	)
}

func rulesSetActiveHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		args := request.GetArguments()
		org, _ := args["organization"].(string)
		id, _ := args["id"].(string)
		active, ok := args["active"].(bool)
		if org == "" || id == "" || !ok {
			return mcp.NewToolResultError("organization, id, and active are required"), nil
		}
		if err := c.SetRuleActive(ctx, org, ruleChannelArg(args), id, active); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("set active failed: %v", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Rule active=%v applied.", active)), nil
	}
}
