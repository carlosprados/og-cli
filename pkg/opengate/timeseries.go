package opengate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

const (
	timeseriesBasePath   = "/north/{v}/timeseries/provision/organizations/%s"
	timeseriesPath       = "/north/{v}/timeseries/provision/organizations/%s/%s"
	timeseriesDataPath   = "/north/{v}/timeseries/provision/organizations/%s/%s/data"
	timeseriesExportPath = "/north/{v}/timeseries/provision/organizations/%s/%s/export"
)

// The list endpoint's expansions are opt-in, and which ones an instance
// accepts depends on its build.
const (
	// tsListExpand is what a current platform accepts. The single-item GET
	// returns every expansion unasked; only the list is opt-in, and without
	// sorts here the server answers with none at all.
	tsListExpand = "columns,context,sorts"
	// tsListExpandLegacy drops sorts. An older on-premises instance validates
	// expand against a whitelist with no sorts entry and rejects the whole
	// request with HTTP 400 "Invalid query parameters", so asking for sorts
	// unconditionally turns a list that would return less into a list that
	// returns nothing at all (observed on an on-premises v80 instance,
	// 2026-09-10; api.opengate.es accepts sorts). The API version does not
	// discriminate — both instances answer to v80 — so degrade on the
	// rejection itself rather than on a version check.
	tsListExpandLegacy = "columns,context"
)

// TimeSeries represents a time series definition.
type TimeSeries struct {
	Identifier       string `json:"identifier,omitempty"`
	Name             string `json:"name"`
	Description      string `json:"description,omitempty"`
	OrganizationID   string `json:"organizationId,omitempty"`
	TimeBucket       int    `json:"timeBucket,omitempty"`
	Retention        int    `json:"retention,omitempty"`
	Origin           string `json:"origin,omitempty"`
	BucketColumn     string `json:"bucketColumn,omitempty"`
	BucketInitColumn string `json:"bucketInitColumn,omitempty"`
	IdentifierColumn string `json:"identifierColumn,omitempty"`
	// Context keeps omitempty, so a time series with no context columns comes
	// back without the key rather than as `"context": []`. Dropping omitempty
	// would emit `null` whenever the field is genuinely absent, which is worse;
	// use --raw when the bytes have to match the platform exactly.
	Context []TSColumn `json:"context,omitempty"`
	Columns []TSColumn `json:"columns,omitempty"`
	Sorts   []TSSort   `json:"sorts,omitempty"`
}

// TSColumn represents a context or data column in a time series.
type TSColumn struct {
	Path                string `json:"path"`
	Name                string `json:"name"`
	Filter              string `json:"filter,omitempty"`
	Type                string `json:"type,omitempty"`
	AggregationFunction string `json:"aggregationFunction,omitempty"`
}

