package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/carlosprados/og-cli/internal/query"
	"github.com/carlosprados/og-cli/internal/views"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerDeviceTools(r *registrar) {
	r.tool(tsDevices, devicesSearchTool(), devicesSearchHandler(r.p))
	r.tool(tsDevices, devicesGetTool(), devicesGetHandler(r.p))
	r.tool(tsDevicesWrite, devicesCreateTool(), devicesCreateHandler(r.p))
	r.tool(tsDevicesWrite, devicesUpdateTool(), devicesUpdateHandler(r.p))
	r.tool(tsDevicesWrite, devicesDeleteTool(), devicesDeleteHandler(r.p))
}

// --- search ---

func devicesSearchTool() mcp.Tool {
	return mcp.NewTool("devices_search",
		mcp.WithDescription(`Search OpenGate devices. ALWAYS use the 'query' parameter for filtering — do NOT build JSON manually.

EFFICIENCY: always project fields with 'view' and/or 'select' — without them every result
is the full device document (thousands of tokens each). For common field sets use 'view'
(e.g. view: "summary" for listings, "summary,power" for battery questions) instead of
guessing datastream paths.

IMPORTANT: devices_search filters on BOTH provisioned metadata AND the latest collected datastream values
(the server stores the _current value of every datastream on the device document, so filters like
"wt gt 20" or "device.temperature.value lte 50" are fully supported by POST /north/v80/search/devices).
Do NOT redirect the user to timeseries_data or datasets_data just because they filter by a datastream
value — only use those tools when the user explicitly asks for historical/time-windowed data.

FILTER BY COLLECTION TIME: each datastream's current value also carries timestamps, filterable via the
path "<datastream>._current.at" (platform reception) and "<datastream>._current.date" (measurement time),
value = ISO-8601 with offset (e.g. "2026-06-21T18:00:00.000+02:00"). The path WITHOUT "._current" is invalid.
  query: "device.identifier._current.at gte 2026-06-21T18:00:00.000+02:00"   # devices that reported after T
For "which devices collected anything after T" use the always-collected device.identifier; for a specific
measurement, filter that stream's "._current.date".

TYPE-STRICT: the search lake does not coerce types — a string-typed field only matches a JSON string, a
number-typed field a JSON number. Types come from the datamodel (read opengate://organizations/{org}/datamodel-fields
or use datamodels_get → schema.type/$ref), NOT the value's shape. provision.device.identifier is a STRING
(ogIdentifier), so an IP-/MAC-looking id is a string. For an all-digit value that is actually a string id,
single-quote it to force string typing: query: "provision.device.identifier eq '00123'".

Provision (metadata) fields:
- provision.device.identifier — device ID
- provision.device.name — device name
- provision.device.administrativeState — ACTIVE, TESTING, BANNED, etc.
- provision.device.operationalStatus — NORMAL, ALARM, etc.
- provision.administration.organization — organization name

Collected datastream fields (current value on the device):
- Default datamodel streams, e.g. device.temperature.value, device.cpu.total, device.ram.total, anin1
- Organization-specific streams defined in custom datamodels, e.g. wt, wp, batteryPercentage
- To discover which datastreams are available in an org, read the resource
  opengate://organizations/{org}/datamodel-fields

Examples:
  query: "provision.device.administrativeState eq ACTIVE"
  query: "provision.device.identifier like sense AND provision.device.administrativeState eq ACTIVE"
  query: "provision.administration.organization eq sensehat"
  query: "wt gt 20"                                              # devices with temperature > 20
  query: "wt gte 10 AND wt lte 30"                               # temperature between 10 and 30
  query: "device.temperature.value gt 50 AND provision.device.operationalStatus eq NORMAL"
  query: "anin1 gt 5 AND provision.administration.organization eq sensehat"`),
		mcp.WithString("query",
			mcp.Description("Filter using: \"field op value\". Multiple conditions joined with AND. Operators: eq, neq, like, gt, lt, gte, lte, in, exists. Example: \"provision.device.administrativeState eq ACTIVE\". Omit to list all devices."),
		),
		mcp.WithString("select",
			mcp.Description("Comma-separated fields to return. Append @at (reception time), @date (measurement time) and/or @source to a field for its current-value sub-fields; repeatable, e.g. \"wt@at@date\". Example: \"provision.device.identifier,provision.device.administrativeState,wt@at\""),
		),
		mcp.WithString("view",
			mcp.Description("Comma-separated named views that expand into field sets — USE THIS instead of guessing field paths. Built-in views: summary (id, type, name, org, status), identifier, name, type, location, organization, topology, status, hardware, software, relations, temperature, power, resources. Collected fields include their at timestamp automatically. Combinable with 'select' (explicit fields win). Read the opengate://views resource for the full dictionary. Example: \"summary,power\""),
		),
		mcp.WithNumber("limit",
			mcp.Description("Max number of results"),
		),
		mcp.WithString("filter",
			mcp.Description("Advanced: raw OpenGate JSON filter. Only use for OR/nested queries that 'query' cannot express. Overrides 'query'."),
		),
	)
}

func devicesSearchHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		args := request.GetArguments()

		filter, err := mcpBuildFilter(args)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid query: %v", err)), nil
		}

		resp, err := c.SearchDevices(filter)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
		}

		result, err := json.Marshal(resp.Devices)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshaling result: %v", err)), nil
		}
		return mcp.NewToolResultText(string(result)), nil
	}
}

