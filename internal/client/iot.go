package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const collectPath = "/south/v80/devices/%s/collect/iot"

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
// Uses X-ApiKey authentication instead of JWT Bearer.
func CollectIoT(host, apiKey, deviceID string, payload IoTPayload) error {
	url := strings.TrimRight(host, "/") + fmt.Sprintf(collectPath, deviceID)

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling IoT payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-ApiKey", apiKey)

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending IoT data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var buf [1024]byte
		n, _ := resp.Body.Read(buf[:])
		return fmt.Errorf("IoT collect failed (HTTP %d): %s", resp.StatusCode, string(buf[:n]))
	}

	return nil
}

// CollectSimple is a convenience function to send a single value to a single datastream.
func CollectSimple(host, apiKey, deviceID, datastreamID string, value any) error {
	payload := IoTPayload{
		Version: "1.0.0",
		Datastreams: []IoTDatastream{
			{
				ID:         datastreamID,
				Datapoints: []IoTDatapoint{{Value: value}},
			},
		},
	}
	return CollectIoT(host, apiKey, deviceID, payload)
}
