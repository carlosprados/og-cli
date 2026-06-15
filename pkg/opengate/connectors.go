package opengate

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	connectorFunctionsCatalogPath = "/north/v80/connectorFunctions/provision/catalog"
	connectorFunctionsPath        = "/north/v80/connectorFunctions/provision/organizations/%s/channels/%s"
	connectorFunctionPath         = "/north/v80/connectorFunctions/provision/organizations/%s/channels/%s/%s"
)

// ConnectorFunctionSummary extracts key fields from a connector function for display.
//
// The API is inconsistent about the name field: create/update bodies use "name"
// while list/get responses sometimes echo "connectorFunctionName". We accept both.
type ConnectorFunctionSummary struct {
	Identifier            string `json:"identifier,omitempty"`
	Name                  string `json:"name,omitempty"`
	ConnectorFunctionName string `json:"connectorFunctionName,omitempty"`
	Type                  string `json:"type,omitempty"` // COLLECTION | REQUEST | RESPONSE
	OperationalStatus     string `json:"operationalStatus,omitempty"`
	OperationName         string `json:"operationName,omitempty"`
	PayloadType           string `json:"payloadType,omitempty"`
	Description           string `json:"description,omitempty"`
}

// DisplayName returns the most reliable name field available.
func (s ConnectorFunctionSummary) DisplayName() string {
	if s.Name != "" {
		return s.Name
	}
	return s.ConnectorFunctionName
}

// ParseConnectorFunctionSummary extracts display fields from a raw connector function.
func ParseConnectorFunctionSummary(raw json.RawMessage) ConnectorFunctionSummary {
	var s ConnectorFunctionSummary
	_ = json.Unmarshal(raw, &s)
	return s
}

// ListConnectorFunctionsResponse is the response from the connector functions list endpoint.
type ListConnectorFunctionsResponse struct {
	ConnectorFunctions []json.RawMessage `json:"connectorFunctions"`
}

// ListConnectorFunctions lists every connector function in an organization channel.
func (c *Client) ListConnectorFunctions(org, channel string) (*ListConnectorFunctionsResponse, error) {
	path := fmt.Sprintf(connectorFunctionsPath, org, channel)

	data, statusCode, err := c.Get(path)
	if err != nil {
		return nil, fmt.Errorf("list connector functions: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	if IsEmptyResponse(data, statusCode) {
		return &ListConnectorFunctionsResponse{}, nil
	}

	var resp ListConnectorFunctionsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing connector functions response: %w", err)
	}
	return &resp, nil
}

// ConnectorFunctionsCatalog returns the platform connector functions catalog
// (predefined templates, not scoped to an organization channel).
func (c *Client) ConnectorFunctionsCatalog() (json.RawMessage, error) {
	data, statusCode, err := c.Get(connectorFunctionsCatalogPath)
	if err != nil {
		return nil, fmt.Errorf("connector functions catalog: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	return data, nil
}

// GetConnectorFunction retrieves a connector function by identifier.
func (c *Client) GetConnectorFunction(org, channel, id string) (json.RawMessage, error) {
	path := fmt.Sprintf(connectorFunctionPath, org, channel, id)

	data, statusCode, err := c.Get(path)
	if err != nil {
		return nil, fmt.Errorf("get connector function: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	return data, nil
}

// CreateConnectorFunction creates a connector function in an organization channel.
func (c *Client) CreateConnectorFunction(org, channel string, body json.RawMessage) (json.RawMessage, error) {
	path := fmt.Sprintf(connectorFunctionsPath, org, channel)

	data, statusCode, err := c.Post(path, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("create connector function: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	return data, nil
}

// UpdateConnectorFunction updates an existing connector function.
func (c *Client) UpdateConnectorFunction(org, channel, id string, body json.RawMessage) error {
	path := fmt.Sprintf(connectorFunctionPath, org, channel, id)

	data, statusCode, err := c.Put(path, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("update connector function: %w", err)
	}
	return CheckResponse(data, statusCode)
}

// DeleteConnectorFunction deletes a connector function.
func (c *Client) DeleteConnectorFunction(org, channel, id string) error {
	path := fmt.Sprintf(connectorFunctionPath, org, channel, id)

	data, statusCode, err := c.Delete(path)
	if err != nil {
		return fmt.Errorf("delete connector function: %w", err)
	}
	return CheckResponse(data, statusCode)
}

// SetConnectorFunctionStatus changes a connector function's operationalStatus
// (GET + patch operationalStatus + PUT). Valid values: DISABLED, PRODUCTION, TEST.
func (c *Client) SetConnectorFunctionStatus(org, channel, id, status string) error {
	raw, err := c.GetConnectorFunction(org, channel, id)
	if err != nil {
		return err
	}

	var cf map[string]json.RawMessage
	if err := json.Unmarshal(raw, &cf); err != nil {
		return fmt.Errorf("parsing connector function: %w", err)
	}
	statusJSON, _ := json.Marshal(status)
	cf["operationalStatus"] = statusJSON

	body, err := json.Marshal(cf)
	if err != nil {
		return fmt.Errorf("marshaling connector function: %w", err)
	}
	return c.UpdateConnectorFunction(org, channel, id, body)
}
