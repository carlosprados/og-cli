package config

import "github.com/carlosprados/og-cli/v2/pkg/opengate"

// ClientOptions turns a resolved profile into the options every og surface
// passes to opengate.New, so the CLI, the TUI and the MCP server configure their
// clients identically instead of each remembering to thread a new setting.
//
// TLS is deliberately absent: the CLI resolves it once at start-up via
// opengate.ConfigureTLS, which the MQTT transport also reads, and a client
// inherits it from there. One endpoint per invocation is exactly the case that
// process-wide setting models correctly.
func (p *Profile) ClientOptions() []opengate.Option {
	if p == nil {
		return nil
	}

	var opts []opengate.Option
	if p.APIVersion != "" {
		opts = append(opts, opengate.WithAPIVersion(p.APIVersion))
	}
	if p.Retries > 1 {
		policy := opengate.DefaultRetryPolicy()
		policy.Attempts = p.Retries
		opts = append(opts, opengate.WithRetry(policy))
	}
	return opts
}
