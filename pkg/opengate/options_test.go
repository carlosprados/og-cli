package opengate

import (
	"context"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// writeServerCert dumps a test server's self-signed certificate as PEM so it can
// be handed to WithTLS as a CA file.
func writeServerCert(t *testing.T, srv *httptest.Server) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ca.pem")
	block := &pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw}
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestTLSPerClientIsolation is the T7 acceptance criterion: several clients with
// different TLS configurations coexist without interfering. Before per-client
// TLS this was impossible — ConfigureTLS was process-wide, so the last caller
// won and every other client silently inherited its policy.
func TestTLSPerClientIsolation(t *testing.T) {
	resetTLS(t)

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	caFile := writeServerCert(t, srv)

	insecureClient := New(srv.URL, "tok", WithTLS(true, ""))
	caClient := New(srv.URL, "tok", WithTLS(false, caFile))
	strictClient := New(srv.URL, "tok") // secure default: system roots only

	for _, c := range []*Client{insecureClient, caClient, strictClient} {
		if err := c.Err(); err != nil {
			t.Fatalf("unexpected construction error: %v", err)
		}
	}

	// Interleave the requests so any shared state would show up as cross-talk.
	for i := 0; i < 2; i++ {
		if _, status, err := insecureClient.Get(context.Background(), "/ping"); err != nil || status != http.StatusOK {
			t.Errorf("insecure client: status %d, err %v", status, err)
		}
		if _, status, err := caClient.Get(context.Background(), "/ping"); err != nil || status != http.StatusOK {
			t.Errorf("ca-file client: status %d, err %v", status, err)
		}
		if _, _, err := strictClient.Get(context.Background(), "/ping"); err == nil {
			t.Error("strict client must reject the self-signed certificate")
		}
	}
}

// TestTLSDoesNotLeakIntoProcessState guards the other direction: configuring one
// client must not mutate the process-wide defaults that MQTT and the CLI read.
func TestTLSDoesNotLeakIntoProcessState(t *testing.T) {
	resetTLS(t)

	_ = New("https://example.invalid", "tok", WithTLS(true, ""))

	if !processTLS.isDefault() {
		t.Error("WithTLS must not touch the process-wide TLS defaults")
	}
}

// TestTLSInheritsProcessDefaults keeps the CLI working: it calls ConfigureTLS
// once at start-up and then constructs clients with no TLS options at all.
func TestTLSInheritsProcessDefaults(t *testing.T) {
	resetTLS(t)

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	if err := ConfigureTLS(true, ""); err != nil {
		t.Fatal(err)
	}
	c := New(srv.URL, "tok")
	if _, status, err := c.Get(context.Background(), "/ping"); err != nil || status != http.StatusOK {
		t.Errorf("client should inherit process-wide insecure: status %d, err %v", status, err)
	}
}

// TestBadCAFileSurfacesViaErr checks the deferred-error contract of New: a bad
// configuration is reported by Err and by every request, instead of panicking or
// being silently ignored.
func TestBadCAFileSurfacesViaErr(t *testing.T) {
	resetTLS(t)

	c := New("https://example.invalid", "tok", WithTLS(false, "/no/such/ca.pem"))
	if c.Err() == nil {
		t.Fatal("expected Err to report the unreadable ca-file")
	}
	if _, _, err := c.Get(context.Background(), "/ping"); err == nil {
		t.Error("requests must fail while the client is misconfigured")
	}
	if _, _, _, err := c.PostMultipartFile(context.Background(), "/x", "f", "/no/such/file.xlsx", ""); err == nil {
		t.Error("multipart upload must fail while the client is misconfigured")
	}
}

// TestAPIVersionResolution is the T8 acceptance check: the version segment comes
// from configuration, so retargeting an on-premises instance pinned to another
// version needs no fork.
func TestAPIVersionResolution(t *testing.T) {
	resetTLS(t)

	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tests := []struct {
		name string
		opts []Option
		want string
	}{
		{"default", nil, "/north/v80/search/devices"},
		{"override", []Option{WithAPIVersion("v81")}, "/north/v81/search/devices"},
		{"tolerates slashes", []Option{WithAPIVersion("/v79/")}, "/north/v79/search/devices"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := New(srv.URL, "tok", tc.opts...)
			if _, _, err := c.Get(context.Background(), searchDevicesPath); err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("path = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestNorthAuthHeaders pins the credential each plane uses. The API key must
// never reach the Web API, which only understands its own bearer token.
func TestNorthAuthHeaders(t *testing.T) {
	resetTLS(t)

	var authz, apiKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authz = r.Header.Get("Authorization")
		apiKey = r.Header.Get("X-ApiKey")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	t.Run("jwt by default", func(t *testing.T) {
		c := New(srv.URL, "jwt-token")
		if _, _, err := c.Get(context.Background(), "/north/{v}/x"); err != nil {
			t.Fatal(err)
		}
		if authz != "Bearer jwt-token" {
			t.Errorf("Authorization = %q", authz)
		}
		if apiKey != "" {
			t.Errorf("X-ApiKey should be absent, got %q", apiKey)
		}
	})

	t.Run("api key wins over jwt", func(t *testing.T) {
		authz, apiKey = "", ""
		c := New(srv.URL, "jwt-token", WithAPIKey("secret-key"))
		if _, _, err := c.Get(context.Background(), "/north/{v}/x"); err != nil {
			t.Fatal(err)
		}
		if apiKey != "secret-key" {
			t.Errorf("X-ApiKey = %q", apiKey)
		}
		if authz != "" {
			t.Errorf("Authorization should be absent when using an API key, got %q", authz)
		}
	})

	t.Run("web api keeps its own bearer", func(t *testing.T) {
		authz, apiKey = "", ""
		c := New(srv.URL, "jwt-token", WithAPIKey("secret-key")).WithWebToken("web-token")
		if _, _, err := c.WebGet(context.Background(), "/api/workspaces"); err != nil {
			t.Fatal(err)
		}
		if authz != "Bearer web-token" {
			t.Errorf("Authorization = %q, want the web token", authz)
		}
		if apiKey != "" {
			t.Errorf("the API key must not leak into the Web API, got %q", apiKey)
		}
	})
}

// TestWithHTTPClient checks the injected client is the one actually used.
func TestWithHTTPClient(t *testing.T) {
	resetTLS(t)

	custom := &http.Client{}
	c := New("https://example.invalid", "tok", WithHTTPClient(custom))
	if c.HTTPClient != custom {
		t.Error("WithHTTPClient must install the given client")
	}
}
