package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerOperationTools(r *registrar) {
	r.tool(tsJobs, jobsSearchTool(), jobsSearchHandler(r.p))
	r.tool(tsJobs, jobsGetTool(), jobsGetHandler(r.p))
	r.tool(tsJobsRun, jobsCreateTool(), jobsCreateHandler(r.p))
	r.tool(tsJobsRun, jobsCancelTool(), jobsCancelHandler(r.p))
	r.tool(tsJobs, jobsOpsTool(), jobsOpsHandler(r.p))
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
		resp, err := c.SearchJobs(filter)
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
		data, err := c.GetJob(id)
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
		data, err := c.CreateJob(json.RawMessage(body))
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
		if err := c.CancelJob(id); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("cancel failed: %v", err)), nil
		}
		return mcp.NewToolResultText("Job cancelled."), nil
	}
}

// --- jobs operations ---

func jobsOpsTool() mcp.Tool {
	return mcp.NewTool("jobs_operations",
		mcp.WithDescription("List individual operations within a job (one per target entity)."),
		mcp.WithString("id", mcp.Description("Job ID (UUID)"), mcp.Required()),
	)
}

func jobsOpsHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		id, _ := request.GetArguments()["id"].(string)
		if id == "" {
			return mcp.NewToolResultError("id is required"), nil
		}
		resp, err := c.GetJobOperations(id)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed: %v", err)), nil
		}
		result, _ := json.Marshal(resp.Operations)
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
		resp, err := c.SearchTasks(filter)
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
		data, err := c.GetTask(id)
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
		data, err := c.CreateTask(json.RawMessage(body))
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
		if err := c.CancelTask(id); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("cancel failed: %v", err)), nil
		}
		return mcp.NewToolResultText("Task cancelled."), nil
	}
}
