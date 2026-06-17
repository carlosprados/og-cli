package config

import (
	"os"
	"path/filepath"
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
