package client

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	searchDatamodelsPath    = "/north/v80/search/datamodels"
	provisionDatamodelsPath = "/north/v80/provision/organizations/%s/datamodels"
	datamodelPath           = "/north/v80/provision/organizations/%s/datamodels/%s"
)

// Datamodel represents an OpenGate data model.
type Datamodel struct {
	Identifier           string       `json:"identifier"`
	OrganizationName     string       `json:"organizationName,omitempty"`
	Name                 string       `json:"name"`
	Description          string       `json:"description,omitempty"`
	Version              string       `json:"version"`
	AllowedResourceTypes []string     `json:"allowedResourceTypes,omitempty"`
	Categories           []Category   `json:"categories,omitempty"`
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
	QRating     json.RawMessage `json:"qrating,omitempty"`
	Encryption  json.RawMessage `json:"encryption,omitempty"`
	Views       json.RawMessage `json:"views,omitempty"`
	Icon        json.RawMessage `json:"icon,omitempty"`
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

// Page holds pagination info.
type Page struct {
	Number int `json:"number,omitempty"`
}

// SearchDatamodels searches for datamodels using a filter body.
// If filter is nil, all datamodels are returned.
func (c *Client) SearchDatamodels(filter json.RawMessage) (*SearchDatamodelsResponse, error) {
	var body string
	if filter != nil {
		body = string(filter)
	} else {
		body = "{}"
	}

	data, statusCode, err := c.Post(searchDatamodelsPath, strings.NewReader(body))
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
func (c *Client) GetDatamodel(orgName, id string) (*Datamodel, error) {
	path := fmt.Sprintf(datamodelPath, orgName, id)

	data, statusCode, err := c.Get(path)
	if err != nil {
		return nil, fmt.Errorf("get datamodel: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}

	var dm Datamodel
	if err := json.Unmarshal(data, &dm); err != nil {
		return nil, fmt.Errorf("parsing datamodel: %w", err)
	}
	return &dm, nil
}

// CreateDatamodel creates a new datamodel in the given organization.
// The body should be the full JSON datamodel payload.
func (c *Client) CreateDatamodel(orgName string, body json.RawMessage) error {
	path := fmt.Sprintf(provisionDatamodelsPath, orgName)

	data, statusCode, err := c.Post(path, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("create datamodel: %w", err)
	}
	return CheckResponse(data, statusCode)
}

// UpdateDatamodel updates an existing datamodel.
func (c *Client) UpdateDatamodel(orgName, id string, body json.RawMessage) error {
	path := fmt.Sprintf(datamodelPath, orgName, id)

	data, statusCode, err := c.Put(path, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("update datamodel: %w", err)
	}
	return CheckResponse(data, statusCode)
}

// DeleteDatamodel deletes a datamodel by organization and identifier.
func (c *Client) DeleteDatamodel(orgName, id string) error {
	path := fmt.Sprintf(datamodelPath, orgName, id)

	data, statusCode, err := c.Delete(path)
	if err != nil {
		return fmt.Errorf("delete datamodel: %w", err)
	}
	return CheckResponse(data, statusCode)
}
