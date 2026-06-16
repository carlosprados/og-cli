package mcp

import (
	"context"
	"testing"
)

func TestBearerToken(t *testing.T) {
	cases := map[string]string{
		"Bearer abc.def.ghi": "abc.def.ghi",
		"bearer-less-token":  "bearer-less-token",
		"Bearer  spaced ":    "spaced",
		"":                   "",
	}
	for in, want := range cases {
		if got := bearerToken(in); got != want {
			t.Errorf("bearerToken(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestProviderSingleTenant(t *testing.T) {
	fixed := credentials{token: "tok", webToken: "web", apiKey: "key"}
	p := newProvider("https://api.opengate.es", fixed, false)

	// single-tenant ignores context and always returns the fixed credentials
	c, errRes := p.client(context.Background())
	if errRes != nil {
		t.Fatalf("unexpected error result: %v", errRes)
	}
	if c.Token != "tok" || c.WebToken != "web" {
		t.Errorf("client built with wrong creds: token=%q web=%q", c.Token, c.WebToken)
	}
	key, errRes := p.apiKey(context.Background())
	if errRes != nil || key != "key" {
		t.Errorf("apiKey = %q, errRes=%v", key, errRes)
	}
}

func TestProviderMultiTenantRequiresToken(t *testing.T) {
	p := newProvider("https://api.opengate.es", credentials{token: "startup", apiKey: "startupkey"}, true)

	// no credentials in context → must NOT fall back to the startup profile
	if _, errRes := p.client(context.Background()); errRes == nil {
		t.Error("multi-tenant client() must error when no token in context (no startup fallback)")
	}
	if _, errRes := p.apiKey(context.Background()); errRes == nil {
		t.Error("multi-tenant apiKey() must error when no credentials in context")
	}
}

func TestProviderMultiTenantUsesRequestCreds(t *testing.T) {
	p := newProvider("https://api.opengate.es", credentials{token: "startup"}, true)
	ctx := withCredentials(context.Background(), credentials{token: "user-jwt", webToken: "user-web", apiKey: "user-key"})

	c, errRes := p.client(ctx)
	if errRes != nil {
		t.Fatalf("unexpected error: %v", errRes)
	}
	if c.Token != "user-jwt" || c.WebToken != "user-web" {
		t.Errorf("client should carry per-request creds, got token=%q web=%q", c.Token, c.WebToken)
	}
	if c.Token == "startup" {
		t.Error("must NOT use the startup token in multi-tenant mode")
	}
	key, errRes := p.apiKey(ctx)
	if errRes != nil || key != "user-key" {
		t.Errorf("apiKey = %q, errRes=%v", key, errRes)
	}
}

func TestProviderMultiTenantApiKeyRequiredOnlyWhenAbsent(t *testing.T) {
	p := newProvider("h", credentials{}, true)
	// token present but apiKey absent → client() OK (Bearer is enough), apiKey() errors
	ctx := withCredentials(context.Background(), credentials{token: "jwt"})
	if _, errRes := p.client(ctx); errRes != nil {
		t.Errorf("client() should succeed with only a token: %v", errRes)
	}
	if _, errRes := p.apiKey(ctx); errRes == nil {
		t.Error("apiKey() should error when X-OG-Api-Key is absent in multi-tenant mode")
	}
}

func TestProviderSharesHTTPClient(t *testing.T) {
	p := newProvider("h", credentials{token: "t"}, false)
	c1, _ := p.client(context.Background())
	c2, _ := p.client(context.Background())
	if c1.HTTPClient != c2.HTTPClient {
		t.Error("per-request clients should share one *http.Client to reuse connections")
	}
}
