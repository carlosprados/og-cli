package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/carlosprados/og-cli/internal/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerIoTTools(s *server.MCPServer, host, apiKey string) {
	s.AddTool(iotCollectTool(), iotCollectHandler(host, apiKey))
	s.AddTool(iotCollectPayloadTool(), iotCollectPayloadHandler(host, apiKey))
}

// --- collect simple ---

func iotCollectTool() mcp.Tool {
	return mcp.NewTool("iot_collect",
		mcp.WithDescription(`Send a single data point to a device datastream via the OpenGate South API.
Uses X-ApiKey authentication. The API key is obtained from the login response.

Examples:
  iot_collect(device_id: "sense-001", datastream_id: "wt", value: "25.3")
  iot_collect(device_id: "sense-001", datastream_id: "wp", value: "1013")`),
		mcp.WithString("device_id",
			mcp.Description("Device identifier (e.g. \"sense-001\")"),
			mcp.Required(),
		),
		mcp.WithString("datastream_id",
			mcp.Description("Datastream identifier (e.g. \"wt\" for temperature)"),
			mcp.Required(),
		),
		mcp.WithString("value",
			mcp.Description("Value to send (number, boolean, or string)"),
			mcp.Required(),
		),
	)
}

func iotCollectHandler(host, apiKey string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if apiKey == "" {
			return mcp.NewToolResultError("no API key available. Login first."), nil
		}

		args := request.GetArguments()
		deviceID, _ := args["device_id"].(string)
		datastreamID, _ := args["datastream_id"].(string)
		rawValue, _ := args["value"].(string)

		if deviceID == "" || datastreamID == "" || rawValue == "" {
			return mcp.NewToolResultError("device_id, datastream_id, and value are required"), nil
		}

		value := mcpParseValue(rawValue)

		if err := client.CollectSimple(host, apiKey, deviceID, datastreamID, value); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("collect failed: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Sent %v to %s/%s", value, deviceID, datastreamID)), nil
	}
}

// --- collect payload ---

func iotCollectPayloadTool() mcp.Tool {
	return mcp.NewTool("iot_collect_payload",
		mcp.WithDescription(`Send a full IoT payload to a device via the OpenGate South API.
The payload follows the OpenGate collection format with version, datastreams, and datapoints.`),
		mcp.WithString("device_id",
			mcp.Description("Device identifier"),
			mcp.Required(),
		),
		mcp.WithString("payload",
			mcp.Description(`Full IoT JSON payload. Example: {"version":"1.0.0","datastreams":[{"id":"wt","datapoints":[{"value":25.3}]}]}`),
			mcp.Required(),
		),
	)
}

func iotCollectPayloadHandler(host, apiKey string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if apiKey == "" {
			return mcp.NewToolResultError("no API key available. Login first."), nil
		}

		args := request.GetArguments()
		deviceID, _ := args["device_id"].(string)
		payloadStr, _ := args["payload"].(string)

		if deviceID == "" || payloadStr == "" {
			return mcp.NewToolResultError("device_id and payload are required"), nil
		}

		var payload client.IoTPayload
		if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid payload: %v", err)), nil
		}

		if err := client.CollectIoT(host, apiKey, deviceID, payload); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("collect failed: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Sent IoT data to %s (%d datastreams)", deviceID, len(payload.Datastreams))), nil
	}
}

func mcpParseValue(s string) any {
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		return v
	}
	if v, err := strconv.ParseBool(s); err == nil {
		return v
	}
	return s
}
