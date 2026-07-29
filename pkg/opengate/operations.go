package opengate

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Operations API uses /v80/ prefix (not /north/v80/)
const (
	searchJobsPath    = "/north/{v}/search/jobs"
	searchTasksPath   = "/north/{v}/search/tasks"
	jobsPath          = "/north/{v}/operation/jobs"
	jobPath           = "/north/{v}/operation/jobs/%s"
	jobOperationsPath = "/north/{v}/operation/jobs/%s/operations"
	tasksPath         = "/north/{v}/operation/tasks"
	taskPath          = "/north/{v}/operation/tasks/%s"
	taskJobsPath      = "/north/{v}/operation/tasks/%s/jobs"
)

// Job represents an OpenGate operation job.
type Job struct {
	ID      string          `json:"id,omitempty"`
	TaskID  string          `json:"taskId,omitempty"`
	Request json.RawMessage `json:"request,omitempty"`
	Report  json.RawMessage `json:"report,omitempty"`
}

// Task represents an OpenGate operation task.
type Task struct {
	ID          string          `json:"id,omitempty"`
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	State       string          `json:"state,omitempty"`
	Domain      string          `json:"domain,omitempty"`
	Workgroup   string          `json:"workgroup,omitempty"`
	Schedule    json.RawMessage `json:"schedule,omitempty"`
	Job         json.RawMessage `json:"job,omitempty"`
}

// SearchJobsResponse is the response from the jobs search endpoint.
type SearchJobsResponse struct {
	Jobs []json.RawMessage `json:"jobs"`
	Page *Page             `json:"page,omitempty"`
}

// SearchTasksResponse is the response from the tasks search endpoint.
type SearchTasksResponse struct {
	Tasks []json.RawMessage `json:"tasks"`
	Page  *Page             `json:"page,omitempty"`
}

// JobOperationsResponse is the response listing operations within a job.
type JobOperationsResponse struct {
	Operations []Operation `json:"operations"`
	Page       *Page       `json:"page,omitempty"`
}

// SearchJobs searches for jobs.
func (c *Client) SearchJobs(ctx context.Context, filter json.RawMessage) (*SearchJobsResponse, error) {
	var body string
	if filter != nil {
		body = string(filter)
	} else {
		body = "{}"
	}

	data, statusCode, err := c.Post(ctx, searchJobsPath, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("search jobs: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	if IsEmptyResponse(data, statusCode) {
		return &SearchJobsResponse{}, nil
	}

	var resp SearchJobsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing jobs response: %w", err)
	}
	return &resp, nil
}

// SearchTasks searches for tasks.
func (c *Client) SearchTasks(ctx context.Context, filter json.RawMessage) (*SearchTasksResponse, error) {
	var body string
	if filter != nil {
		body = string(filter)
	} else {
		body = "{}"
	}

	data, statusCode, err := c.Post(ctx, searchTasksPath, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("search tasks: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	if IsEmptyResponse(data, statusCode) {
		return &SearchTasksResponse{}, nil
	}

	var resp SearchTasksResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing tasks response: %w", err)
	}
	return &resp, nil
}

// CreateJob creates a new operation job.
func (c *Client) CreateJob(ctx context.Context, body json.RawMessage) (json.RawMessage, error) {
	data, statusCode, err := c.Post(ctx, jobsPath, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("create job: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	return data, nil
}

// GetJob retrieves a job report.
func (c *Client) GetJob(ctx context.Context, jobID string) (json.RawMessage, error) {
	path := fmt.Sprintf(jobPath, jobID)

	data, statusCode, err := c.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("get job: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	return data, nil
}

// UpdateJob updates an existing job (add/remove targets, pause/resume, etc).
func (c *Client) UpdateJob(ctx context.Context, jobID string, body json.RawMessage) (json.RawMessage, error) {
	path := fmt.Sprintf(jobPath, jobID)

	data, statusCode, err := c.Put(ctx, path, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("update job: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	return data, nil
}

// CancelJob cancels (deletes) a job.
func (c *Client) CancelJob(ctx context.Context, jobID string) error {
	path := fmt.Sprintf(jobPath, jobID)

	data, statusCode, err := c.Delete(ctx, path)
	if err != nil {
		return fmt.Errorf("cancel job: %w", err)
	}
	return CheckResponse(data, statusCode)
}

// GetJobOperations lists operations within a job.
func (c *Client) GetJobOperations(ctx context.Context, jobID string) (*JobOperationsResponse, error) {
	return c.getJobOperations(ctx, fmt.Sprintf(jobOperationsPath, jobID))
}

// getJobOperations fetches and decodes a job operations listing from a fully
// built path, shared by the plain and the paged variants.
func (c *Client) getJobOperations(ctx context.Context, path string) (*JobOperationsResponse, error) {
	data, statusCode, err := c.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("get job operations: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	if IsEmptyResponse(data, statusCode) {
		return &JobOperationsResponse{}, nil
	}

	var resp JobOperationsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing job operations: %w", err)
	}
	return &resp, nil
}

// CreateTask creates a new operation task.
func (c *Client) CreateTask(ctx context.Context, body json.RawMessage) (json.RawMessage, error) {
	data, statusCode, err := c.Post(ctx, tasksPath, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	return data, nil
}

// GetTask retrieves a task.
func (c *Client) GetTask(ctx context.Context, taskID string) (json.RawMessage, error) {
	path := fmt.Sprintf(taskPath, taskID)

	data, statusCode, err := c.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	return data, nil
}

// CancelTask cancels (deletes) a task.
func (c *Client) CancelTask(ctx context.Context, taskID string) error {
	path := fmt.Sprintf(taskPath, taskID)

	data, statusCode, err := c.Delete(ctx, path)
	if err != nil {
		return fmt.Errorf("cancel task: %w", err)
	}
	return CheckResponse(data, statusCode)
}

// GetTaskJobs lists jobs within a task.
func (c *Client) GetTaskJobs(ctx context.Context, taskID string) (*SearchJobsResponse, error) {
	path := fmt.Sprintf(taskJobsPath, taskID)

	data, statusCode, err := c.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("get task jobs: %w", err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	if IsEmptyResponse(data, statusCode) {
		return &SearchJobsResponse{}, nil
	}

	var resp SearchJobsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing task jobs: %w", err)
	}
	return &resp, nil
}
