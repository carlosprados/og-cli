package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/carlosprados/og-cli/pkg/opengate"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerProvisionTools(s *server.MCPServer, c *opengate.Client) {
	s.AddTool(provisionListTool(), provisionListHandler(c))
	s.AddTool(provisionGetTool(), provisionGetHandler(c))
	s.AddTool(provisionCreateTool(), provisionCreateHandler(c))
	s.AddTool(provisionUpdateTool(), provisionUpdateHandler(c))
	s.AddTool(provisionDeleteTool(), provisionDeleteHandler(c))
	s.AddTool(provisionPlanTool(), provisionPlanHandler(c))
	s.AddTool(provisionBulkTool(), provisionBulkHandler(c))
	s.AddTool(provisionBulkStatusTool(), provisionBulkStatusHandler(c))
	s.AddTool(provisionBulkDetailsTool(), provisionBulkDetailsHandler(c))
}

const provisionScriptContract = `A provision function ("provision processor") is a JavaScript script that turns inbound rows (typically an Excel sheet) into ODM provisioning actions. It MUST implement two functions:
  normalizeRawObject(rawObject)     — validate + shape one inbound row into a normalized object
  actionsPlanning(normalizedObject) — return an array of actions (CREATE_DEVICE_ACTION, UPDATE_ASSET_ACTION, ...)
Helpers available in the runtime: printLog, getEntity, entitiesGenericSearch, checkAsset/checkDevice/checkSubscription/checkSubscriber, the Entity builder (new Entity().addDatastream(...)), and the *_ACTION builders. The script lives in scriptProcessor.script. Authoring tips from the platform: use single quotes for strings and block comments (/* */).`

func provisionStringArg(args map[string]any, key string) string {
	v, _ := args[key].(string)
	return v
}

// --- list ---

func provisionListTool() mcp.Tool {
	return mcp.NewTool("provision_list",
		mcp.WithDescription("List OpenGate provision functions (provision processors) in an organization. "+provisionScriptContract),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
	)
}

func provisionListHandler(c *opengate.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		org := provisionStringArg(args, "organization")
		if org == "" {
			return mcp.NewToolResultError("organization is required"), nil
		}
		items, err := c.ListProvisionProcessors(org)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("list failed: %v", err)), nil
		}
		result, err := json.Marshal(items)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshaling result: %v", err)), nil
		}
		return mcp.NewToolResultText(string(result)), nil
	}
}

// --- get ---

func provisionGetTool() mcp.Tool {
	return mcp.NewTool("provision_get",
		mcp.WithDescription("Get a provision function by identifier. The code is in scriptProcessor.script."),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
		mcp.WithString("id", mcp.Description("Provision processor identifier (provisionProcessorId, from provision_list)"), mcp.Required()),
	)
}

func provisionGetHandler(c *opengate.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		org := provisionStringArg(args, "organization")
		id := provisionStringArg(args, "id")
		if org == "" || id == "" {
			return mcp.NewToolResultError("organization and id are required"), nil
		}
		data, err := c.GetProvisionProcessor(org, id)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("get failed: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// --- create ---

func provisionCreateTool() mcp.Tool {
	return mcp.NewTool("provision_create",
		mcp.WithDescription(`Create a provision function. `+provisionScriptContract+`

Body shape:
  {"name":"createUpdate","configurationParams":{"spreadsheet":{"sheetName":"PARA","headerRow":2,"resultColumnName":"ODM_Result"}},
   "scriptProcessor":{"script":"function normalizeRawObject(o){...} function actionsPlanning(n){...}"}}
Note: name must match ^[a-zA-Z0-9]+$ (alphanumeric only).`),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
		mcp.WithString("body", mcp.Description("Provision function JSON definition"), mcp.Required()),
	)
}

func provisionCreateHandler(c *opengate.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		org := provisionStringArg(args, "organization")
		body := provisionStringArg(args, "body")
		if org == "" || body == "" {
			return mcp.NewToolResultError("organization and body are required"), nil
		}
		if _, err := c.CreateProvisionProcessor(org, json.RawMessage(body)); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("create failed: %v", err)), nil
		}
		return mcp.NewToolResultText("Provision function created successfully."), nil
	}
}

// --- update ---

func provisionUpdateTool() mcp.Tool {
	return mcp.NewTool("provision_update",
		mcp.WithDescription("Update an existing provision function. Fetch it first with provision_get, modify, and send the FULL body back."),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
		mcp.WithString("id", mcp.Description("Provision processor identifier"), mcp.Required()),
		mcp.WithString("body", mcp.Description("Full provision function JSON definition"), mcp.Required()),
	)
}

func provisionUpdateHandler(c *opengate.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		org := provisionStringArg(args, "organization")
		id := provisionStringArg(args, "id")
		body := provisionStringArg(args, "body")
		if org == "" || id == "" || body == "" {
			return mcp.NewToolResultError("organization, id, and body are required"), nil
		}
		if err := c.UpdateProvisionProcessor(org, id, json.RawMessage(body)); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("update failed: %v", err)), nil
		}
		return mcp.NewToolResultText("Provision function updated successfully."), nil
	}
}

