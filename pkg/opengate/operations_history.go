package opengate

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/url"
	"strings"
)

const operationsHistoryPath = "/north/{v}/search/entities/operations/history"

// MaxJobOperationsPageSize is the page-size ceiling of the per-job operations
// endpoint. It is lower than MaxPageSize, which applies to the filter-based
// searches: the platform documents "the top margin for the page size ... is
// 1000. If you setup a size attribute over this limit, you'll receive a server
// error response."
const MaxJobOperationsPageSize = 1000

// OperationStep is one step of an operation's execution.
//
// Result is the step's outcome — note the field is result, not status; status
// belongs to the Operation. Names are operation-specific: a DIAGNOSIS reports
// PROVISION, PRESENCE, MODEM and REGISTER, while a REBOOT_EQUIPMENT reports
// REBOOT, so this is deliberately a plain string and not an enumeration.
type OperationStep struct {
	Name        string          `json:"name,omitempty"`
	Title       string          `json:"title,omitempty"`
	Description string          `json:"description,omitempty"`
	Result      string          `json:"result,omitempty"`
	Timestamp   string          `json:"timestamp,omitempty"`
	Response    json.RawMessage `json:"response,omitempty"`
}

// Step result values.
const (
	StepResultSuccessful  = "SUCCESSFUL"
	StepResultError       = "ERROR"
	StepResultSkipped     = "SKIPPED"
	StepResultNotExecuted = "NOT_EXECUTED"
)

// OperationExecution holds an operation's execution timestamps.
type OperationExecution struct {
	ActivatedDate string `json:"activatedDate,omitempty"`
	StartedDate   string `json:"startedDate,omitempty"`
	FinishedDate  string `json:"finishedDate,omitempty"`
}

// OperationAttempts counts an operation's delivery attempts.
type OperationAttempts struct {
	Current int `json:"current,omitempty"`
	Total   int `json:"total,omitempty"`
}

// Operation is a single operation executed on one entity, as returned by the
// per-job operations listing and by the operations history search.
//
// Status is the lifecycle state (e.g. FINISHED) and Result the outcome (e.g.
// SUCCESSFUL); Steps carries the per-step detail.
type Operation struct {
	OperationID  string              `json:"operationId,omitempty"`
	JobID        string              `json:"jobId,omitempty"`
	Name         string              `json:"name,omitempty"`
	EntityID     string              `json:"entityId,omitempty"`
	ResourceType string              `json:"resourceType,omitempty"`
	Status       string              `json:"status,omitempty"`
	Result       string              `json:"result,omitempty"`
	Description  string              `json:"description,omitempty"`
	User         string              `json:"user,omitempty"`
	Notify       bool                `json:"notify,omitempty"`
	Date         string              `json:"date,omitempty"`
	Attempts     *OperationAttempts  `json:"attempts,omitempty"`
	Execution    *OperationExecution `json:"execution,omitempty"`
	Steps        []OperationStep     `json:"steps,omitempty"`
	Parameters   json.RawMessage     `json:"parameters,omitempty"`
}

// StepResult returns the result of the named step and whether it was present.
func (o Operation) StepResult(name string) (string, bool) {
	for _, s := range o.Steps {
		if s.Name == name {
			return s.Result, true
		}
	}
	return "", false
}

// SearchOperationsResponse is the response from the operations history search.
type SearchOperationsResponse struct {
	Operations []Operation     `json:"operations"`
	Page       *Page           `json:"page,omitempty"`
	Summary    json.RawMessage `json:"summary,omitempty"`
}