// mcpBuildFilter is shared by all MCP search handlers.
func mcpBuildFilter(args map[string]any) (json.RawMessage, error) {
	// Raw filter takes precedence
	if f, _ := args["filter"].(string); f != "" {
		return json.RawMessage(f), nil
	}

	var conditions []query.Condition
	if q, _ := args["query"].(string); q != "" {
		var err error
		conditions, err = query.ParseQuery(q)
		if err != nil {
			return nil, err
		}
	}

	var limit int
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	var selectFields []string
	if s, _ := args["select"].(string); s != "" {
		for _, f := range strings.Split(s, ",") {
			if f = strings.TrimSpace(f); f != "" {
				selectFields = append(selectFields, f)
			}
		}
	}
	selectClauses := query.SelectFromFields(selectFields, false)

	if v, _ := args["view"].(string); v != "" {
		var viewNames []string
		for _, name := range strings.Split(v, ",") {
			if name = strings.TrimSpace(name); name != "" {
				viewNames = append(viewNames, name)
			}
		}
		reg, err := views.Load()
		if err != nil {
			return nil, err
		}
		selectClauses, err = reg.ResolveSelect(viewNames, selectClauses)
		if err != nil {
			return nil, err
		}
	}

	p := query.SearchParams{
		Conditions: conditions,
		Limit:      limit,
		Select:     selectClauses,
	}

	if len(p.Conditions) == 0 && p.Limit == 0 && len(p.Select) == 0 {
		return nil, nil
	}

	return query.BuildFilter(p)
}

// --- get ---

func devicesGetTool() mcp.Tool {
	return mcp.NewTool("devices_get",
		mcp.WithDescription("Get full detail of a specific OpenGate device. Returns the device in flattened JSON format with all provisioned and collected datastreams."),
		mcp.WithString("organization",
			mcp.Description("Organization name (e.g. \"sensehat\")"),
			mcp.Required(),
		),
		mcp.WithString("id",
			mcp.Description("Device identifier (e.g. \"sense-001\")"),
			mcp.Required(),
		),
	)
}

func devicesGetHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		args := request.GetArguments()
		orgName, _ := args["organization"].(string)
		id, _ := args["id"].(string)

		if orgName == "" || id == "" {
			return mcp.NewToolResultError("organization and id are required"), nil
		}

		data, err := c.GetDevice(orgName, id)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("get failed: %v", err)), nil
		}

		return mcp.NewToolResultText(string(data)), nil
	}
}

// --- create ---

func devicesCreateTool() mcp.Tool {
	return mcp.NewTool("devices_create",
		mcp.WithDescription("Create a new OpenGate device in an organization."),
		mcp.WithString("organization",
			mcp.Description("Organization name"),
			mcp.Required(),
		),
		mcp.WithString("body",
			mcp.Description("Full device JSON definition (flattened format)"),
			mcp.Required(),
		),
	)
}

func devicesCreateHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		args := request.GetArguments()
		orgName, _ := args["organization"].(string)
		body, _ := args["body"].(string)

		if orgName == "" || body == "" {
			return mcp.NewToolResultError("organization and body are required"), nil
		}

		if err := c.CreateDevice(orgName, json.RawMessage(body)); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("create failed: %v", err)), nil
		}

		return mcp.NewToolResultText("Device created successfully."), nil
	}
}

// --- update ---

func devicesUpdateTool() mcp.Tool {
	return mcp.NewTool("devices_update",
		mcp.WithDescription("Update an existing OpenGate device."),
		mcp.WithString("organization",
			mcp.Description("Organization name"),
			mcp.Required(),
		),
		mcp.WithString("id",
			mcp.Description("Device identifier"),
			mcp.Required(),
		),
		mcp.WithString("body",
			mcp.Description("Full device JSON definition (flattened format)"),
			mcp.Required(),
		),
	)
}

func devicesUpdateHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		args := request.GetArguments()
		orgName, _ := args["organization"].(string)
		id, _ := args["id"].(string)
		body, _ := args["body"].(string)

		if orgName == "" || id == "" || body == "" {
			return mcp.NewToolResultError("organization, id, and body are required"), nil
		}

		if err := c.UpdateDevice(orgName, id, json.RawMessage(body)); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("update failed: %v", err)), nil
		}

		return mcp.NewToolResultText("Device updated successfully."), nil
	}
}

// --- delete ---

func devicesDeleteTool() mcp.Tool {
	return mcp.NewTool("devices_delete",
		mcp.WithDescription("Delete an OpenGate device."),
		mcp.WithString("organization",
			mcp.Description("Organization name"),
			mcp.Required(),
		),
		mcp.WithString("id",
			mcp.Description("Device identifier"),
			mcp.Required(),
		),
	)
}

func devicesDeleteHandler(p *provider) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, errRes := p.client(ctx)
		if errRes != nil {
			return errRes, nil
		}
		args := request.GetArguments()
		orgName, _ := args["organization"].(string)
		id, _ := args["id"].(string)

		if orgName == "" || id == "" {
			return mcp.NewToolResultError("organization and id are required"), nil
		}

		if err := c.DeleteDevice(orgName, id); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("delete failed: %v", err)), nil
		}

		return mcp.NewToolResultText("Device deleted successfully."), nil
	}
}
