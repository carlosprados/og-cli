package opengate

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	provisionProcessorsPath = "/north/{v}/provisionProcessors/provision/organizations/%s"
	provisionProcessorPath  = "/north/{v}/provisionProcessors/provision/organizations/%s/%s"
	provisionPlanPath       = "/north/{v}/provisionProcessors/provision/organizations/%s/%s/plan?numberOfEntriesToProcess=%d"
	provisionBulkPath       = "/north/{v}/provisionProcessors/provision/organizations/%s/%s/bulk"
	provisionBulkStatusPath = "/north/{v}/provisionProcessors/provision/organizations/%s/bulk/%s"
	provisionBulkDetailPath = "/north/{v}/provisionProcessors/provision/organizations/%s/bulk/%s/details"
)

// ProvisionProcessorSummary extracts key fields from a provision processor for display.
//
// A provision processor has no status field. Its identifier is provisionProcessorId
// (not "identifier"), and its spreadsheet config is nested under
// configurationParams.spreadsheet.
type ProvisionProcessorSummary struct {
	ProvisionProcessorID string `json:"provisionProcessorId,omitempty"`
	Name                 string `json:"name,omitempty"`
	SheetName            string `json:"-"`
	HeaderRow            string `json:"-"`
	ResultColumnName     string `json:"-"`
}

// ParseProvisionProcessorSummary extracts display fields from a raw provision processor.
func ParseProvisionProcessorSummary(raw json.RawMessage) ProvisionProcessorSummary {
	var pp struct {
		ProvisionProcessorID string `json:"provisionProcessorId"`
		Name                 string `json:"name"`
		ConfigurationParams  struct {
			Spreadsheet struct {
				SheetName        string          `json:"sheetName"`
				HeaderRow        json.RawMessage `json:"headerRow"`
				ResultColumnName string          `json:"resultColumnName"`
			} `json:"spreadsheet"`
		} `json:"configurationParams"`
	}
	_ = json.Unmarshal(raw, &pp)

	headerRow := strings.Trim(string(pp.ConfigurationParams.Spreadsheet.HeaderRow), `"`)
	return ProvisionProcessorSummary{
		ProvisionProcessorID: pp.ProvisionProcessorID,
		Name:                 pp.Name,
		SheetName:            pp.ConfigurationParams.Spreadsheet.SheetName,
		HeaderRow:            headerRow,
		ResultColumnName:     pp.ConfigurationParams.Spreadsheet.ResultColumnName,
	}
}

// ListProvisionProcessorsResponse is the response from the provision processors
// list endpoint. The API is inconsistent about the array key: the schema names
// it "processors" while live responses use "provisionProcessors" — we accept both.
type ListProvisionProcessorsResponse struct {
	ProvisionProcessors []json.RawMessage `json:"provisionProcessors"`
	Processors          []json.RawMessage `json:"processors"`
}

// Items returns whichever array the API populated.
func (r *ListProvisionProcessorsResponse) Items() []json.RawMessage {
	if len(r.ProvisionProcessors) > 0 {
		return r.ProvisionProcessors
	}
	return r.Processors
}

// ListProvisionProcessors lists every provision processor in an organization.
func (c *Client) ListProvisionProcessors(ctx context.Context, org string) ([]json.RawMessage, error) {
	path := fmt.Sprintf(provisionProcessorsPath, org)

	data, statusCode, err := c.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("list provision processors: %w", err)
	}
	// 404 here means "no processors for this organization", not a hard error.
	if statusCode == 404 {
		return nil, nil
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	if IsEmptyResponse(data, statusCode) {
		return nil, nil
	}

	var resp ListProvisionProcessorsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing provision processors response: %w", err)
	}
	return resp.Items(), nil
}

