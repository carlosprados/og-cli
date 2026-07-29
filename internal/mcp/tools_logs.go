package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/carlosprados/og-cli/pkg/opengate"
	"github.com/mark3labs/mcp-go/mcp"
)

// functionLogsResult collects bounded execution logs for a rule or connector
// function from the functions-logger WebSocket. kind is opengate.LoggerRules or
// opengate.LoggerConnectorFunctions. It stops at 'count' lines or 'timeout_seconds'.
func functionLogsResult(ctx context.Context, c *opengate.Client, apiKey, kind, channel string, args map[string]any) (*mcp.CallToolResult, error) {
	if apiKey == "" {
		return mcp.NewToolResultError("no API key available. Login first."), nil
	}
	org, _ := args["organization"].(string)
	id, _ := args["id"].(string)
	if org == "" || id == "" {
		return mcp.NewToolResultError("organization and id are required"), nil
	}
	level, _ := args["level"].(string)
	if level == "" {
		level = "INFO"
	}
	count := 20
	if v, ok := args["count"].(float64); ok && v > 0 {
		count = int(v)
	}
	timeout := 15
	if v, ok := args["timeout_seconds"].(float64); ok && v > 0 {
		timeout = int(v)
	}

	stop := make(chan struct{})
	timer := time.AfterFunc(time.Duration(timeout)*time.Second, func() { close(stop) })
	defer timer.Stop()

	msgs, err := c.CollectFunctionLogs(ctx, apiKey, kind, org, channel, id, level, count, stop)
	if err != nil && len(msgs) == 0 {
		return mcp.NewToolResultError(fmt.Sprintf("logs failed: %v", err)), nil
	}
	out, _ := json.Marshal(msgs)
	return mcp.NewToolResultText(fmt.Sprintf("Collected %d log line(s):\n%s", len(msgs), string(out))), nil
}
