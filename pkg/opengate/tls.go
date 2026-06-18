package opengate

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"time"
)

// Process-wide TLS behaviour for every HTTP and MQTT client created afterwards.
// These are set once at startup (from CLI flags / env / profile) via ConfigureTLS.
// Defaults are secure: verify against the system root store.
var (
	tlsInsecure bool
	tlsCAFile   string
)

// ConfigureTLS sets the process-wide TLS behaviour applied to all HTTP and MQTT
// clients created afterwards. insecure skips server certificate verification
// entirely — the escape hatch for self-signed certs. caFile, when non-empty,
// appends an extra CA/chain PEM to the system pool. The caFile is validated
// eagerly so a bad path/PEM fails fast at startup rather than on first request.
func ConfigureTLS(insecure bool, caFile string) error {
	if caFile != "" {
		if _, err := loadCAPool(caFile); err != nil {
			return err
		}
	}
	tlsInsecure = insecure
	tlsCAFile = caFile
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

// tlsConfig builds a *tls.Config from the process-wide settings. serverName may
// be empty for HTTP (the transport derives it from the request URL); MQTT passes
// the broker host so verification has a name to check against.
func tlsConfig(serverName string) (*tls.Config, error) {
	cfg := &tls.Config{}
	if serverName != "" {
		cfg.ServerName = serverName
	}
	if tlsInsecure {
		cfg.InsecureSkipVerify = true
		return cfg, nil
	}
	if tlsCAFile != "" {
		pool, err := loadCAPool(tlsCAFile)
		if err != nil {
			return nil, err
		}
		cfg.RootCAs = pool
	}
	return cfg, nil
}

// NewHTTPClient returns the HTTP client used by every REST call (North and South
// planes). When custom TLS is configured it clones the default transport and
// applies the TLS config; otherwise it leaves the default transport untouched.
// Exported so other transports (e.g. the MCP server's shared client) can build a
// client that honours the process-wide TLS settings.
func NewHTTPClient() *http.Client {
	c := &http.Client{Timeout: 30 * time.Second}
	if tlsInsecure || tlsCAFile != "" {
		// caFile validated in ConfigureTLS, so the error path is unreachable here.
		cfg, _ := tlsConfig("")
		tr := http.DefaultTransport.(*http.Transport).Clone()
		tr.TLSClientConfig = cfg
		c.Transport = tr
	}
	return c
}
