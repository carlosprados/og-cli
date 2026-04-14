package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/carlosprados/og-cli/internal/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerTools registers all MCP tools that mirror CLI commands.
// Each tool calls into internal/client for parity with the CLI.
func registerTools(s *server.MCPServer, host, token string) {
	s.AddTool(loginTool(), loginHandler(host))

	c := client.New(host, token)

	// Datamodels — full CRUD + search
	s.AddTool(datamodelsSearchTool(), datamodelsSearchHandler(c))
	s.AddTool(datamodelsGetTool(), datamodelsGetHandler(c))
	s.AddTool(datamodelsCreateTool(), datamodelsCreateHandler(c))
	s.AddTool(datamodelsUpdateTool(), datamodelsUpdateHandler(c))
	s.AddTool(datamodelsDeleteTool(), datamodelsDeleteHandler(c))

	// Devices — full CRUD + search
	registerDeviceTools(s, c)
}

// --- login ---

func loginTool() mcp.Tool {
	return mcp.NewTool("login",
		mcp.WithDescription("Authenticate against OpenGate with email/password and return a JWT token"),
		mcp.WithString("email",
			mcp.Description("OpenGate email"),
			mcp.Required(),
		),
		mcp.WithString("password",
			mcp.Description("OpenGate password"),
			mcp.Required(),
		),
		mcp.WithString("host",
			mcp.Description("OpenGate API host URL (optional, uses profile default if omitted)"),
		),
	)
}

func loginHandler(defaultHost string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()

		email, _ := args["email"].(string)
		password, _ := args["password"].(string)

		if email == "" || password == "" {
			return mcp.NewToolResultError("email and password are required"), nil
		}

		host := defaultHost
		if h, _ := args["host"].(string); h != "" {
			host = h
		}

		c := client.New(host, "")
		result, err := c.Login(email, password)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("login failed: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Login successful. JWT: %s, APIKey: %s, Domain: %s", result.JWT, result.APIKey, result.Domain)), nil
	}
}

// --- datamodels search ---

func datamodelsSearchTool() mcp.Tool {
	return mcp.NewTool("datamodels_search",
		mcp.WithDescription(`Search OpenGate data models. ALWAYS use the 'query' parameter for filtering — do NOT build JSON manually.

Common datamodel fields for filtering:
- datamodels.identifier — datamodel ID
- datamodels.name — datamodel name
- datamodels.organizationName — organization name
- datamodels.version — version string

Examples:
  query: "datamodels.identifier like weather"
  query: "datamodels.organizationName eq sensehat"`,
		),
		mcp.WithString("query",
			mcp.Description("Filter using: \"field op value\". Multiple conditions joined with AND. Operators: eq, neq, like, gt, lt, gte, lte. Example: \"datamodels.identifier like weather\". Omit to list all."),
		),
		mcp.WithString("select",
			mcp.Description("Comma-separated fields to return"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Max number of results"),
		),
		mcp.WithString("filter",
			mcp.Description("Advanced: raw OpenGate JSON filter. Only use for OR/nested queries that 'query' cannot express. Overrides 'query'."),
		),
	)
}

func datamodelsSearchHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()

		filter, err := mcpBuildFilter(args)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid query: %v", err)), nil
		}

		resp, err := c.SearchDatamodels(filter)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
		}

		result, err := json.Marshal(resp.Datamodels)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshaling result: %v", err)), nil
		}
		return mcp.NewToolResultText(string(result)), nil
	}
}

// --- datamodels get ---

func datamodelsGetTool() mcp.Tool {
	return mcp.NewTool("datamodels_get",
		mcp.WithDescription("Get a specific OpenGate data model by organization and identifier."),
		mcp.WithString("organization",
			mcp.Description("Organization name"),
			mcp.Required(),
		),
		mcp.WithString("id",
			mcp.Description("Datamodel identifier"),
			mcp.Required(),
		),
	)
}

func datamodelsGetHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		orgName, _ := args["organization"].(string)
		id, _ := args["id"].(string)

		if orgName == "" || id == "" {
			return mcp.NewToolResultError("organization and id are required"), nil
		}

		dm, err := c.GetDatamodel(orgName, id)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("get failed: %v", err)), nil
		}

		result, err := json.Marshal(dm)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshaling result: %v", err)), nil
		}
		return mcp.NewToolResultText(string(result)), nil
	}
}

// --- datamodels create ---

func datamodelsCreateTool() mcp.Tool {
	return mcp.NewTool("datamodels_create",
		mcp.WithDescription("Create a new OpenGate data model in an organization."),
		mcp.WithString("organization",
			mcp.Description("Organization name"),
			mcp.Required(),
		),
		mcp.WithString("body",
			mcp.Description("Full datamodel JSON definition"),
			mcp.Required(),
		),
	)
}

func datamodelsCreateHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		orgName, _ := args["organization"].(string)
		body, _ := args["body"].(string)

		if orgName == "" || body == "" {
			return mcp.NewToolResultError("organization and body are required"), nil
		}

		if err := c.CreateDatamodel(orgName, json.RawMessage(body)); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("create failed: %v", err)), nil
		}

		return mcp.NewToolResultText("Datamodel created successfully."), nil
	}
}

// --- datamodels update ---

func datamodelsUpdateTool() mcp.Tool {
	return mcp.NewTool("datamodels_update",
		mcp.WithDescription("Update an existing OpenGate data model."),
		mcp.WithString("organization",
			mcp.Description("Organization name"),
			mcp.Required(),
		),
		mcp.WithString("id",
			mcp.Description("Datamodel identifier"),
			mcp.Required(),
		),
		mcp.WithString("body",
			mcp.Description("Full datamodel JSON definition"),
			mcp.Required(),
		),
	)
}

func datamodelsUpdateHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		orgName, _ := args["organization"].(string)
		id, _ := args["id"].(string)
		body, _ := args["body"].(string)

		if orgName == "" || id == "" || body == "" {
			return mcp.NewToolResultError("organization, id, and body are required"), nil
		}

		if err := c.UpdateDatamodel(orgName, id, json.RawMessage(body)); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("update failed: %v", err)), nil
		}

		return mcp.NewToolResultText("Datamodel updated successfully."), nil
	}
}

// --- datamodels delete ---

func datamodelsDeleteTool() mcp.Tool {
	return mcp.NewTool("datamodels_delete",
		mcp.WithDescription("Delete an OpenGate data model."),
		mcp.WithString("organization",
			mcp.Description("Organization name"),
			mcp.Required(),
		),
		mcp.WithString("id",
			mcp.Description("Datamodel identifier"),
			mcp.Required(),
		),
	)
}

func datamodelsDeleteHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		orgName, _ := args["organization"].(string)
		id, _ := args["id"].(string)

		if orgName == "" || id == "" {
			return mcp.NewToolResultError("organization and id are required"), nil
		}

		if err := c.DeleteDatamodel(orgName, id); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("delete failed: %v", err)), nil
		}

		return mcp.NewToolResultText("Datamodel deleted successfully."), nil
	}
}
