package opengate

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	datasetsBasePath = "/north/{v}/datasets/provision/organizations/%s"
	datasetPath      = "/north/{v}/datasets/provision/organizations/%s/%s"
	datasetDataPath  = "/north/{v}/datasets/provision/organizations/%s/%s/data"
)

// Dataset represents a dataset definition.
type Dataset struct {
	Identifier       string     `json:"identifier,omitempty"`
	Name             string     `json:"name"`
	Description      string     `json:"description,omitempty"`
	OrganizationID   string     `json:"organizationId,omitempty"`
	IdentifierColumn string     `json:"identifierColumn,omitempty"`
	Columns          []DSColumn `json:"columns,omitempty"`
}

// DSColumn represents a column in a dataset.
type DSColumn struct {
	Path   string `json:"path"`
	Name   string `json:"name"`
	Filter string `json:"filter,omitempty"`
	Sort   bool   `json:"sort,omitempty"`
	Type   string `json:"type,omitempty"`
}

// DatasetListResponse is the response from the list endpoint.
type DatasetListResponse struct {
	Datasets []Dataset `json:"datasets"`
}

// DatasetDataResponse is the tabular response from the data endpoint.
type DatasetDataResponse struct {
	Columns []string `json:"columns"`
	Data    [][]any  `json:"data"`
	Page    *Page    `json:"page,omitempty"`
}

// ListDatasets returns all datasets in an organization.
func (c *Client) ListDatasets(ctx context.Context, orgName string) (*DatasetListResponse, error) {
	path := fmt.Sprintf(datasetsBasePath, orgName)

	data, statusCode, err := c.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("list datasets: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	if IsEmptyResponse(data, statusCode) {
		return &DatasetListResponse{}, nil
	}

	var resp DatasetListResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing datasets list: %w", err)
	}
	return &resp, nil
}

// GetDataset retrieves a single dataset by org and identifier.
func (c *Client) GetDataset(ctx context.Context, orgName, id string) (*Dataset, error) {
	path := fmt.Sprintf(datasetPath, orgName, id)

	data, statusCode, err := c.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("get dataset: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}

	var ds Dataset
	if err := json.Unmarshal(data, &ds); err != nil {
		return nil, fmt.Errorf("parsing dataset: %w", err)
	}
	return &ds, nil
}

// CreateDataset creates a new dataset.
func (c *Client) CreateDataset(ctx context.Context, orgName string, body json.RawMessage) error {
	path := fmt.Sprintf(datasetsBasePath, orgName)

	data, statusCode, err := c.Post(ctx, path, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("create dataset: %w", err)
	}
	return CheckResponse(data, statusCode)
}

// UpdateDataset updates an existing dataset.
func (c *Client) UpdateDataset(ctx context.Context, orgName, id string, body json.RawMessage) error {
	path := fmt.Sprintf(datasetPath, orgName, id)

	data, statusCode, err := c.Put(ctx, path, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("update dataset: %w", err)
	}
	return CheckResponse(data, statusCode)
}

// DeleteDataset deletes a dataset.
func (c *Client) DeleteDataset(ctx context.Context, orgName, id string) error {
	path := fmt.Sprintf(datasetPath, orgName, id)

	data, statusCode, err := c.Delete(ctx, path)
	if err != nil {
		return fmt.Errorf("delete dataset: %w", err)
	}
	return CheckResponse(data, statusCode)
}

// QueryDatasetData searches data in a dataset with filter/sort/limit.
func (c *Client) QueryDatasetData(ctx context.Context, orgName, id string, filter json.RawMessage) (*DatasetDataResponse, error) {
	path := fmt.Sprintf(datasetDataPath, orgName, id)

	var body string
	if filter != nil {
		body = string(filter)
	} else {
		body = "{}"
	}

	data, statusCode, err := c.Post(ctx, path, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("query dataset data: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	if IsEmptyResponse(data, statusCode) {
		return &DatasetDataResponse{}, nil
	}

	var resp DatasetDataResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing dataset data: %w", err)
	}
	return &resp, nil
}
