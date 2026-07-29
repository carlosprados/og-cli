package opengate

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const (
	collectPath    = "/south/{v}/devices/%s/collect/iot"
	collectRawPath = "/south/{v}/devices/%s/%s"
)

// IoTPayload is the body for the data collection endpoint.
type IoTPayload struct {
	Version     string          `json:"version"`
	Device      string          `json:"device,omitempty"`
	Datastreams []IoTDatastream `json:"datastreams"`
}

// IoTDatastream is a single datastream with datapoints.
type IoTDatastream struct {
	ID         string         `json:"id"`
	Feed       string         `json:"feed,omitempty"`
	Datapoints []IoTDatapoint `json:"datapoints"`
}

// IoTDatapoint is a single measurement.
type IoTDatapoint struct {
	At    *int64 `json:"at,omitempty"`
	Value any    `json:"value"`
}

// CollectIoT sends IoT data to a device via the South API.
//
// South calls authenticate with the device/server API key rather than a JWT, so
// apiKey is a parameter instead of coming from the client: the key belongs to the
// device, while the client's credential belongs to the user.
//
// These are Client methods rather than package functions so the South plane
// inherits the client's API version, TLS settings and retry policy — as package
// functions they had the version hardcoded.
func (c *Client) CollectIoT(ctx context.Context, apiKey, deviceID string, payload IoTPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling IoT payload: %w", err)
	}

	path := fmt.Sprintf(collectPath, deviceID)
	data, status, err := c.doRequestWithAuth(ctx, http.MethodPost, path,
		strings.NewReader(string(body)), authHeader{name: "X-ApiKey", value: apiKey}, "")
	if err != nil {
		return fmt.Errorf("sending IoT data: %w", err)
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("IoT collect failed (HTTP %d): %s", status, excerpt(data))
	}
	return nil
}

// CollectRaw posts a raw body to a connector function's custom south route
// (its HTTP southCriteria path), i.e. POST /south/{v}/devices/{id}/{route}.
// Unlike CollectIoT (which posts a structured collection payload to collect/iot
// and bypasses connector functions), this triggers a COLLECTION/RESPONSE
// connector function that matches the route.
//
// Returns the response body (a CF may return content) and the status code.
func (c *Client) CollectRaw(ctx context.Context, apiKey, deviceID, route string, body []byte, contentType string) ([]byte, int, error) {
	route = strings.TrimPrefix(route, "/")
	if route == "" {
		return nil, 0, fmt.Errorf("route is required (the connector function's south path)")
	}
	if contentType == "" {
		contentType = "application/json"
	}

	path := fmt.Sprintf(collectRawPath, deviceID, route)
	data, status, err := c.doRequestFull(ctx, http.MethodPost, path, strings.NewReader(string(body)),
		authHeader{name: "X-ApiKey", value: apiKey}, "", contentType)
	if err != nil {
		return nil, 0, fmt.Errorf("sending raw south data: %w", err)
	}
	if status < 200 || status >= 300 {
		return data, status, fmt.Errorf("collect-raw failed (HTTP %d): %s", status, excerpt(data))
	}
	return data, status, nil
}

// CollectSimple sends a single value to a single datastream.
func (c *Client) CollectSimple(ctx context.Context, apiKey, deviceID, datastreamID string, value any) error {
	payload := IoTPayload{
		Version: "1.0.0",
		Datastreams: []IoTDatastream{
			{
				ID:         datastreamID,
				Datapoints: []IoTDatapoint{{Value: value}},
			},
		},
	}
	return c.CollectIoT(ctx, apiKey, deviceID, payload)
}

// excerpt caps an error body so a large HTML error page does not become the whole
// error message.
func excerpt(data []byte) string {
	const max = 1024
	if len(data) <= max {
		return string(data)
	}
	return string(data[:max]) + "…"
}
