package client

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	timeseriesBasePath = "/north/v80/timeseries/provision/organizations/%s"
	timeseriesPath     = "/north/v80/timeseries/provision/organizations/%s/%s"
	timeseriesDataPath = "/north/v80/timeseries/provision/organizations/%s/%s/data"
	timeseriesExportPath = "/north/v80/timeseries/provision/organizations/%s/%s/export"
)

// TimeSeries represents a time series definition.
type TimeSeries struct {
	Identifier       string             `json:"identifier,omitempty"`
	Name             string             `json:"name"`
	Description      string             `json:"description,omitempty"`
	OrganizationID   string             `json:"organizationId,omitempty"`
	TimeBucket       int                `json:"timeBucket,omitempty"`
	Retention        int                `json:"retention,omitempty"`
	Origin           string             `json:"origin,omitempty"`
	BucketColumn     string             `json:"bucketColumn,omitempty"`
	BucketInitColumn string             `json:"bucketInitColumn,omitempty"`
	IdentifierColumn string             `json:"identifierColumn,omitempty"`
	Context          []TSColumn         `json:"context,omitempty"`
	Columns          []TSColumn         `json:"columns,omitempty"`
	Sorts            []TSSort           `json:"sorts,omitempty"`
}

// TSColumn represents a context or data column in a time series.
type TSColumn struct {
	Path               string `json:"path"`
	Name               string `json:"name"`
	Filter             string `json:"filter,omitempty"`
	Type               string `json:"type,omitempty"`
	AggregationFunction string `json:"aggregationFunction,omitempty"`
}

// TSSort represents a sort definition.
type TSSort struct {
	Identifier string         `json:"identifier"`
	Columns    []TSSortColumn `json:"columns"`
}

// TSSortColumn is a column reference within a sort.
type TSSortColumn struct {
	Name      string `json:"name"`
	Direction string `json:"direction"`
}

// TimeSeriesListResponse is the response from the list endpoint.
type TimeSeriesListResponse struct {
	Timeseries []TimeSeries `json:"timeseries"`
}

// TimeSeriesDataResponse is the tabular response from the data endpoint.
type TimeSeriesDataResponse struct {
	Columns []string `json:"columns"`
	Data    [][]any  `json:"data"`
	Page    *Page    `json:"page,omitempty"`
}

// ListTimeSeries returns all time series in an organization.
func (c *Client) ListTimeSeries(orgName string) (*TimeSeriesListResponse, error) {
	path := fmt.Sprintf(timeseriesBasePath, orgName) + "?expand=columns,context"

	data, statusCode, err := c.Get(path)
	if err != nil {
		return nil, fmt.Errorf("list timeseries: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	if IsEmptyResponse(data, statusCode) {
		return &TimeSeriesListResponse{}, nil
	}

	var resp TimeSeriesListResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing timeseries list: %w", err)
	}
	return &resp, nil
}

// GetTimeSeries retrieves a single time series by org and identifier.
func (c *Client) GetTimeSeries(orgName, id string) (*TimeSeries, error) {
	path := fmt.Sprintf(timeseriesPath, orgName, id)

	data, statusCode, err := c.Get(path)
	if err != nil {
		return nil, fmt.Errorf("get timeseries: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}

	var ts TimeSeries
	if err := json.Unmarshal(data, &ts); err != nil {
		return nil, fmt.Errorf("parsing timeseries: %w", err)
	}
	return &ts, nil
}

// CreateTimeSeries creates a new time series.
func (c *Client) CreateTimeSeries(orgName string, body json.RawMessage) error {
	path := fmt.Sprintf(timeseriesBasePath, orgName)

	data, statusCode, err := c.Post(path, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("create timeseries: %w", err)
	}
	return CheckResponse(data, statusCode)
}

// UpdateTimeSeries updates an existing time series.
func (c *Client) UpdateTimeSeries(orgName, id string, body json.RawMessage) error {
	path := fmt.Sprintf(timeseriesPath, orgName, id)

	data, statusCode, err := c.Put(path, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("update timeseries: %w", err)
	}
	return CheckResponse(data, statusCode)
}

// DeleteTimeSeries deletes a time series.
func (c *Client) DeleteTimeSeries(orgName, id string) error {
	path := fmt.Sprintf(timeseriesPath, orgName, id)

	data, statusCode, err := c.Delete(path)
	if err != nil {
		return fmt.Errorf("delete timeseries: %w", err)
	}
	return CheckResponse(data, statusCode)
}

// QueryTimeSeriesData searches data in a time series with filter/sort/limit.
func (c *Client) QueryTimeSeriesData(orgName, id string, filter json.RawMessage) (*TimeSeriesDataResponse, error) {
	path := fmt.Sprintf(timeseriesDataPath, orgName, id)

	var body string
	if filter != nil {
		body = string(filter)
	} else {
		body = "{}"
	}

	data, statusCode, err := c.Post(path, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("query timeseries data: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	if IsEmptyResponse(data, statusCode) {
		return &TimeSeriesDataResponse{}, nil
	}

	var resp TimeSeriesDataResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing timeseries data: %w", err)
	}
	return &resp, nil
}

// ExportTimeSeries triggers a Parquet export of a time series.
func (c *Client) ExportTimeSeries(orgName, id string, filter json.RawMessage) error {
	path := fmt.Sprintf(timeseriesExportPath, orgName, id)

	var body string
	if filter != nil {
		body = string(filter)
	} else {
		body = "{}"
	}

	data, statusCode, err := c.Post(path, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("export timeseries: %w", err)
	}
	return CheckResponse(data, statusCode)
}
