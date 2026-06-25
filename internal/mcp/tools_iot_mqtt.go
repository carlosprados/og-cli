package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/carlosprados/og-cli/pkg/opengate"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerIoTMQTTTools adds the MQTT south-plane tools. host is the profile host
// (scheme stripped for the broker); apiKey is the device/user API key used as the
// MQTT password (username = device id).
func registerIoTMQTTTools(r *registrar) {
	r.tool(tsIoT, iotMQTTPublishTool(), iotMQTTPublishHandler(r.p))
	r.tool(tsIoT, iotMQTTSubscribeTool(), iotMQTTSubscribeHandler(r.p))
	r.tool(tsIoT, iotMQTTDeviceTool(), iotMQTTDeviceHandler(r.p))
}

func mqttArgInt(args map[string]any, key string, def int) int {
	if v, ok := args[key].(float64); ok && v > 0 {
		return int(v)
	}
	return def
}

// mqttArgBool reads a bool arg defaulting to def when absent. tls defaults to true
// (the broker requires it); insecure defaults to false (the broker serves a valid
// public chain, so TLS verifies against the system roots).
func mqttArgBool(args map[string]any, key string, def bool) bool {
	if v, ok := args[key].(bool); ok {
		return v
	}
	return def
}

// mqttConnect builds an MQTT client for an MCP tool call (TLS on, verification on).
func mqttConnect(broker, deviceID, apiKey string, args map[string]any) (*opengate.MQTTClient, error) {
	useTLS := mqttArgBool(args, "tls", true)
	insecure := mqttArgBool(args, "insecure", false)
	caFile, _ := args["ca_file"].(string)
	return opengate.NewMQTTClient(broker, 0, useTLS, insecure, caFile, deviceID, apiKey)
}

// --- publish ---

func iotMQTTPublishTool() mcp.Tool {
	return mcp.NewTool("iot_mqtt_publish",
		mcp.WithDescription(`Publish a message to OpenGate over MQTT (South plane). Auth uses device_id as username and the profile API key as password.

Default topic is odm/iot/<device_id>; override 'topic' to publish to a connector function's custom south route (topics are NOT fixed when CFs define their own southCriterias). Provide the body as 'payload' (a full IoT collection JSON) or 'raw' (a literal string body).`),
		mcp.WithString("device_id", mcp.Description("Device identifier"), mcp.Required()),
		mcp.WithString("payload", mcp.Description(`Full IoT JSON, e.g. {"version":"1.0.0","datastreams":[{"id":"wt","datapoints":[{"value":25.3}]}]}`)),
		mcp.WithString("raw", mcp.Description("Literal string body to publish verbatim (for custom CF routes)")),
		mcp.WithString("topic", mcp.Description("Topic to publish to (default: odm/iot/<device_id>)")),
		mcp.WithBoolean("tls", mcp.Description("Use TLS (port 8883)")),
	)
}

func iotMQTTPublishHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		broker := opengate.MQTTHostFromProfile(p.host)
		apiKey, errRes := p.apiKey(ctx)
		if errRes != nil {
			return errRes, nil
		}
		if apiKey == "" {
			return mcp.NewToolResultError("no API key available. Login first."), nil
		}
		args := request.GetArguments()
		deviceID, _ := args["device_id"].(string)
		if deviceID == "" {
			return mcp.NewToolResultError("device_id is required"), nil
		}
		payload, _ := args["payload"].(string)
		raw, _ := args["raw"].(string)
		if payload == "" && raw == "" {
			return mcp.NewToolResultError("provide 'payload' (IoT JSON) or 'raw' (literal body)"), nil
		}
		body := []byte(raw)
		if raw == "" {
			body = []byte(payload)
		}
		topic, _ := args["topic"].(string)
		if topic == "" {
			topic = opengate.MQTTDataTopic(deviceID)
		}
		client, err := mqttConnect(broker, deviceID, apiKey, args)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("connect failed: %v", err)), nil
		}
		defer client.Disconnect()
		if err := client.Publish(topic, body, 0); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("publish failed: %v", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Published %d bytes to %s", len(body), topic)), nil
	}
}

// --- subscribe (bounded) ---

func iotMQTTSubscribeTool() mcp.Tool {
	return mcp.NewTool("iot_mqtt_subscribe",
		mcp.WithDescription(`Subscribe to an MQTT topic and collect up to 'count' messages or until 'timeout_seconds' elapses, then return them. Default topic is odm/request/<device_id> (incoming operations); override 'topic' to observe any topic, including a connector function's custom south route. Use this to debug CF/operation flows.`),
		mcp.WithString("device_id", mcp.Description("Device identifier (MQTT username)"), mcp.Required()),
		mcp.WithString("topic", mcp.Description("Topic to subscribe to (default: odm/request/<device_id>)")),
		mcp.WithNumber("count", mcp.Description("Max messages to collect (default 5)")),
		mcp.WithNumber("timeout_seconds", mcp.Description("Max seconds to wait (default 10)")),
		mcp.WithBoolean("tls", mcp.Description("Use TLS (port 8883)")),
	)
}

func iotMQTTSubscribeHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		broker := opengate.MQTTHostFromProfile(p.host)
		apiKey, errRes := p.apiKey(ctx)
		if errRes != nil {
			return errRes, nil
		}
		if apiKey == "" {
			return mcp.NewToolResultError("no API key available. Login first."), nil
		}
		args := request.GetArguments()
		deviceID, _ := args["device_id"].(string)
		if deviceID == "" {
			return mcp.NewToolResultError("device_id is required"), nil
		}
		topic, _ := args["topic"].(string)
		if topic == "" {
			topic = opengate.MQTTRequestTopic(deviceID)
		}
		count := mqttArgInt(args, "count", 5)
		timeout := mqttArgInt(args, "timeout_seconds", 10)
		client, err := mqttConnect(broker, deviceID, apiKey, args)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("connect failed: %v", err)), nil
		}
		defer client.Disconnect()

		type msg struct {
			Topic   string `json:"topic"`
			Payload string `json:"payload"`
		}
		var mu sync.Mutex
		var msgs []msg
		done := make(chan struct{})
		var once sync.Once

		if err := client.Subscribe(topic, 0, func(t string, p []byte) {
			mu.Lock()
			msgs = append(msgs, msg{Topic: t, Payload: string(p)})
			n := len(msgs)
			mu.Unlock()
			if n >= count {
				once.Do(func() { close(done) })
			}
		}); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("subscribe failed: %v", err)), nil
		}

		select {
		case <-done:
		case <-time.After(time.Duration(timeout) * time.Second):
		case <-ctx.Done():
		}

		mu.Lock()
		defer mu.Unlock()
		out, _ := json.Marshal(msgs)
		return mcp.NewToolResultText(fmt.Sprintf("Collected %d message(s) on %s:\n%s", len(msgs), topic, string(out))), nil
	}
}

// --- virtual device (bounded) ---

func iotMQTTDeviceTool() mcp.Tool {
	return mcp.NewTool("iot_mqtt_device",
		mcp.WithDescription(`Act as a virtual MQTT device: subscribe to odm/request/<device_id>, auto-answer each operation on odm/response/<device_id> (echoing name+id), and return the handled operations. Bounded: stops after 'max_operations' or 'timeout_seconds'. Use this to satisfy operation jobs (REBOOT_EQUIPMENT, REFRESH_INFO, ...) launched against the device.`),
		mcp.WithString("device_id", mcp.Description("Device identifier (MQTT username)"), mcp.Required()),
		mcp.WithNumber("max_operations", mcp.Description("Stop after answering N operations (default 1)")),
		mcp.WithNumber("timeout_seconds", mcp.Description("Max seconds to run (default 30)")),
		mcp.WithString("result", mcp.Description("resultCode to answer with: SUCCESSFUL (default) or ERROR")),
		mcp.WithBoolean("tls", mcp.Description("Use TLS (port 8883)")),
	)
}

func iotMQTTDeviceHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		broker := opengate.MQTTHostFromProfile(p.host)
		apiKey, errRes := p.apiKey(ctx)
		if errRes != nil {
			return errRes, nil
		}
		if apiKey == "" {
			return mcp.NewToolResultError("no API key available. Login first."), nil
		}
		args := request.GetArguments()
		deviceID, _ := args["device_id"].(string)
		if deviceID == "" {
			return mcp.NewToolResultError("device_id is required"), nil
		}
		maxOps := mqttArgInt(args, "max_operations", 1)
		timeout := mqttArgInt(args, "timeout_seconds", 30)
		result, _ := args["result"].(string)
		if result == "" {
			result = "SUCCESSFUL"
		}
		client, err := mqttConnect(broker, deviceID, apiKey, args)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("connect failed: %v", err)), nil
		}
		defer client.Disconnect()

		respTopic := opengate.MQTTResponseTopic(deviceID)
		type handled struct {
			Name string `json:"name"`
			ID   string `json:"id"`
		}
		var mu sync.Mutex
		var ops []handled
		done := make(chan struct{})
		var once sync.Once

		if err := client.Subscribe(opengate.MQTTRequestTopic(deviceID), 0, func(_ string, p []byte) {
			req, err := opengate.ParseOperationRequest(p)
			if err != nil {
				return
			}
			op := req.Operation.Request
			resp := opengate.BuildOperationResponse(deviceID, op.Name, op.ID, result, "")
			_ = client.Publish(respTopic, resp, 0)
			mu.Lock()
			ops = append(ops, handled{Name: op.Name, ID: op.ID})
			n := len(ops)
			mu.Unlock()
			if n >= maxOps {
				once.Do(func() { close(done) })
			}
		}); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("subscribe failed: %v", err)), nil
		}

		select {
		case <-done:
		case <-time.After(time.Duration(timeout) * time.Second):
		case <-ctx.Done():
		}

		mu.Lock()
		defer mu.Unlock()
		out, _ := json.Marshal(ops)
		return mcp.NewToolResultText(fmt.Sprintf("Answered %d operation(s) with %s on %s:\n%s", len(ops), result, respTopic, string(out))), nil
	}
}