// GetProvisionProcessor retrieves a provision processor by identifier.
func (c *Client) GetProvisionProcessor(ctx context.Context, org, id string) (json.RawMessage, error) {
	path := fmt.Sprintf(provisionProcessorPath, org, id)

	data, statusCode, err := c.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("get provision processor: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	return data, nil
}

// CreateProvisionProcessor creates a provision processor in an organization.
func (c *Client) CreateProvisionProcessor(ctx context.Context, org string, body json.RawMessage) (json.RawMessage, error) {
	path := fmt.Sprintf(provisionProcessorsPath, org)

	data, statusCode, err := c.Post(ctx, path, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("create provision processor: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	return data, nil
}

// UpdateProvisionProcessor updates an existing provision processor.
func (c *Client) UpdateProvisionProcessor(ctx context.Context, org, id string, body json.RawMessage) error {
	path := fmt.Sprintf(provisionProcessorPath, org, id)

	data, statusCode, err := c.Put(ctx, path, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("update provision processor: %w", err)
	}
	return CheckResponse(data, statusCode)
}

// DeleteProvisionProcessor deletes a provision processor.
func (c *Client) DeleteProvisionProcessor(ctx context.Context, org, id string) error {
	path := fmt.Sprintf(provisionProcessorPath, org, id)

	data, statusCode, err := c.Delete(ctx, path)
	if err != nil {
		return fmt.Errorf("delete provision processor: %w", err)
	}
	return CheckResponse(data, statusCode)
}

// PlanProvisionBulk runs a dry-run plan for the first `rows` entries of an Excel
// file against a provision processor and returns the computed action plan as JSON.
// No data is mutated. rows defaults to 1 when <= 0.
func (c *Client) PlanProvisionBulk(ctx context.Context, org, id, filePath string, rows int) (json.RawMessage, error) {
	if rows <= 0 {
		rows = 1
	}
	path := fmt.Sprintf(provisionPlanPath, org, id, rows)

	// plan returns a JSON action plan, so Accept stays application/json (default).
	data, statusCode, _, err := c.PostMultipartFile(ctx, path, "file", filePath, "")
	if err != nil {
		return nil, fmt.Errorf("plan provision bulk: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	return data, nil
}

// RunProvisionBulk executes a full bulk from an Excel file against a provision
// processor and returns the created bulk process id (parsed from the Location header).
func (c *Client) RunProvisionBulk(ctx context.Context, org, id, filePath string) (string, error) {
	path := fmt.Sprintf(provisionBulkPath, org, id)

	// The bulk endpoint rejects the request (HTTP 409) unless Accept matches the
	// uploaded file's content type.
	data, statusCode, location, err := c.PostMultipartFile(ctx, path, "file", filePath, excelContentType(filePath))
	if err != nil {
		return "", fmt.Errorf("run provision bulk: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return "", err
	}
	return bulkIDFromLocation(location), nil
}

// bulkIDFromLocation extracts the trailing path segment of a Location header URI.
func bulkIDFromLocation(location string) string {
	location = strings.TrimRight(location, "/")
	if idx := strings.LastIndex(location, "/"); idx != -1 {
		return location[idx+1:]
	}
	return location
}

// GetProvisionBulkStatus reads the status summary of a bulk process.
func (c *Client) GetProvisionBulkStatus(ctx context.Context, org, bulkID string) (json.RawMessage, error) {
	path := fmt.Sprintf(provisionBulkStatusPath, org, bulkID)

	data, statusCode, err := c.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("get provision bulk status: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	return data, nil
}

// GetProvisionBulkDetails downloads the result Excel of a finished bulk process.
// ready is false (with nil data) when the process has not finished yet (HTTP 204).
func (c *Client) GetProvisionBulkDetails(ctx context.Context, org, bulkID string) (data []byte, ready bool, err error) {
	path := fmt.Sprintf(provisionBulkDetailPath, org, bulkID)

	// The details endpoint returns an Excel file; Accept must match it or the
	// server responds with HTTP 409.
	body, statusCode, err := c.GetWithAccept(ctx, path, excelContentType("x.xlsx"))
	if err != nil {
		return nil, false, fmt.Errorf("get provision bulk details: %w", err)
	}
	if statusCode == 204 {
		return nil, false, nil
	}
	if err := CheckResponse(body, statusCode); err != nil {
		return nil, false, err
	}
	return body, true, nil
}
