package mcp

import (
	"github.com/carlosprados/og-cli/internal/client"
	"github.com/mark3labs/mcp-go/server"
)

const (
	serverName    = "og-mcp"
	serverVersion = "0.1.0"
)

// newServer creates a configured MCP server with all tools, prompts, and resources.
func newServer(host, token, apiKey string) *server.MCPServer {
	s := server.NewMCPServer(
		serverName,
		serverVersion,
		server.WithToolCapabilities(true),
		server.WithPromptCapabilities(true),
		server.WithResourceCapabilities(true, false),
	)

	c := client.New(host, token)

	registerTools(s, host, token)
	registerAlarmTools(s, c)
	registerTimeSeriesTools(s, c)
	registerDatasetTools(s, c)
	registerIoTTools(s, host, apiKey)
	registerPrompts(s)
	registerResources(s, c)

	return s
}

// ServeStdio starts the MCP server over stdio.
func ServeStdio(host, token, apiKey string) error {
	s := newServer(host, token, apiKey)
	return server.ServeStdio(s)
}

// ServeHTTP starts the MCP server over HTTP (Streamable HTTP transport).
func ServeHTTP(addr, host, token, apiKey string) error {
	s := newServer(host, token, apiKey)
	httpServer := server.NewStreamableHTTPServer(s)
	return httpServer.Start(addr)
}
