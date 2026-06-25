package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestUpsertServerCreatesAndUpdates(t *testing.T) {
	cfg := map[string]any{}
	_, updated := upsertServer(cfg, "opengate", map[string]any{"command": "og"})
	if updated {
		t.Fatal("first insert reported as update")
	}
	servers := cfg["mcpServers"].(map[string]any)
	if _, ok := servers["opengate"]; !ok {
		t.Fatal("server not inserted")
	}

	_, updated = upsertServer(cfg, "opengate", map[string]any{"command": "og2"})
	if !updated {
		t.Fatal("second insert not reported as update")
	}
}

func TestUpsertServerPreservesOthers(t *testing.T) {
	cfg := map[string]any{
		"mcpServers":  map[string]any{"keep": map[string]any{"command": "x"}},
		"otherKey":    "value",
		"nestedBlock": map[string]any{"a": 1},
	}
	upsertServer(cfg, "opengate", map[string]any{"command": "og"})

	servers := cfg["mcpServers"].(map[string]any)
	if _, ok := servers["keep"]; !ok {
		t.Fatal("existing server was dropped")
	}
	if cfg["otherKey"] != "value" {
		t.Fatal("top-level key was dropped")
	}
}

func TestRemoveServer(t *testing.T) {
	cfg := map[string]any{
		"mcpServers": map[string]any{
			"opengate": map[string]any{"command": "og"},
			"other":    map[string]any{"command": "x"},
		},
	}
	if !removeServer(cfg, "opengate") {
		t.Fatal("removeServer returned false for present entry")
	}
	servers := cfg["mcpServers"].(map[string]any)
	if _, ok := servers["opengate"]; ok {
		t.Fatal("entry not removed")
	}
	if _, ok := servers["other"]; !ok {
		t.Fatal("sibling entry was removed")
	}
	if removeServer(cfg, "opengate") {
		t.Fatal("removeServer returned true for absent entry")
	}
}

func TestReadMCPConfigMissingAndMalformed(t *testing.T) {
	dir := t.TempDir()

	cfg, existed, err := readMCPConfig(filepath.Join(dir, "nope.json"))
	if err != nil || existed || len(cfg) != 0 {
		t.Fatalf("missing file: cfg=%v existed=%v err=%v", cfg, existed, err)
	}

	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte("{ not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readMCPConfig(bad); err == nil {
		t.Fatal("expected error on malformed JSON")
	}
}

func TestMCPConfigPath(t *testing.T) {
	p, err := mcpConfigPath(clientCode, "/tmp/proj")
	if err != nil || p != filepath.Join("/tmp/proj", ".mcp.json") {
		t.Fatalf("claude-code path = %q err=%v", p, err)
	}
	p, err = mcpConfigPath(clientDesktop, "/tmp/cfg")
	if err != nil || p != filepath.Join("/tmp/cfg", "claude_desktop_config.json") {
		t.Fatalf("claude-desktop path = %q err=%v", p, err)
	}
	if _, err := mcpConfigPath("bogus", ""); err == nil {
		t.Fatal("expected error for unknown client")
	}
}

func TestMCPServerEntry(t *testing.T) {
	e := mcpServerEntry("/usr/bin/og", "")
	if e["command"] != "/usr/bin/og" {
		t.Fatalf("command = %v", e["command"])
	}
	args := e["args"].([]string)
	if len(args) != 3 || args[0] != "mcp" || args[1] != "--stdio" || args[2] != "--lean" {
		t.Fatalf("args = %v", args)
	}
	e = mcpServerEntry("/usr/bin/og", "prod")
	args = e["args"].([]string)
	if len(args) != 5 || args[3] != "--profile" || args[4] != "prod" {
		t.Fatalf("args with profile = %v", args)
	}
}

func TestDesktopConfigDirCurrentOS(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", `C:\Users\x\AppData\Roaming`)
	}
	dir, err := desktopConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(dir, "Claude") {
		t.Fatalf("desktop dir %q does not contain Claude", dir)
	}
}

// End-to-end: install then uninstall against a temp file via --dir, asserting
// the on-disk JSON round-trips and other content survives.
func TestInstallUninstallRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mcp.json")
	if err := os.WriteFile(path, []byte(`{"mcpServers":{"keep":{"command":"x"}},"top":"v"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	prevClient, prevName, prevDir, prevPrint := mcpInstallClient, mcpInstallName, mcpInstallDir, mcpInstallPrint
	t.Cleanup(func() {
		mcpInstallClient, mcpInstallName, mcpInstallDir, mcpInstallPrint = prevClient, prevName, prevDir, prevPrint
	})
	mcpInstallClient, mcpInstallName, mcpInstallDir, mcpInstallPrint = clientCode, "opengate", dir, false

	if err := runMCPInstall(mcpInstallCmd, nil); err != nil {
		t.Fatalf("install: %v", err)
	}
	cfg := readJSON(t, path)
	servers := cfg["mcpServers"].(map[string]any)
	if _, ok := servers["opengate"]; !ok {
		t.Fatal("opengate not installed")
	}
	if _, ok := servers["keep"]; !ok {
		t.Fatal("install dropped the existing server")
	}
	if cfg["top"] != "v" {
		t.Fatal("install dropped top-level key")
	}
	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Fatal("no backup written on install over existing file")
	}

	if err := runMCPUninstall(mcpUninstallCmd, nil); err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	cfg = readJSON(t, path)
	servers = cfg["mcpServers"].(map[string]any)
	if _, ok := servers["opengate"]; ok {
		t.Fatal("uninstall left opengate behind")
	}
	if _, ok := servers["keep"]; !ok {
		t.Fatal("uninstall removed a sibling server")
	}
}

func readJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}
	return m
}
