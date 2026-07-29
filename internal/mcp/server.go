package mcp

import (
	"context"
	"github.com/carlosprados/og-cli/v2/pkg/opengate"
	"net/http"

	"github.com/mark3labs/mcp-go/server"
)

const (
	serverName    = "og-mcp"
	serverVersion = "0.1.0"
)

// Options configures the MCP server surface and credentials.
type Options struct {
	Host     string
	Token    string
	WebToken string
	APIKey   string

	// MultiTenant takes per-request credentials from HTTP headers instead of the
	// startup profile and drops the login tool (it would leak a JWT to the LLM).
	MultiTenant bool

	// Lean exposes the exec-over-CLI tools (og_exec/og_help) instead of the named
	// per-operation tools — the smallest token footprint. When true, Toolsets is
	// ignored. See lean.go.
	Lean bool

	// Toolsets is the resolved active set (from resolveToolsets); it alone decides
	// which named tools/prompts/resources are exposed. Ignored when Lean is true.
	Toolsets map[toolset]bool

	// ClientOptions configures every client the server builds — API version and
	// retry policy — so an MCP session behaves like the CLI it was launched from.
	ClientOptions []opengate.Option
}

// newServer creates a configured MCP server. The exposed surface is governed by
// opts.Lean / opts.Toolsets so a client only pays the context-token cost of the
// tools it asked for.
func newServer(opts Options) *server.MCPServer {
	s := server.NewMCPServer(
		serverName,
		serverVersion,
		server.WithToolCapabilities(true),
		server.WithPromptCapabilities(true),
		server.WithResourceCapabilities(true, false),
	)

	p := newProvider(opts.Host, credentials{token: opts.Token, webToken: opts.WebToken, apiKey: opts.APIKey}, opts.MultiTenant, opts.ClientOptions...)

	// Lean mode: a tiny exec-over-CLI surface. The guidance prompt still helps.
	if opts.Lean {
		r := &registrar{s: s, p: p, host: opts.Host, active: map[toolset]bool{tsMeta: true}}
		registerLeanTools(r)
		registerPrompts(r)
		return s
	}

	r := &registrar{s: s, p: p, host: opts.Host, active: opts.Toolsets}

	// login returns a JWT in its result text; never expose it in multi-tenant mode.
	if !opts.MultiTenant {
		registerLoginTool(r)
	}
	registerToolsetsMeta(r)
	registerDatamodelTools(r)
	registerDeviceTools(r)
	registerAlarmTools(r)
	registerTimeSeriesTools(r)
	registerDatasetTools(r)
	registerOperationTools(r)
	registerRuleTools(r)
	registerConnectorTools(r)
	registerProvisionTools(r)
	registerOpTypeTools(r)
	registerWorkspaceTools(r)
	registerDashboardTools(r)
	registerShareTools(r)
	registerIoTTools(r)
	registerPrompts(r)
	registerResources(r)

	return s
}

// ServeStdio starts the MCP server over stdio (always single-tenant).
func ServeStdio(opts Options) error {
	opts.MultiTenant = false
	s := newServer(opts)
	return server.ServeStdio(s)
}

// ServeHTTP starts the MCP server over HTTP (Streamable HTTP transport). When
// opts.MultiTenant is true, credentials are read per request from HTTP headers.
func ServeHTTP(addr string, opts Options) error {
	s := newServer(opts)

	httpOpts := []server.StreamableHTTPOption{}
	if opts.MultiTenant {
		httpOpts = append(httpOpts, server.WithHTTPContextFunc(func(ctx context.Context, req *http.Request) context.Context {
			return withCredentials(ctx, credentials{
				token:    bearerToken(req.Header.Get("Authorization")),
				webToken: req.Header.Get("X-OG-Web-Token"),
				apiKey:   req.Header.Get("X-OG-Api-Key"),
			})
		}))
	}

	httpServer := server.NewStreamableHTTPServer(s, httpOpts...)
	return httpServer.Start(addr)
}
