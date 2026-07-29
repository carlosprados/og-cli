package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/carlosprados/og-cli/pkg/opengate"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerOperationTools(r *registrar) {
	r.tool(tsJobs, jobsSearchTool(), jobsSearchHandler(r.p))
	r.tool(tsJobs, jobsGetTool(), jobsGetHandler(r.p))
	r.tool(tsJobsRun, jobsCreateTool(), jobsCreateHandler(r.p))
	r.tool(tsJobsRun, jobsCancelTool(), jobsCancelHandler(r.p))
	r.tool(tsJobs, jobsOpsTool(), jobsOpsHandler(r.p))
	r.tool(tsJobs, operationsHistoryTool(), operationsHistoryHandler(r.p))
	r.tool(tsTasks, tasksSearchTool(), tasksSearchHandler(r.p))
	r.tool(tsTasks, tasksGetTool(), tasksGetHandler(r.p))
	r.tool(tsTasksWrite, tasksCreateTool(), tasksCreateHandler(r.p))
	r.tool(tsTasksWrite, tasksCancelTool(), tasksCancelHandler(r.p))
}

// --- jobs search ---

func jobsSearchTool() mcp.Tool {
	return mcp.NewTool("jobs_search",
		mcp.WithDescription(`Search OpenGate operation jobs. Use 'query' to filter.

Common fields: jobs.request.name, jobs.report.summary.status (IN_PROGRESS, FINISHED, CANCELLED, PAUSED)`),
		mcp.WithString("query", mcp.Description("Filter: \"field op value\". Omit to list all.")),
		mcp.WithNumber("limit", mcp.Description("Max results")),
		mcp.WithString("filter", mcp.Description("Advanced: raw JSON filter.")),
	)
}

func jobsSearchHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		filter, err := mcpBuildFilter(request.GetArguments())
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid query: %v", err)), nil
		}
		resp, err := c.SearchJobs(ctx, filter)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
		}
		result, _ := json.Marshal(resp.Jobs)
		return mcp.NewToolResultText(string(result)), nil
	}
}

// --- jobs get ---

func jobsGetTool() mcp.Tool {
	return mcp.NewTool("jobs_get",
		mcp.WithDescription("Get a job report including execution summary and target status."),
		mcp.WithString("id", mcp.Description("Job ID (UUID)"), mcp.Required()),
	)
}

func jobsGetHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		id, _ := request.GetArguments()["id"].(string)
		if id == "" {
			return mcp.NewToolResultError("id is required"), nil
		}
		data, err := c.GetJob(ctx, id)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("get failed: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// --- jobs create ---

func jobsCreateTool() mcp.Tool {
	return mcp.NewTool("jobs_create",
		mcp.WithDescription(`Create an operation job. The body must follow the OpenGate job format.

Example for REFRESH_INFO on a device:
{"job":{"request":{"name":"REFRESH_INFO","parameters":{},"active":true,"schedule":{"stop":{"delayed":90000}},"operationParameters":{"timeout":85000,"retries":0},"target":{"append":{"entities":["device_id"]}}}}}`),
		mcp.WithString("body", mcp.Description("Full job JSON definition"), mcp.Required()),
	)
}

func jobsCreateHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		body, _ := request.GetArguments()["body"].(string)
		if body == "" {
			return mcp.NewToolResultError("body is required"), nil
		}
		data, err := c.CreateJob(ctx, json.RawMessage(body))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("create failed: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// --- jobs cancel ---

func jobsCancelTool() mcp.Tool {
	return mcp.NewTool("jobs_cancel",
		mcp.WithDescription("Cancel a running job."),
		mcp.WithString("id", mcp.Description("Job ID (UUID)"), mcp.Required()),
	)
}

func jobsCancelHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		id, _ := request.GetArguments()["id"].(string)
		if id == "" {
			return mcp.NewToolResultError("id is required"), nil
		}
		if err := c.CancelJob(ctx, id); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("cancel failed: %v", err)), nil
		}
		return mcp.NewToolResultText("Job cancelled."), nil
	}
}

// --- jobs operations ---

func jobsOpsTool() mcp.Tool {
	return mcp.NewTool("jobs_operations",
		mcp.WithDescription(`List individual operations within a job (one per target entity), each with its execution steps.

Every operation carries status (lifecycle, e.g. FINISHED) and result (outcome, e.g. SUCCESSFUL), plus steps[] where each step has name, result (SUCCESSFUL, ERROR, SKIPPED, NOT_EXECUTED) and description — that is where "why did this device fail" is answered.

Results are paged: the response includes page.number and page.of. For a job over a large fleet, read successive pages instead of assuming the first one is the whole job. To filter operations across jobs (by entity, name or result) use operations_history instead.`),
		mcp.WithString("id", mcp.Description("Job ID (UUID)"), mcp.Required()),
		mcp.WithNumber("limit", mcp.Description("Page size (maximum 1000 on this endpoint).")),
		mcp.WithNumber("page", mcp.Description("Page number to fetch. Omit for the platform's first page; check page.of in the response for how many there are.")),
	)
}

func jobsOpsHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		args := request.GetArguments()
		id, _ := args["id"].(string)
		if id == "" {
			return mcp.NewToolResultError("id is required"), nil
		}

		page := -1 // omit start: let the platform serve its first page
		if v, ok := args["page"].(float64); ok && v > 0 {
			page = int(v)
		}
		size := 0
		if v, ok := args["limit"].(float64); ok && v > 0 {
			size = int(v)
		}

		resp, err := c.GetJobOperationsPage(ctx, id, page, size)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed: %v", err)), nil
		}
		// Include the page block: without it the caller cannot tell a complete
		// answer from the first page of a larger one.
		result, _ := json.Marshal(resp)
		return mcp.NewToolResultText(string(result)), nil
	}
}

