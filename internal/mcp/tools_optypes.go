package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerOpTypeTools(r *registrar) {
	r.tool(tsOptypes, optypesCatalogTool(), optypesCatalogHandler(r.p))
	r.tool(tsOptypes, optypesSearchTool(), optypesSearchHandler(r.p))
	r.tool(tsOptypes, optypesGetTool(), optypesGetHandler(r.p))
	r.tool(tsOptypesWrite, optypesCreateTool(), optypesCreateHandler(r.p))
	r.tool(tsOptypesWrite, optypesUpdateTool(), optypesUpdateHandler(r.p))
	r.tool(tsOptypesWrite, optypesDeleteTool(), optypesDeleteHandler(r.p))
}

func optypesCatalogTool() mcp.Tool {
	return mcp.NewTool("optypes_catalog",
		mcp.WithDescription("List the catalog of predefined OpenGate operation types (REBOOT_EQUIPMENT, FACTORY_RESET, ...). Use BEFORE jobs_create to discover which operation names exist and their parameters."),
	)
}

func optypesCatalogHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		data, err := c.OpTypesCatalog(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("catalog failed: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func optypesSearchTool() mcp.Tool {
	return mcp.NewTool("optypes_search",
		mcp.WithDescription("Search operation type definitions available to the logged-in user (catalog + organization-specific custom operations)."),
	)
}

func optypesSearchHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		data, err := c.SearchOpTypes(ctx, nil)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func optypesGetTool() mcp.Tool {
	return mcp.NewTool("optypes_get",
		mcp.WithDescription("Get an operation type definition (parameters schema, steps, applicable entity types) by name."),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Operation type name (e.g. \"REBOOT_EQUIPMENT\")"), mcp.Required()),
	)
}

func optypesGetHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		args := request.GetArguments()
		org, _ := args["organization"].(string)
		name, _ := args["name"].(string)
		if org == "" || name == "" {
			return mcp.NewToolResultError("organization and name are required"), nil
		}
		data, err := c.GetOpType(ctx, org, name)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("get failed: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func optypesCreateTool() mcp.Tool {
	return mcp.NewTool("optypes_create",
		mcp.WithDescription(`Create a NEW custom operation type in an organization (DDL — defines the operation; launch it later with jobs_create).

Body shape (ExtendedOperation):
  {"name":"CALIBRATE_SENSOR","title":"Calibrate Sensor",
   "description":"Applies a calibration offset to a datastream",
   "applicableTo":["entity.device"],
   "parameters":{"type":"object","properties":{"offset":{"type":"number"},"datastream":{"type":"string"}}},
   "steps":[{"name":"CALIBRATE","title":"Calibrate","description":"Apply offset"}]}`),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
		mcp.WithString("body", mcp.Description("Operation type JSON definition"), mcp.Required()),
	)
}

func optypesCreateHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		args := request.GetArguments()
		org, _ := args["organization"].(string)
		body, _ := args["body"].(string)
		if org == "" || body == "" {
			return mcp.NewToolResultError("organization and body are required"), nil
		}
		if _, err := c.CreateOpType(ctx, org, json.RawMessage(body)); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("create failed: %v", err)), nil
		}
		return mcp.NewToolResultText("Operation type created successfully."), nil
	}
}

func optypesUpdateTool() mcp.Tool {
	return mcp.NewTool("optypes_update",
		mcp.WithDescription("Update an existing custom operation type definition."),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Operation type name"), mcp.Required()),
		mcp.WithString("body", mcp.Description("Full operation type JSON definition"), mcp.Required()),
	)
}

func optypesUpdateHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		args := request.GetArguments()
		org, _ := args["organization"].(string)
		name, _ := args["name"].(string)
		body, _ := args["body"].(string)
		if org == "" || name == "" || body == "" {
			return mcp.NewToolResultError("organization, name, and body are required"), nil
		}
		if err := c.UpdateOpType(ctx, org, name, json.RawMessage(body)); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("update failed: %v", err)), nil
		}
		return mcp.NewToolResultText("Operation type updated successfully."), nil
	}
}

func optypesDeleteTool() mcp.Tool {
	return mcp.NewTool("optypes_delete",
		mcp.WithDescription("Delete a custom operation type definition."),
		mcp.WithString("organization", mcp.Description("Organization name"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Operation type name"), mcp.Required()),
	)
}

func optypesDeleteHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		args := request.GetArguments()
		org, _ := args["organization"].(string)
		name, _ := args["name"].(string)
		if org == "" || name == "" {
			return mcp.NewToolResultError("organization and name are required"), nil
		}
		if err := c.DeleteOpType(ctx, org, name); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("delete failed: %v", err)), nil
		}
		return mcp.NewToolResultText("Operation type deleted successfully."), nil
	}
}
