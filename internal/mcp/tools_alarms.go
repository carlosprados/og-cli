package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/carlosprados/og-cli/internal/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerAlarmTools(s *server.MCPServer, c *client.Client) {
	s.AddTool(alarmsSearchTool(), alarmsSearchHandler(c))
	s.AddTool(alarmsSummaryTool(), alarmsSummaryHandler(c))
	s.AddTool(alarmsAttendTool(), alarmsAttendHandler(c))
	s.AddTool(alarmsCloseTool(), alarmsCloseHandler(c))
}

// --- search ---

func alarmsSearchTool() mcp.Tool {
	return mcp.NewTool("alarms_search",
		mcp.WithDescription(`Search OpenGate alarms. ALWAYS use the 'query' parameter for filtering.

Common alarm fields for filtering:
- alarm.severity — INFORMATIVE, URGENT, CRITICAL
- alarm.status — OPEN, ATTEND, CLOSED
- alarm.name — alarm name
- alarm.rule — rule that triggered the alarm
- alarm.entityIdentifier — device/entity that raised the alarm
- alarm.organization — organization name
- alarm.channel — channel name
- alarm.priority — LOW, MEDIUM, HIGH
- alarm.openingDate — when the alarm was opened (ISO 8601)

Examples:
  query: "alarm.severity eq CRITICAL"
  query: "alarm.status eq OPEN AND alarm.severity eq URGENT"
  query: "alarm.entityIdentifier like sense"`),
		mcp.WithString("query",
			mcp.Description("Filter: \"field op value\". Multiple with AND. Operators: eq, neq, like, gt, lt, gte, lte. Omit to list all."),
		),
		mcp.WithNumber("limit",
			mcp.Description("Max number of results"),
		),
		mcp.WithString("filter",
			mcp.Description("Advanced: raw OpenGate JSON filter. Only for OR/nested queries. Overrides 'query'."),
		),
	)
}

func alarmsSearchHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()

		filter, err := mcpBuildFilter(args)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid query: %v", err)), nil
		}

		resp, err := c.SearchAlarms(filter)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
		}

		result, err := json.Marshal(resp.Alarms)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshaling result: %v", err)), nil
		}
		return mcp.NewToolResultText(string(result)), nil
	}
}

// --- summary ---

func alarmsSummaryTool() mcp.Tool {
	return mcp.NewTool("alarms_summary",
		mcp.WithDescription("Get aggregated alarm counts grouped by severity, status, rule, and name. Useful for a quick overview of the alarm situation."),
		mcp.WithString("query",
			mcp.Description("Optional filter: \"field op value\". Example: \"alarm.status eq OPEN\". Omit for full summary."),
		),
		mcp.WithString("filter",
			mcp.Description("Advanced: raw OpenGate JSON filter. Overrides 'query'."),
		),
	)
}

func alarmsSummaryHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()

		filter, err := mcpBuildFilter(args)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid query: %v", err)), nil
		}

		resp, err := c.SummaryAlarms(filter)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("summary failed: %v", err)), nil
		}

		result, err := json.Marshal(resp)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshaling result: %v", err)), nil
		}
		return mcp.NewToolResultText(string(result)), nil
	}
}

// --- attend ---

func alarmsAttendTool() mcp.Tool {
	return mcp.NewTool("alarms_attend",
		mcp.WithDescription("Mark one or more alarms as attended. Changes alarm status from OPEN to ATTEND."),
		mcp.WithString("ids",
			mcp.Description("Comma-separated alarm UUIDs to attend"),
			mcp.Required(),
		),
		mcp.WithString("notes",
			mcp.Description("Optional notes for the attend action"),
		),
	)
}

func alarmsAttendHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		return handleAlarmAction(c, "attend", args)
	}
}

// --- close ---

func alarmsCloseTool() mcp.Tool {
	return mcp.NewTool("alarms_close",
		mcp.WithDescription("Close one or more alarms. Changes alarm status to CLOSED."),
		mcp.WithString("ids",
			mcp.Description("Comma-separated alarm UUIDs to close"),
			mcp.Required(),
		),
		mcp.WithString("notes",
			mcp.Description("Optional notes for the close action"),
		),
	)
}

func alarmsCloseHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		return handleAlarmAction(c, "close", args)
	}
}

func handleAlarmAction(c *client.Client, action string, args map[string]any) (*mcp.CallToolResult, error) {
	idsStr, _ := args["ids"].(string)
	if idsStr == "" {
		return mcp.NewToolResultError("ids is required"), nil
	}

	var ids []string
	for _, id := range strings.Split(idsStr, ",") {
		if id = strings.TrimSpace(id); id != "" {
			ids = append(ids, id)
		}
	}

	notes, _ := args["notes"].(string)

	var resp *client.AlarmActionResponse
	var err error
	if action == "attend" {
		resp, err = c.AttendAlarms(ids, notes)
	} else {
		resp, err = c.CloseAlarms(ids, notes)
	}
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("%s failed: %v", action, err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Success: %d, Errors: %d", resp.Result.Successful, resp.Result.Error.Count)), nil
}