// SearchOperationsHistory searches closed operations across jobs.
//
// This is the endpoint to use for reading results back. Verified against a live
// instance: GetJobOperations returned HTTP 204 (no content) for a FINISHED job
// whose operation this endpoint returned complete with its steps. Do not read an
// empty per-job listing as "the job had no operations".
//
// It also takes a filter, so an entire job's outcome can be pulled with
// JobIDFilter and paged with limit, or many operations selected at once.
//
// Filter fields are unprefixed and the accepted set is narrow — probed live:
// jobId, entityId, operationId, resourceType and operationName (alias
// operation.name). Anything else, including status, result, user, date and any
// "operations." prefix, is rejected with HTTP 400 "Field in filter unknown".
// There is therefore no server-side filter by outcome: fetch and inspect Result
// or Steps yourself.
//
// It requests JSON explicitly. The endpoint can also emit text/plain CSV, but
// that format is unusable here: the separator collides with the contents of
// steps[].description, so rows break on any step whose description contains one.
func (c *Client) SearchOperationsHistory(ctx context.Context, filter json.RawMessage) (*SearchOperationsResponse, error) {
	body := "{}"
	if filter != nil {
		body = string(filter)
	}

	data, statusCode, err := c.PostAccepting(ctx, operationsHistoryPath, strings.NewReader(body), "application/json")
	if err != nil {
		return nil, fmt.Errorf("search operations history: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	if IsEmptyResponse(data, statusCode) {
		return &SearchOperationsResponse{}, nil
	}

	var resp SearchOperationsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing operations history: %w", err)
	}
	return &resp, nil
}

// SearchOperationsHistoryAll walks every page of an operations history search.
// The page size comes from filter's limit.size, otherwise DefaultPageSize.
func (c *Client) SearchOperationsHistoryAll(ctx context.Context, filter json.RawMessage) iter.Seq2[Operation, error] {
	return paginate(ctx, filter, func(ctx context.Context, f json.RawMessage) ([]Operation, *Page, error) {
		resp, err := c.SearchOperationsHistory(ctx, f)
		if err != nil {
			return nil, nil, err
		}
		return resp.Operations, resp.Page, nil
	})
}

// JobIDFilter builds the operations history filter for one job — the shape used
// to read back the per-device outcome of a job.
func JobIDFilter(jobID string) json.RawMessage {
	body := map[string]any{
		"filter": map[string]any{
			"and": []map[string]any{
				{"eq": map[string]any{"jobId": jobID}},
			},
		},
	}
	data, _ := json.Marshal(body) // shape is fixed and always marshalable
	return data
}

// GetJobOperationsPage lists one page of a job's operations. Unlike the search
// endpoints, this one pages with start/size query parameters rather than a body
// limit, and its size ceiling is MaxJobOperationsPageSize.
//
// A page < 0 omits the start parameter and lets the platform serve its first
// page, whichever number that is.
func (c *Client) GetJobOperationsPage(ctx context.Context, jobID string, page, size int) (*JobOperationsResponse, error) {
	if size <= 0 {
		size = MaxJobOperationsPageSize
	}
	if size > MaxJobOperationsPageSize {
		size = MaxJobOperationsPageSize
	}

	q := url.Values{}
	q.Set("size", fmt.Sprint(size))
	if page >= 0 {
		q.Set("start", fmt.Sprint(page))
	}

	path := fmt.Sprintf(jobOperationsPath, jobID) + "?" + q.Encode()
	return c.getJobOperations(ctx, path)
}

// GetJobOperationsAll walks every page of a job's operations.
//
// It starts by letting the platform serve its own first page and then follows
// the page numbers the platform reports, because the start parameter is
// documented with a default of 0 while the search endpoints count from 1.
//
// Beware: this endpoint has been observed returning HTTP 204 for jobs whose
// operations do exist — see SearchOperationsHistory, which is the dependable way
// to read results back.
func (c *Client) GetJobOperationsAll(ctx context.Context, jobID string) iter.Seq2[Operation, error] {
	const size = MaxJobOperationsPageSize
	return walkPages(ctx, -1, size, func(ctx context.Context, page int) ([]Operation, *Page, error) {
		resp, err := c.GetJobOperationsPage(ctx, jobID, page, size)
		if err != nil {
			return nil, nil, err
		}
		return resp.Operations, resp.Page, nil
	})
}
