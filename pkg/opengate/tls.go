package opengate

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"time"
)

// defaultHTTPTimeout bounds a single request. It is generous enough for the
// slowest search endpoints and short enough that a wedged connection does not
// hang a long-running consumer forever. Override it with WithHTTPClient.
const defaultHTTPTimeout = 30 * time.Second

// tlsSettings is the TLS behaviour of one client. The zero value is the secure
// default: verify against the system root store.
type tlsSettings struct {
	insecure bool   // skip server certificate verification entirely
	caFile   string // extra CA/chain PEM appended to the system pool
}

// isDefault reports whether the settings need no custom transport at all.
func (s tlsSettings) isDefault() bool {
	return !s.insecure && s.caFile == ""
}

// validate checks the CA file eagerly so a bad path or PEM fails at
// construction time rather than on the first request.
func (s tlsSettings) validate() error {
	if s.caFile == "" {
		return nil
	}
	_, err := loadCAPool(s.caFile)
	return err
}

// config builds a *tls.Config from the settings. No ServerName is set: the HTTP
// transport derives it from the request URL. MQTT does not come through here —
// it builds its own config via mqttTLSConfig, which needs the broker host.
func (s tlsSettings) config() (*tls.Config, error) {
	cfg := &tls.Config{}
	if s.insecure {
		cfg.InsecureSkipVerify = true
		return cfg, nil
	}
	if s.caFile != "" {
		pool, err := loadCAPool(s.caFile)
		if err != nil {
			return nil, err
		}
		cfg.RootCAs = pool
	}
	return cfg, nil
}

// httpClient returns an HTTP client honouring these settings. When they are the
// secure default it leaves the standard transport untouched, so connection
// pooling is shared process-wide as the stdlib intends.
func (s tlsSettings) httpClient() (*http.Client, error) {
	c := &http.Client{Timeout: defaultHTTPTimeout}
	if s.isDefault() {
		return c, nil
	}
	cfg, err := s.config()
	if err != nil {
		return nil, err
	}
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = cfg
	c.Transport = tr
	return c, nil
}

// processTLS holds the process-wide TLS defaults set by ConfigureTLS. A Client
// inherits them at construction time unless WithTLS overrides them, and the
// MQTT transport reads them directly.
//
// Prefer WithTLS: a process-wide setting cannot express two endpoints with
// different certificate authorities.
var processTLS tlsSettings

// ConfigureTLS sets the process-wide TLS defaults inherited by clients created
// afterwards and by the MQTT transport. insecure skips server certificate
// verification entirely — the escape hatch for self-signed certs. caFile, when
// non-empty, appends an extra CA/chain PEM to the system pool. The caFile is
// validated eagerly so a bad path/PEM fails fast at startup rather than on
// first request.
//
// Deprecated: this is process-wide mutable state, so two clients talking to
// endpoints with different certificate authorities cannot both be configured
// correctly. Pass WithTLS to New instead. It remains for the CLI, which does
// have exactly one endpoint per invocation.
func ConfigureTLS(insecure bool, caFile string) error {
	s := tlsSettings{insecure: insecure, caFile: caFile}
	if err := s.validate(); err != nil {
		return err
	}
	processTLS = s
	return nil
}

// loadCAPool reads caFile and returns the system pool with its certs appended.
func loadCAPool(caFile string) (*x509.CertPool, error) {
	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
		pool = x509.NewCertPool()
	}
	pem, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("reading --ca-file %s: %w", caFile, err)
	}
	if !pool.AppendCertsFromPEM(pem) {
		return nil, fmt.Errorf("no valid certificates found in --ca-file %s", caFile)
	}
	return pool, nil
}

// NewHTTPClient returns an HTTP client honouring the process-wide TLS settings.
// Exported so other transports (e.g. the MCP server's shared client) can build
// a client that matches the CLI's TLS behaviour.
//
// Deprecated: prefer New with WithTLS, which keeps TLS per client. This helper
// cannot express more than one CA per process.
func NewHTTPClient() *http.Client {
	// processTLS was validated in ConfigureTLS, so the error path is unreachable.
	c, err := processTLS.httpClient()
	if err != nil {
		return &http.Client{Timeout: defaultHTTPTimeout}
	}
	return c
}
