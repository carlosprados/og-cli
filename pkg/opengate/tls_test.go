package opengate

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

// resetTLS restores the process-wide TLS state so tests don't leak into each other.
func resetTLS(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { processTLS = tlsSettings{} })
}

func TestConfigureTLS(t *testing.T) {
	resetTLS(t)

	// Default secure state.
	if err := ConfigureTLS(false, ""); err != nil {
		t.Fatalf("default: unexpected error: %v", err)
	}
	if !processTLS.isDefault() {
		t.Error("default ConfigureTLS should leave secure state")
	}

	// insecure sets the flag.
	if err := ConfigureTLS(true, ""); err != nil {
		t.Fatalf("insecure: unexpected error: %v", err)
	}
	if !processTLS.insecure {
		t.Error("insecure not recorded")
	}

	// A garbage ca-file must fail fast and not mutate state.
	bad := filepath.Join(t.TempDir(), "bad.pem")
	if err := os.WriteFile(bad, []byte("not a cert"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ConfigureTLS(false, bad); err == nil {
		t.Error("expected error for a PEM with no valid certificates")
	}

	// A missing ca-file must error.
	if err := ConfigureTLS(false, "/no/such/file.pem"); err == nil {
		t.Error("expected error for a missing ca-file")
	}
}

func TestNewHTTPClientTransport(t *testing.T) {
	resetTLS(t)

	// Secure default: no custom transport, no skip-verify.
	if err := ConfigureTLS(false, ""); err != nil {
		t.Fatal(err)
	}
	if c := NewHTTPClient(); c.Transport != nil {
		t.Error("secure default should leave the standard transport untouched")
	}

	// insecure: custom transport with InsecureSkipVerify.
	if err := ConfigureTLS(true, ""); err != nil {
		t.Fatal(err)
	}
	c := NewHTTPClient()
	tr, ok := c.Transport.(*http.Transport)
	if !ok {
		t.Fatal("insecure should install a custom *http.Transport")
	}
	if tr.TLSClientConfig == nil || !tr.TLSClientConfig.InsecureSkipVerify {
		t.Error("insecure transport must set InsecureSkipVerify")
	}
}
