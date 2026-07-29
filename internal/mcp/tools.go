package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/carlosprados/og-cli/pkg/opengate"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerLoginTool registers the login tool (single-tenant only — it returns a
// JWT in its result text, which must never reach the LLM in multi-tenant mode).
func registerLoginTool(r *registrar) {
	r.tool(tsLogin, loginTool(), loginHandler(r.host))
}

// registerDatamodelTools registers datamodel tools (parity with the CLI).
// Device tools are registered separately (registerDeviceTools) by newServer.
func registerDatamodelTools(r *registrar) {
	r.tool(tsDatamodels, datamodelsSearchTool(), datamodelsSearchHandler(r.p))
	r.tool(tsDatamodels, datamodelsGetTool(), datamodelsGetHandler(r.p))
	r.tool(tsDatamodelsWrite, datamodelsCreateTool(), datamodelsCreateHandler(r.p))
	r.tool(tsDatamodelsWrite, datamodelsUpdateTool(), datamodelsUpdateHandler(r.p))
	r.tool(tsDatamodelsWrite, datamodelsDeleteTool(), datamodelsDeleteHandler(r.p))
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
		mcp.WithString("2FaCode",
			mcp.Description("6-digit TOTP code, required only for accounts with 2FA enabled. If login reports 2FA is required, re-call this tool with a fresh code."),
		),
		mcp.WithString("2FaSecret",
			mcp.Description("base32 TOTP secret; when provided, the 6-digit code is derived server-side (alternative to 2FaCode, for unattended logins)."),
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
		twoFaCode, _ := args["2FaCode"].(string)
		twoFaSecret, _ := args["2FaSecret"].(string)

		if email == "" || password == "" {
			return mcp.NewToolResultError("email and password are required"), nil
		}

		if twoFaCode == "" && twoFaSecret != "" {
			code, gerr := opengate.GenerateTOTPCode(twoFaSecret)
			if gerr != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid 2FA secret: %v", gerr)), nil
			}
			twoFaCode = code
		}

		host := defaultHost
		if h, _ := args["host"].(string); h != "" {
			host = h
		}

		c := opengate.New(host, "")
		result, err := c.Login(ctx, email, password, twoFaCode)
		if err != nil {
			if opengate.Is2FAChallenge(err) {
				return mcp.NewToolResultError("this account has 2FA enabled — re-call the login tool with a fresh 6-digit '2FaCode' from the authenticator app"), nil
			}
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

func datamodelsSearchHandler(p *provider) server.ToolHandlerFunc {
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

		resp, err := c.SearchDatamodels(ctx, filter)
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

func datamodelsGetHandler(p *provider) server.ToolHandlerFunc {
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

		dm, err := c.GetDatamodel(ctx, orgName, id)
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

func datamodelsCreateHandler(p *provider) server.ToolHandlerFunc {
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

		if err := c.CreateDatamodel(ctx, orgName, json.RawMessage(body)); err != nil {
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

func datamodelsUpdateHandler(p *provider) server.ToolHandlerFunc {
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

		if err := c.UpdateDatamodel(ctx, orgName, id, json.RawMessage(body)); err != nil {
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

func datamodelsDeleteHandler(p *provider) server.ToolHandlerFunc {
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

		if err := c.DeleteDatamodel(ctx, orgName, id); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("delete failed: %v", err)), nil
		}

		return mcp.NewToolResultText("Datamodel deleted successfully."), nil
	}
}
