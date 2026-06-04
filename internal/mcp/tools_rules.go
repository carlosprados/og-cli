package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/carlosprados/og-cli/internal/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const defaultRulesChannel = "default_channel"

func registerRuleTools(s *server.MCPServer, c *client.Client) {
	s.AddTool(rulesSearchTool(), rulesSearchHandler(c))
	s.AddTool(rulesGetTool(), rulesGetHandler(c))
	s.AddTool(rulesCreateTool(), rulesCreateHandler(c))
	s.AddTool(rulesUpdateTool(), rulesUpdateHandler(c))
	s.AddTool(rulesDeleteTool(), rulesDeleteHandler(c))
	s.AddTool(rulesSetActiveTool(), rulesSetActiveHandler(c))
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

func rulesSearchHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		filter, err := mcpBuildFilter(request.GetArguments())
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid query: %v", err)), nil
		}
		resp, err := c.SearchRules(filter)
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

func rulesGetHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		org, _ := args["organization"].(string)
		id, _ := args["id"].(string)
		if org == "" || id == "" {
			return mcp.NewToolResultError("organization and id are required"), nil
		}
		data, err := c.GetRule(org, ruleChannelArg(args), id)
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

func rulesCreateHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		org, _ := args["organization"].(string)
		body, _ := args["body"].(string)
		if org == "" || body == "" {
			return mcp.NewToolResultError("organization and body are required"), nil
		}
		if _, err := c.CreateRule(org, ruleChannelArg(args), json.RawMessage(body)); err != nil {
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

func rulesUpdateHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		org, _ := args["organization"].(string)
		id, _ := args["id"].(string)
		body, _ := args["body"].(string)
		if org == "" || id == "" || body == "" {
			return mcp.NewToolResultError("organization, id, and body are required"), nil
		}
		if err := c.UpdateRule(org, ruleChannelArg(args), id, json.RawMessage(body)); err != nil {
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

func rulesDeleteHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		org, _ := args["organization"].(string)
		id, _ := args["id"].(string)
		if org == "" || id == "" {
			return mcp.NewToolResultError("organization and id are required"), nil
		}
		if err := c.DeleteRule(org, ruleChannelArg(args), id); err != nil {
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

func rulesSetActiveHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		org, _ := args["organization"].(string)
		id, _ := args["id"].(string)
		active, ok := args["active"].(bool)
		if org == "" || id == "" || !ok {
			return mcp.NewToolResultError("organization, id, and active are required"), nil
		}
		if err := c.SetRuleActive(org, ruleChannelArg(args), id, active); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("set active failed: %v", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Rule active=%v applied.", active)), nil
	}
}