// --- delete ---

func provisionDeleteTool() mcp.Tool {
	return mcp.NewTool("provision_delete",
		mcp.WithDescription("Delete a provision function by identifier."),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
		mcp.WithString("id", mcp.Description("Provision processor identifier"), mcp.Required()),
	)
}

func provisionDeleteHandler(c *opengate.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		org := provisionStringArg(args, "organization")
		id := provisionStringArg(args, "id")
		if org == "" || id == "" {
			return mcp.NewToolResultError("organization and id are required"), nil
		}
		if err := c.DeleteProvisionProcessor(org, id); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("delete failed: %v", err)), nil
		}
		return mcp.NewToolResultText("Provision function deleted successfully."), nil
	}
}

// --- plan ---

func provisionPlanTool() mcp.Tool {
	return mcp.NewTool("provision_plan",
		mcp.WithDescription("Dry-run a provision function against the first N rows of a local Excel file and return the computed action plan as JSON. NO data is mutated — use this to iterate on a script before running it for real."),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
		mcp.WithString("id", mcp.Description("Provision processor identifier"), mcp.Required()),
		mcp.WithString("file", mcp.Description("Local path to the Excel file (.xlsx/.xls) with rows to plan"), mcp.Required()),
		mcp.WithNumber("rows", mcp.Description("Number of entries (rows) to plan (default: 1)")),
	)
}

func provisionPlanHandler(c *opengate.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		org := provisionStringArg(args, "organization")
		id := provisionStringArg(args, "id")
		file := provisionStringArg(args, "file")
		if org == "" || id == "" || file == "" {
			return mcp.NewToolResultError("organization, id, and file are required"), nil
		}
		if _, err := os.Stat(file); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("file not accessible: %v", err)), nil
		}
		rows := 1
		if v, ok := args["rows"].(float64); ok && v > 0 {
			rows = int(v)
		}
		data, err := c.PlanProvisionBulk(org, id, file, rows)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("plan failed: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// --- bulk ---

func provisionBulkTool() mcp.Tool {
	return mcp.NewTool("provision_bulk",
		mcp.WithDescription("Execute a full provisioning bulk from a local Excel file. This MUTATES data (creates/updates/deletes entities). Returns the bulk process id — track it with provision_bulk_status."),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
		mcp.WithString("id", mcp.Description("Provision processor identifier"), mcp.Required()),
		mcp.WithString("file", mcp.Description("Local path to the Excel file (.xlsx/.xls) with rows to provision"), mcp.Required()),
	)
}

func provisionBulkHandler(c *opengate.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		org := provisionStringArg(args, "organization")
		id := provisionStringArg(args, "id")
		file := provisionStringArg(args, "file")
		if org == "" || id == "" || file == "" {
			return mcp.NewToolResultError("organization, id, and file are required"), nil
		}
		if _, err := os.Stat(file); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("file not accessible: %v", err)), nil
		}
		bulkID, err := c.RunProvisionBulk(org, id, file)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("bulk failed: %v", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Bulk process started: %s", bulkID)), nil
	}
}

// --- bulk status ---

func provisionBulkStatusTool() mcp.Tool {
	return mcp.NewTool("provision_bulk_status",
		mcp.WithDescription("Read the status summary of a bulk process (status, processed/successful/error counts)."),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
		mcp.WithString("bulk_id", mcp.Description("Bulk process id (from provision_bulk)"), mcp.Required()),
	)
}

func provisionBulkStatusHandler(c *opengate.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		org := provisionStringArg(args, "organization")
		bulkID := provisionStringArg(args, "bulk_id")
		if org == "" || bulkID == "" {
			return mcp.NewToolResultError("organization and bulk_id are required"), nil
		}
		data, err := c.GetProvisionBulkStatus(org, bulkID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("bulk status failed: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// --- bulk details ---

func provisionBulkDetailsTool() mcp.Tool {
	return mcp.NewTool("provision_bulk_details",
		mcp.WithDescription("Download the result Excel of a finished bulk process to a local path. Reports if the process is not finished yet."),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
		mcp.WithString("bulk_id", mcp.Description("Bulk process id"), mcp.Required()),
		mcp.WithString("out", mcp.Description("Local path to write the result Excel (default: <bulk-id>.xlsx)")),
	)
}

func provisionBulkDetailsHandler(c *opengate.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		org := provisionStringArg(args, "organization")
		bulkID := provisionStringArg(args, "bulk_id")
		if org == "" || bulkID == "" {
			return mcp.NewToolResultError("organization and bulk_id are required"), nil
		}
		data, ready, err := c.GetProvisionBulkDetails(org, bulkID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("bulk details failed: %v", err)), nil
		}
		if !ready {
			return mcp.NewToolResultText("Bulk process not finished yet — no details available."), nil
		}
		out := provisionStringArg(args, "out")
		if out == "" {
			out = bulkID + ".xlsx"
		}
		if err := os.WriteFile(out, data, 0o644); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("writing %s: %v", out, err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Bulk details written to %s", out)), nil
	}
}
