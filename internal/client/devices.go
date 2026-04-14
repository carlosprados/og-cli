package client

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	searchDevicesPath    = "/north/v80/search/devices?flattened=true"
	provisionDevicesPath = "/north/v80/provision/organizations/%s/devices?flattened=true"
	devicePath           = "/north/v80/provision/organizations/%s/devices/%s?flattened=true"
)

// SearchDevicesResponse is the response from the devices search endpoint.
type SearchDevicesResponse struct {
	Devices []json.RawMessage `json:"devices"`
	Page    *Page             `json:"page,omitempty"`
}

// DeviceSummary extracts key fields from a flattened device for display.
type DeviceSummary struct {
	Identifier string
	Name       string
	Org        string
	Status     string
}

// extractFlatValue extracts the value from the flattened OpenGate format.
// In flattened format, each field is a root-level dotted key with structure:
// { "_value": { "_current": { "value": <val> } } }
func ExtractFlatValue(raw json.RawMessage, field string) string {
	var root map[string]json.RawMessage
	if json.Unmarshal(raw, &root) != nil {
		return ""
	}

	fieldData, ok := root[field]
	if !ok {
		return ""
	}

	var wrapper struct {
		Value struct {
			Current struct {
				Value json.RawMessage `json:"value"`
			} `json:"_current"`
		} `json:"_value"`
	}
	if json.Unmarshal(fieldData, &wrapper) == nil && len(wrapper.Value.Current.Value) > 0 {
		var s string
		if json.Unmarshal(wrapper.Value.Current.Value, &s) == nil {
			return s
		}
		return strings.Trim(string(wrapper.Value.Current.Value), `"`)
	}

	return ""
}

// ParseDeviceSummary extracts key display fields from a raw flattened device.
func ParseDeviceSummary(raw json.RawMessage) DeviceSummary {
	return DeviceSummary{
		Identifier: ExtractFlatValue(raw, "provision.device.identifier"),
		Name:       ExtractFlatValue(raw, "provision.device.name"),
		Org:        ExtractFlatValue(raw, "provision.administration.organization"),
		Status:     ExtractFlatValue(raw, "provision.device.administrativeState"),
	}
}

// SearchDevices searches for devices using a filter body.
func (c *Client) SearchDevices(filter json.RawMessage) (*SearchDevicesResponse, error) {
	var body string
	if filter != nil {
		body = string(filter)
	} else {
		body = "{}"
	}

	data, statusCode, err := c.Post(searchDevicesPath, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("search devices: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	if IsEmptyResponse(data, statusCode) {
		return &SearchDevicesResponse{}, nil
	}

	var resp SearchDevicesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing search response: %w", err)
	}
	return &resp, nil
}

// GetDevice retrieves a single device by organization and identifier (flattened format).
func (c *Client) GetDevice(orgName, id string) (json.RawMessage, error) {
	path := fmt.Sprintf(devicePath, orgName, id)

	data, statusCode, err := c.Get(path)
	if err != nil {
		return nil, fmt.Errorf("get device: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}

	return data, nil
}

// CreateDevice creates a new device in the given organization.
func (c *Client) CreateDevice(orgName string, body json.RawMessage) error {
	path := fmt.Sprintf(provisionDevicesPath, orgName)

	data, statusCode, err := c.Post(path, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("create device: %w", err)
	}
	return CheckResponse(data, statusCode)
}

// UpdateDevice updates an existing device.
func (c *Client) UpdateDevice(orgName, id string, body json.RawMessage) error {
	path := fmt.Sprintf(devicePath, orgName, id)

	data, statusCode, err := c.Put(path, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("update device: %w", err)
	}
	return CheckResponse(data, statusCode)
}

// DeleteDevice deletes a device by organization and identifier.
func (c *Client) DeleteDevice(orgName, id string) error {
	path := fmt.Sprintf(devicePath, orgName, id)

	data, statusCode, err := c.Delete(path)
	if err != nil {
		return fmt.Errorf("delete device: %w", err)
	}
	return CheckResponse(data, statusCode)
}
