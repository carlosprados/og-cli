package mcp

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/carlosprados/og-cli/pkg/opengate"
	"github.com/mark3labs/mcp-go/mcp"
)

// credentials are the three per-identity secrets OpenGate uses. North operations
// use the JWT (token); workspaces/dashboards use a separate web JWT; South/IoT
// uses the API key.
type credentials struct {
	token    string
	webToken string
	apiKey   string
}

type ctxKey int

const credsCtxKey ctxKey = 0

func withCredentials(ctx context.Context, c credentials) context.Context {
	return context.WithValue(ctx, credsCtxKey, c)
}

func credentialsFromContext(ctx context.Context) (credentials, bool) {
	c, ok := ctx.Value(credsCtxKey).(credentials)
	return c, ok
}

// bearerToken extracts the token from an "Authorization: Bearer <token>" header.
func bearerToken(header string) string {
	const prefix = "Bearer "
	if strings.HasPrefix(header, prefix) {
		return strings.TrimSpace(header[len(prefix):])
	}
	return strings.TrimSpace(header)
}

// provider builds per-request OpenGate clients for MCP tool handlers.
//
//   - single-tenant (stdio, or HTTP without --multi-tenant): every handler gets
//     the fixed credentials captured from the startup profile.
//   - multi-tenant (HTTP with --multi-tenant): credentials are read per request
//     from the HTTP headers (stashed in the context by WithHTTPContextFunc). The
//     server is a stateless conduit — it never falls back to the startup profile
//     when a required header is absent (that would leak one user's identity to
//     another); it returns an error instead.
//
// The host is always a fixed server config (one og MCP server = one OpenGate
// instance); only the credentials vary per request.
type provider struct {
	host        string
	multiTenant bool
	fixed       credentials
	httpClient  *http.Client // shared across per-request clients to reuse connections
}

func newProvider(host string, fixed credentials, multiTenant bool) *provider {
	return &provider{
		host:        host,
		multiTenant: multiTenant,
		fixed:       fixed,
		// The MCP server serves one host per process, so the process-wide TLS
		// settings resolved at startup are the right ones for every session.
		httpClient: opengate.NewHTTPClient(), //nolint:staticcheck // SA1019: intentional, see above
	}
}

// credsErr resolves the credentials for the current request. In multi-tenant mode
// a missing/empty token is a hard error (no startup-profile fallback).
func (p *provider) credsErr(ctx context.Context) (credentials, error) {
	if !p.multiTenant {
		return p.fixed, nil
	}
	c, ok := credentialsFromContext(ctx)
	if !ok || c.token == "" {
		return credentials{}, errors.New("missing Authorization: Bearer token (multi-tenant MCP requires a per-request token)")
	}
	return c, nil
}

// clientErr returns an OpenGate client for the current request, carrying the north
// JWT and the web token (the latter only matters for workspace/dashboard tools,
// which surface a clear error if it is absent).
func (p *provider) clientErr(ctx context.Context) (*opengate.Client, error) {
	creds, err := p.credsErr(ctx)
	if err != nil {
		return nil, err
	}
	c := opengate.New(p.host, creds.token).WithWebToken(creds.webToken)
	c.HTTPClient = p.httpClient
	return c, nil
}

// apiKeyErr returns the South/IoT API key for the current request. In multi-tenant
// mode an absent key is an error (these tools require it).
func (p *provider) apiKeyErr(ctx context.Context) (string, error) {
	creds, err := p.credsErr(ctx)
	if err != nil {
		return "", err
	}
	if p.multiTenant && creds.apiKey == "" {
		return "", errors.New("missing X-OG-Api-Key header (required for South/IoT tools in multi-tenant mode)")
	}
	return creds.apiKey, nil
}

// client and apiKey are tool-facing adapters returning an MCP error result.
func (p *provider) client(ctx context.Context) (*opengate.Client, *mcp.CallToolResult) {
	c, err := p.clientErr(ctx)
	if err != nil {
		return nil, mcp.NewToolResultError(err.Error())
	}
	return c, nil
}

func (p *provider) apiKey(ctx context.Context) (string, *mcp.CallToolResult) {
	k, err := p.apiKeyErr(ctx)
	if err != nil {
		return "", mcp.NewToolResultError(err.Error())
	}
	return k, nil
}
