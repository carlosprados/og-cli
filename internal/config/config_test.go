package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestSaveCredentialsTargetsNamedProfile guards the bug where logging in with a
// non-default --profile clobbered the default profile instead of the selected one.
func TestSaveCredentialsTargetsNamedProfile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	existing := `default_profile: default
profiles:
  default:
    host: https://default.example.com
    token: default-token
    organization: ORG_DEFAULT
  prod:
    host: https://prod.example.com
    organization: ORG_PROD
`
	if err := os.WriteFile(cfgPath, []byte(existing), 0o600); err != nil {
		t.Fatal(err)
	}

	creds := Credentials{Token: "prod-token", APIKey: "prod-key", Email: "u@example.com"}
	if err := SaveCredentials("prod", creds, cfgPath); err != nil {
		t.Fatalf("SaveCredentials: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	prod := cfg.Profiles["prod"]
	if prod.Token != "prod-token" || prod.APIKey != "prod-key" {
		t.Errorf("prod profile not updated: %+v", prod)
	}
	if prod.Host != "https://prod.example.com" {
		t.Errorf("prod host should be preserved, got %q", prod.Host)
	}
	if prod.Organization != "ORG_PROD" {
		t.Errorf("prod org should be preserved, got %q", prod.Organization)
	}

	def := cfg.Profiles["default"]
	if def.Token != "default-token" {
		t.Errorf("default profile token was clobbered: %q", def.Token)
	}
	if def.Organization != "ORG_DEFAULT" {
		t.Errorf("default profile org was clobbered: %q", def.Organization)
	}
}

// TestSaveCredentialsPersistsTLS guards that `og login --insecure` (and --ca-file)
// are written to the profile so later commands inherit them.
func TestSaveCredentialsPersistsTLS(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	creds := Credentials{Token: "tok", Insecure: true, CAFile: "/etc/og/ca.pem"}
	if err := SaveCredentials("default", creds, cfgPath); err != nil {
		t.Fatalf("SaveCredentials: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	p, err := cfg.ActiveProfile("default")
	if err != nil {
		t.Fatalf("ActiveProfile: %v", err)
	}
	if !p.Insecure {
		t.Error("insecure not persisted/loaded")
	}
	if p.CAFile != "/etc/og/ca.pem" {
		t.Errorf("ca_file = %q, want /etc/og/ca.pem", p.CAFile)
	}
}

// TestSaveCredentialsPersistsTOTPSecret checks the TOTP seed round-trips and
// that the config file is locked down to 0600 (it holds a master credential).
func TestSaveCredentialsPersistsTOTPSecret(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	creds := Credentials{Token: "tok", TOTPSecret: "JBSWY3DPEHPK3PXP"}
	if err := SaveCredentials("default", creds, cfgPath); err != nil {
		t.Fatalf("SaveCredentials: %v", err)
	}

	info, err := os.Stat(cfgPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	// Unix permission bits are not enforced on Windows: os.Chmod only toggles the
	// read-only attribute there, so Stat reports 0666 regardless. Assert the 0600
	// lock-down only where the OS actually honors it.
	if runtime.GOOS != "windows" {
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Errorf("config perms = %o, want 600", perm)
		}
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	p, err := cfg.ActiveProfile("default")
	if err != nil {
		t.Fatalf("ActiveProfile: %v", err)
	}
	if p.TOTPSecret != "JBSWY3DPEHPK3PXP" {
		t.Errorf("totp_secret = %q, want JBSWY3DPEHPK3PXP", p.TOTPSecret)
	}
}

// TestActiveProfileTOTPSecretEnvOverride guards the OG_2FA_SECRET env override.
func TestActiveProfileTOTPSecretEnvOverride(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	existing := `default_profile: default
profiles:
  default:
    host: https://default.example.com
    totp_secret: STOREDSECRET
`
	if err := os.WriteFile(cfgPath, []byte(existing), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("OG_2FA_SECRET", "ENVSECRET")

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	p, err := cfg.ActiveProfile("default")
	if err != nil {
		t.Fatalf("ActiveProfile: %v", err)
	}
	if p.TOTPSecret != "ENVSECRET" {
		t.Errorf("OG_2FA_SECRET override = %q, want ENVSECRET", p.TOTPSecret)
	}
}

// TestActiveProfileTLSEnvOverride guards the OG_INSECURE / OG_CA_FILE env overrides.
func TestActiveProfileTLSEnvOverride(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	existing := `default_profile: default
profiles:
  default:
    host: https://default.example.com
`
	if err := os.WriteFile(cfgPath, []byte(existing), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("OG_INSECURE", "true")
	t.Setenv("OG_CA_FILE", "/tmp/ca.pem")

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	p, err := cfg.ActiveProfile("default")
	if err != nil {
		t.Fatalf("ActiveProfile: %v", err)
	}
	if !p.Insecure {
		t.Error("OG_INSECURE override not applied")
	}
	if p.CAFile != "/tmp/ca.pem" {
		t.Errorf("OG_CA_FILE override = %q", p.CAFile)
	}
}
