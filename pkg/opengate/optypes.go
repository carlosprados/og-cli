package opengate

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	opTypesCatalogPath = "/north/v80/operationTypes/catalog"
	searchOpTypesPath  = "/north/v80/operationTypes/search"
	opTypesPath        = "/north/v80/operationTypes/provision/organizations/%s"
	opTypePath         = "/north/v80/operationTypes/provision/organizations/%s/%s"
)

// OpTypeSummary extracts key fields from an operation type for display.
type OpTypeSummary struct {
	Name         string   `json:"name,omitempty"`
	Title        string   `json:"title,omitempty"`
	Description  string   `json:"description,omitempty"`
	ApplicableTo []string `json:"applicableTo,omitempty"`
	FromCatalog  string   `json:"fromCatalog,omitempty"`
}

// ParseOpTypeSummary extracts display fields from a raw operation type.
func ParseOpTypeSummary(raw json.RawMessage) OpTypeSummary {
	var s OpTypeSummary
	_ = json.Unmarshal(raw, &s)
	return s
}

// OpTypesCatalog returns the catalog of predefined operation types.
func (c *Client) OpTypesCatalog() (json.RawMessage, error) {
	data, statusCode, err := c.Get(opTypesCatalogPath)
	if err != nil {
		return nil, fmt.Errorf("operation types catalog: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	return data, nil
}

// SearchOpTypes searches operation types using a filter body.
func (c *Client) SearchOpTypes(filter json.RawMessage) (json.RawMessage, error) {
	var body string
	if filter != nil {
		body = string(filter)
	} else {
		body = "{}"
	}

	data, statusCode, err := c.Post(searchOpTypesPath, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("search operation types: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	if IsEmptyResponse(data, statusCode) {
		return json.RawMessage("{}"), nil
	}
	return data, nil
}

// GetOpType retrieves an operation type definition by name.
func (c *Client) GetOpType(org, name string) (json.RawMessage, error) {
	path := fmt.Sprintf(opTypePath, org, name)

	data, statusCode, err := c.Get(path)
	if err != nil {
		return nil, fmt.Errorf("get operation type: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	return data, nil
}

// CreateOpType creates a new operation type definition in an organization.
func (c *Client) CreateOpType(org string, body json.RawMessage) (json.RawMessage, error) {
	path := fmt.Sprintf(opTypesPath, org)

	data, statusCode, err := c.Post(path, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("create operation type: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	return data, nil
}

// UpdateOpType updates an existing operation type definition.
func (c *Client) UpdateOpType(org, name string, body json.RawMessage) error {
	path := fmt.Sprintf(opTypePath, org, name)

	data, statusCode, err := c.Put(path, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("update operation type: %w", err)
	}
	return CheckResponse(data, statusCode)
}

// DeleteOpType deletes an operation type definition.
func (c *Client) DeleteOpType(org, name string) error {
	path := fmt.Sprintf(opTypePath, org, name)

	data, statusCode, err := c.Delete(path)
	if err != nil {
		return fmt.Errorf("delete operation type: %w", err)
	}
	return CheckResponse(data, statusCode)
}
