package opengate

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	searchDatamodelsPath    = "/north/{v}/search/datamodels"
	provisionDatamodelsPath = "/north/{v}/provision/organizations/%s/datamodels"
	datamodelPath           = "/north/{v}/provision/organizations/%s/datamodels/%s"
)

// Datamodel represents an OpenGate data model.
type Datamodel struct {
	Identifier           string     `json:"identifier"`
	OrganizationName     string     `json:"organizationName,omitempty"`
	Name                 string     `json:"name"`
	Description          string     `json:"description,omitempty"`
	Version              string     `json:"version"`
	AllowedResourceTypes []string   `json:"allowedResourceTypes,omitempty"`
	Categories           []Category `json:"categories,omitempty"`
}

// Category groups datastream templates within a data model.
type Category struct {
	Identifier  string       `json:"identifier"`
	Name        string       `json:"name,omitempty"`
	Datastreams []Datastream `json:"datastreams,omitempty"`
}

// Datastream defines a data stream template.
type Datastream struct {
	Identifier  string          `json:"identifier"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Period      string          `json:"period,omitempty"`
	Access      string          `json:"access,omitempty"`
	Schema      json.RawMessage `json:"schema,omitempty"`
	Storage     *Storage        `json:"storage,omitempty"`
	Unit        *Unit           `json:"unit,omitempty"`
	Tags        []string        `json:"tags,omitempty"`
	Modifiable  *bool           `json:"modifiable,omitempty"`
	Calculated  *bool           `json:"calculated,omitempty"`
	Required    *bool           `json:"required,omitempty"`

	// Indexed and NotFilterable are returned by the platform on every
	// datastream but are absent from the published OpenAPI spec, so they were
	// missing here and silently dropped on every round-trip. Pointers, so an
	// absent field stays absent rather than being invented as false.
	Indexed       *bool `json:"indexed,omitempty"`
	NotFilterable *bool `json:"notFilterable,omitempty"`

	QRating    json.RawMessage `json:"qrating,omitempty"`
	Encryption json.RawMessage `json:"encryption,omitempty"`
	Views      json.RawMessage `json:"views,omitempty"`
	Icon       json.RawMessage `json:"icon,omitempty"`
}

// Storage defines the data retention policy.
type Storage struct {
	Period string `json:"period"`
	Total  int    `json:"total,omitempty"`
}

// Unit describes measurement units.
type Unit struct {
	Type   string `json:"type,omitempty"`
	Label  string `json:"label,omitempty"`
	Symbol string `json:"symbol,omitempty"`
}

// SearchDatamodelsResponse is the response from the search endpoint.
type SearchDatamodelsResponse struct {
	Datamodels []Datamodel `json:"datamodels"`
	Page       *Page       `json:"page,omitempty"`
}

// SearchDatamodels searches for datamodels using a filter body.
// If filter is nil, all datamodels are returned.
func (c *Client) SearchDatamodels(ctx context.Context, filter json.RawMessage) (*SearchDatamodelsResponse, error) {
	var body string
	if filter != nil {
		body = string(filter)
	} else {
		body = "{}"
	}

	data, statusCode, err := c.Post(ctx, searchDatamodelsPath, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("search datamodels: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	if IsEmptyResponse(data, statusCode) {
		return &SearchDatamodelsResponse{}, nil
	}

	var resp SearchDatamodelsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing search response: %w", err)
	}
	return &resp, nil
}

// GetDatamodel retrieves a single datamodel by organization and identifier.
func (c *Client) GetDatamodel(ctx context.Context, orgName, id string) (*Datamodel, error) {
	path := fmt.Sprintf(datamodelPath, orgName, id)

	data, statusCode, err := c.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("get datamodel: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	if err := notFoundIfEmpty(data, statusCode, "datamodel", id); err != nil {
		return nil, err
	}

	var dm Datamodel
	if err := json.Unmarshal(data, &dm); err != nil {
		return nil, fmt.Errorf("parsing datamodel: %w", err)
	}
	return &dm, nil
}

// GetDatamodelRaw retrieves a single datamodel as the exact bytes the platform
// returned, with no struct in the way.
//
// GetDatamodel decodes into Datamodel, so any field the platform adds and this
// package does not yet know about is dropped on the way out. That is fine for
// display and for editing, and wrong for a backup: use this when the caller
// needs fidelity rather than typed access.
func (c *Client) GetDatamodelRaw(ctx context.Context, orgName, id string) (json.RawMessage, error) {
	path := fmt.Sprintf(datamodelPath, orgName, id)

	data, statusCode, err := c.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("get datamodel: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	if err := notFoundIfEmpty(data, statusCode, "datamodel", id); err != nil {
		return nil, err
	}
	return data, nil
}

// CreateDatamodel creates a new datamodel in the given organization.
// The body should be the full JSON datamodel payload.
func (c *Client) CreateDatamodel(ctx context.Context, orgName string, body json.RawMessage) error {
	path := fmt.Sprintf(provisionDatamodelsPath, orgName)

	data, statusCode, err := c.Post(ctx, path, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("create datamodel: %w", err)
	}
	return CheckResponse(data, statusCode)
}

// UpdateDatamodel updates an existing datamodel.
func (c *Client) UpdateDatamodel(ctx context.Context, orgName, id string, body json.RawMessage) error {
	path := fmt.Sprintf(datamodelPath, orgName, id)

	data, statusCode, err := c.Put(ctx, path, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("update datamodel: %w", err)
	}
	return CheckResponse(data, statusCode)
}

// DeleteDatamodel deletes a datamodel by organization and identifier.
func (c *Client) DeleteDatamodel(ctx context.Context, orgName, id string) error {
	path := fmt.Sprintf(datamodelPath, orgName, id)

	data, statusCode, err := c.Delete(ctx, path)
	if err != nil {
		return fmt.Errorf("delete datamodel: %w", err)
	}
	return CheckResponse(data, statusCode)
}
