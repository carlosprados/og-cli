package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/carlosprados/og-cli/pkg/opengate"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerIoTTools(s *server.MCPServer, p *provider) {
	s.AddTool(iotCollectTool(), iotCollectHandler(p))
	s.AddTool(iotCollectPayloadTool(), iotCollectPayloadHandler(p))
	s.AddTool(iotCollectRawTool(), iotCollectRawHandler(p))
	registerIoTMQTTTools(s, p)
}

// --- collect raw (HTTP connector-function south route) ---

func iotCollectRawTool() mcp.Tool {
	return mcp.NewTool("iot_collect_raw",
		mcp.WithDescription(`Trigger a COLLECTION/RESPONSE connector function over its HTTP south route by POSTing a raw body to /south/v80/devices/<device_id>/<route>.

Unlike iot_collect (structured payload to collect/iot, which bypasses connector functions), this hits the CF's HTTP southCriteria path so the CF transforms the body and emits datapoints (verify with devices_search). Uses X-ApiKey auth.

NOTE: if the device does not exist, the South API returns HTTP 401 0x04 "Unauthorized user for this operation" — that wording is misleading, it usually means device-not-found, NOT a credentials problem; verify the device exists with devices_get first. To see the CF's logger output via connectors_logs, the device must be in TESTING administrativeState.`),
		mcp.WithString("device_id", mcp.Description("Device identifier"), mcp.Required()),
		mcp.WithString("route", mcp.Description("Connector function HTTP south path (e.g. \"ogcli-demo\")"), mcp.Required()),
		mcp.WithString("body", mcp.Description("Raw body to POST"), mcp.Required()),
		mcp.WithString("content_type", mcp.Description("Content-Type header (default application/json)")),
	)
}

func iotCollectRawHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		host := p.host
		apiKey, errRes := p.apiKey(ctx)
		if errRes != nil {
			return errRes, nil
		}
		if apiKey == "" {
			return mcp.NewToolResultError("no API key available. Login first."), nil
		}
		args := request.GetArguments()
		deviceID, _ := args["device_id"].(string)
		route, _ := args["route"].(string)
		body, _ := args["body"].(string)
		if deviceID == "" || route == "" || body == "" {
			return mcp.NewToolResultError("device_id, route, and body are required"), nil
		}
		contentType, _ := args["content_type"].(string)

		data, status, err := opengate.CollectRaw(host, apiKey, deviceID, route, []byte(body), contentType)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("collect-raw failed: %v", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Posted to %s/%s (HTTP %d). Response: %s", deviceID, route, status, string(data))), nil
	}
}

// --- collect simple ---

func iotCollectTool() mcp.Tool {
	return mcp.NewTool("iot_collect",
		mcp.WithDescription(`Send a single data point to a device datastream via the OpenGate South API.
Uses X-ApiKey authentication. The API key is obtained from the login response.

NOTE: collecting to a non-existent device returns HTTP 401 0x04 "Unauthorized user for this operation" — misleading wording that usually means device-not-found, NOT a credentials problem; verify with devices_get. The datastream id must exist in the org datamodel.

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

func iotCollectHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		host := p.host
		apiKey, errRes := p.apiKey(ctx)
		if errRes != nil {
			return errRes, nil
		}
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

		if err := opengate.CollectSimple(host, apiKey, deviceID, datastreamID, value); err != nil {
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

func iotCollectPayloadHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		host := p.host
		apiKey, errRes := p.apiKey(ctx)
		if errRes != nil {
			return errRes, nil
		}
		if apiKey == "" {
			return mcp.NewToolResultError("no API key available. Login first."), nil
		}

		args := request.GetArguments()
		deviceID, _ := args["device_id"].(string)
		payloadStr, _ := args["payload"].(string)

		if deviceID == "" || payloadStr == "" {
			return mcp.NewToolResultError("device_id and payload are required"), nil
		}

		var payload opengate.IoTPayload
		if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid payload: %v", err)), nil
		}

		if err := opengate.CollectIoT(host, apiKey, deviceID, payload); err != nil {
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