// --- operations history ---

func operationsHistoryTool() mcp.Tool {
	return mcp.NewTool("operations_history",
		mcp.WithDescription(`Search closed operations across jobs, with their per-step results.

Use this to answer "what happened to these operations": unlike jobs_operations, which lists a single job by id, this takes a filter, so it can select by entity, operation name or result across jobs.

Each operation carries entityId, name, status, result, execution.startedDate and steps[] (name, result, description). Step results are SUCCESSFUL, ERROR, SKIPPED and NOT_EXECUTED.

Filter field names do NOT match the response field names. Confirmed set: identifiers are unprefixed (jobId, entityId, operationId, resourceType); everything else takes an "operation" prefix — operationName (or operation.name), operationStatus, operationResult (or operation.result), operationDate, operationNotify. Anything else returns HTTP 400 "Field in filter unknown", including the bare name/status/result/user/date/description, any "operations." prefix, and operation.status.

To answer "which devices failed", filter server-side with operationResult eq ERROR instead of fetching everything.

Results are paged; the response includes page.number. Note page.of is NOT always present, so do not rely on it to decide whether more pages exist — a page shorter than 'limit' is the end.

Examples:
  job: "<job-id>"                          # read back one job's results
  query: "operationResult eq ERROR"                  # only the failures
  query: "operationName eq DIAGNOSIS AND operationStatus eq FINISHED"
  query: "entityId eq dev-1"`),
		mcp.WithString("job", mcp.Description("Shortcut: read back the operations of this job id. Do not combine with 'query' or 'filter'.")),
		mcp.WithString("query", mcp.Description("Filter using: \"field op value\", conditions joined with AND. Omit to search all.")),
		mcp.WithNumber("limit", mcp.Description("Page size (maximum 2000).")),
		mcp.WithNumber("page", mcp.Description("Page number to fetch, counting from 1.")),
		mcp.WithString("filter", mcp.Description("Advanced: raw OpenGate JSON filter. Overrides 'query'.")),
	)
}

func operationsHistoryHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		args := request.GetArguments()

		var filter json.RawMessage
		if job, _ := args["job"].(string); job != "" {
			if q, _ := args["query"].(string); q != "" {
				return mcp.NewToolResultError("use either 'job' or 'query', not both"), nil
			}
			filter = opengate.JobIDFilter(job)
			if size, ok := args["limit"].(float64); ok && size > 0 {
				page := 1
				if v, ok := args["page"].(float64); ok && v > 0 {
					page = int(v)
				}
				paged, err := opengate.SetFilterPage(filter, page, int(size))
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("invalid filter: %v", err)), nil
				}
				filter = paged
			}
		} else {
			var err error
			filter, err = mcpBuildFilter(args)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid query: %v", err)), nil
			}
		}

		resp, err := c.SearchOperationsHistory(ctx, filter)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed: %v", err)), nil
		}
		result, _ := json.Marshal(resp)
		return mcp.NewToolResultText(string(result)), nil
	}
}

// --- tasks search ---

func tasksSearchTool() mcp.Tool {
	return mcp.NewTool("tasks_search",
		mcp.WithDescription(`Search OpenGate operation tasks (recurring/scheduled operations).

Common fields: tasks.name, tasks.state (ACTIVE, PAUSED, FINISHED)`),
		mcp.WithString("query", mcp.Description("Filter: \"field op value\". Omit to list all.")),
		mcp.WithNumber("limit", mcp.Description("Max results")),
		mcp.WithString("filter", mcp.Description("Advanced: raw JSON filter.")),
	)
}

func tasksSearchHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		filter, err := mcpBuildFilter(request.GetArguments())
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid query: %v", err)), nil
		}
		resp, err := c.SearchTasks(ctx, filter)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
		}
		result, _ := json.Marshal(resp.Tasks)
		return mcp.NewToolResultText(string(result)), nil
	}
}

// --- tasks get ---

func tasksGetTool() mcp.Tool {
	return mcp.NewTool("tasks_get",
		mcp.WithDescription("Get a task definition and status."),
		mcp.WithString("id", mcp.Description("Task ID (UUID)"), mcp.Required()),
	)
}

func tasksGetHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		id, _ := request.GetArguments()["id"].(string)
		if id == "" {
			return mcp.NewToolResultError("id is required"), nil
		}
		data, err := c.GetTask(ctx, id)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("get failed: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// --- tasks create ---

func tasksCreateTool() mcp.Tool {
	return mcp.NewTool("tasks_create",
		mcp.WithDescription("Create a new operation task (scheduled/recurring operations)."),
		mcp.WithString("body", mcp.Description("Full task JSON definition"), mcp.Required()),
	)
}

func tasksCreateHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		body, _ := request.GetArguments()["body"].(string)
		if body == "" {
			return mcp.NewToolResultError("body is required"), nil
		}
		data, err := c.CreateTask(ctx, json.RawMessage(body))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("create failed: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// --- tasks cancel ---

func tasksCancelTool() mcp.Tool {
	return mcp.NewTool("tasks_cancel",
		mcp.WithDescription("Cancel a task."),
		mcp.WithString("id", mcp.Description("Task ID (UUID)"), mcp.Required()),
	)
}

func tasksCancelHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		id, _ := request.GetArguments()["id"].(string)
		if id == "" {
			return mcp.NewToolResultError("id is required"), nil
		}
		if err := c.CancelTask(ctx, id); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("cancel failed: %v", err)), nil
		}
		return mcp.NewToolResultText("Task cancelled."), nil
	}
}