// TSSort represents a sort definition.
type TSSort struct {
	Identifier  string         `json:"identifier"`
	Description string         `json:"description,omitempty"`
	Columns     []TSSortColumn `json:"columns"`
	// Derived marks the sorts the platform generated itself (typically the
	// reverse of a declared one) rather than ones the user defined. It has no
	// omitempty: the platform states it on every sort, and dropping the false
	// half would turn an answered question into an unanswered one.
	Derived bool `json:"derived"`
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

// listTimeSeries performs the list request, retrying without sorts when the
// instance refuses the expansion. The extra round trip is paid only on an
// instance that does not support sorts anyway, and the caller gets a list
// missing its sorts instead of an error naming a parameter it never chose.
func (c *Client) listTimeSeries(ctx context.Context, orgName string) ([]byte, int, error) {
	base := fmt.Sprintf(timeseriesBasePath, orgName)

	data, statusCode, err := c.Get(ctx, base+"?expand="+tsListExpand)
	if err != nil {
		return nil, 0, err
	}
	if !rejectsExpand(data, statusCode) {
		return data, statusCode, nil
	}
	return c.Get(ctx, base+"?expand="+tsListExpandLegacy)
}

// rejectsExpand reports whether a response is the platform refusing the expand
// parameter itself, as opposed to any other 400. A rejection for a different
// reason is passed through untouched: retrying it would only hide it.
func rejectsExpand(data []byte, statusCode int) bool {
	if statusCode != http.StatusBadRequest {
		return false
	}
	var apiErr *APIError
	if !errors.As(CheckResponse(data, statusCode), &apiErr) {
		return false
	}
	return apiErr.HasField("expand")
}

// ListTimeSeries returns all time series in an organization.
func (c *Client) ListTimeSeries(ctx context.Context, orgName string) (*TimeSeriesListResponse, error) {
	data, statusCode, err := c.listTimeSeries(ctx, orgName)
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
func (c *Client) GetTimeSeries(ctx context.Context, orgName, id string) (*TimeSeries, error) {
	path := fmt.Sprintf(timeseriesPath, orgName, id)

	data, statusCode, err := c.Get(ctx, path)
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

// ListTimeSeriesRaw returns the time series list as the exact bytes the
// platform sent, expansions included.
func (c *Client) ListTimeSeriesRaw(ctx context.Context, orgName string) (json.RawMessage, error) {
	data, statusCode, err := c.listTimeSeries(ctx, orgName)
	if err != nil {
		return nil, fmt.Errorf("list timeseries: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	if IsEmptyResponse(data, statusCode) {
		return json.RawMessage("{}"), nil
	}
	return data, nil
}

// GetTimeSeriesRaw retrieves a time series as the exact bytes the platform
// returned.
func (c *Client) GetTimeSeriesRaw(ctx context.Context, orgName, id string) (json.RawMessage, error) {
	path := fmt.Sprintf(timeseriesPath, orgName, id)

	data, statusCode, err := c.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("get timeseries: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	return data, nil
}

// CreateTimeSeries creates a new time series.
func (c *Client) CreateTimeSeries(ctx context.Context, orgName string, body json.RawMessage) error {
	path := fmt.Sprintf(timeseriesBasePath, orgName)

	data, statusCode, err := c.Post(ctx, path, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("create timeseries: %w", err)
	}
	return CheckResponse(data, statusCode)
}

// UpdateTimeSeries updates an existing time series.
func (c *Client) UpdateTimeSeries(ctx context.Context, orgName, id string, body json.RawMessage) error {
	path := fmt.Sprintf(timeseriesPath, orgName, id)

	data, statusCode, err := c.Put(ctx, path, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("update timeseries: %w", err)
	}
	return CheckResponse(data, statusCode)
}

// DeleteTimeSeries deletes a time series.
func (c *Client) DeleteTimeSeries(ctx context.Context, orgName, id string) error {
	path := fmt.Sprintf(timeseriesPath, orgName, id)

	data, statusCode, err := c.Delete(ctx, path)
	if err != nil {
		return fmt.Errorf("delete timeseries: %w", err)
	}
	return CheckResponse(data, statusCode)
}

// QueryTimeSeriesData searches data in a time series with filter/sort/limit.
func (c *Client) QueryTimeSeriesData(ctx context.Context, orgName, id string, filter json.RawMessage) (*TimeSeriesDataResponse, error) {
	path := fmt.Sprintf(timeseriesDataPath, orgName, id)

	var body string
	if filter != nil {
		body = string(filter)
	} else {
		body = "{}"
	}

	data, statusCode, err := c.Post(ctx, path, strings.NewReader(body))
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
func (c *Client) ExportTimeSeries(ctx context.Context, orgName, id string, filter json.RawMessage) error {
	path := fmt.Sprintf(timeseriesExportPath, orgName, id)

	var body string
	if filter != nil {
		body = string(filter)
	} else {
		body = "{}"
	}

	data, statusCode, err := c.Post(ctx, path, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("export timeseries: %w", err)
	}
	return CheckResponse(data, statusCode)
}
