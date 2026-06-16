package mcp

import (
	"context"
	"net/http"

	"github.com/mark3labs/mcp-go/server"
)

const (
	serverName    = "og-mcp"
	serverVersion = "0.1.0"
)

// newServer creates a configured MCP server with all tools, prompts, and resources.
// In multi-tenant mode, per-request credentials come from HTTP headers and the
// login tool is dropped (it would return a JWT into the LLM).
func newServer(host, token, webToken, apiKey string, multiTenant bool) *server.MCPServer {
	s := server.NewMCPServer(
		serverName,
		serverVersion,
		server.WithToolCapabilities(true),
		server.WithPromptCapabilities(true),
		server.WithResourceCapabilities(true, false),
	)

	p := newProvider(host, credentials{token: token, webToken: webToken, apiKey: apiKey}, multiTenant)

	// login returns a JWT in its result text; never expose it in multi-tenant mode.
	if !multiTenant {
		registerLoginTool(s, host)
	}
	registerDatamodelTools(s, p)
	registerAlarmTools(s, p)
	registerTimeSeriesTools(s, p)
	registerDatasetTools(s, p)
	registerOperationTools(s, p)
	registerRuleTools(s, p)
	registerConnectorTools(s, p)
	registerProvisionTools(s, p)
	registerOpTypeTools(s, p)
	registerWorkspaceTools(s, p)
	registerDashboardTools(s, p)
	registerShareTools(s, p)
	registerIoTTools(s, p)
	registerPrompts(s)
	registerResources(s, p)

	return s
}

// ServeStdio starts the MCP server over stdio (always single-tenant).
func ServeStdio(host, token, webToken, apiKey string) error {
	s := newServer(host, token, webToken, apiKey, false)
	return server.ServeStdio(s)
}

// ServeHTTP starts the MCP server over HTTP (Streamable HTTP transport). When
// multiTenant is true, credentials are read per request from HTTP headers.
func ServeHTTP(addr, host, token, webToken, apiKey string, multiTenant bool) error {
	s := newServer(host, token, webToken, apiKey, multiTenant)

	opts := []server.StreamableHTTPOption{}
	if multiTenant {
		opts = append(opts, server.WithHTTPContextFunc(func(ctx context.Context, r *http.Request) context.Context {
			return withCredentials(ctx, credentials{
				token:    bearerToken(r.Header.Get("Authorization")),
				webToken: r.Header.Get("X-OG-Web-Token"),
				apiKey:   r.Header.Get("X-OG-Api-Key"),
			})
		}))
	}

	httpServer := server.NewStreamableHTTPServer(s, opts...)
	return httpServer.Start(addr)
}
