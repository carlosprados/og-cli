package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

const (
	clientDesktop = "claude-desktop"
	clientCode    = "claude-code"

	desktopConfigFile = "claude_desktop_config.json"
	codeConfigFile    = ".mcp.json"
)

var (
	mcpInstallClient string
	mcpInstallName   string
	mcpInstallDir    string
	mcpInstallPrint  bool
)

var mcpInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Register the og MCP server in a Claude client config",
	Long: "Adds og as an MCP server to a Claude client so users don't have to edit\n" +
		"JSON by hand.\n\n" +
		"Two clients are supported (--client):\n" +
		"  claude-desktop  the Claude Desktop app — writes claude_desktop_config.json\n" +
		"                  (macOS: ~/Library/Application Support/Claude, Windows:\n" +
		"                  %APPDATA%\\Claude, Linux: ~/.config/Claude). Default.\n" +
		"  claude-code     the Claude Code CLI — writes a project-scoped .mcp.json in\n" +
		"                  the current directory (or --dir).\n\n" +
		"The entry uses the ABSOLUTE path of this og binary, because Claude Desktop\n" +
		"does not inherit your shell PATH — \"command\": \"og\" would fail otherwise.\n\n" +
		"Pass the global --profile to bake `--profile <name>` into the server args\n" +
		"(useful for one server per tenant).\n\n" +
		"The merge is non-destructive: other MCP servers in the file are preserved\n" +
		"and the original is backed up to <file>.bak before writing. Use --print to\n" +
		"see exactly what would be written without touching anything.",
	Args: cobra.NoArgs,
	RunE: runMCPInstall,
}

var mcpUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove the og MCP server from a Claude client config",
	Long: "Removes the og MCP server entry added by `og mcp install`, preserving any\n" +
		"other servers in the file. Same --client / --dir / --name flags.",
	Args: cobra.NoArgs,
	RunE: runMCPUninstall,
}

func init() {
	for _, c := range []*cobra.Command{mcpInstallCmd, mcpUninstallCmd} {
		c.Flags().StringVar(&mcpInstallClient, "client", clientDesktop, "target client: claude-desktop | claude-code")
		c.Flags().StringVar(&mcpInstallName, "name", "opengate", "name of the MCP server entry")
		c.Flags().StringVar(&mcpInstallDir, "dir", "", "directory holding the config file (overrides the per-OS default)")
	}
	mcpInstallCmd.Flags().BoolVar(&mcpInstallPrint, "print", false, "show the path and entry that would be written, without writing")

	mcpCmd.AddCommand(mcpInstallCmd, mcpUninstallCmd)
}

func runMCPInstall(cmd *cobra.Command, args []string) error {
	path, err := mcpConfigPath(mcpInstallClient, mcpInstallDir)
	if err != nil {
		return err
	}
	bin, err := ogBinaryPath()
	if err != nil {
		return err
	}
	entry := mcpServerEntry(bin, profile)

	if mcpInstallPrint {
		blob, _ := json.MarshalIndent(map[string]any{mcpInstallName: entry}, "", "  ")
		fmt.Printf("Would write to: %s\n\nmcpServers entry:\n%s\n", path, blob)
		return nil
	}

	cfg, existed, err := readMCPConfig(path)
	if err != nil {
		return err
	}
	_, updated := upsertServer(cfg, mcpInstallName, entry)

	if existed {
		if err := backupFile(path); err != nil {
			return err
		}
	}
	if err := writeMCPConfig(path, cfg); err != nil {
		return err
	}

	verb := "Added"
	if updated {
		verb = "Updated"
	}
	fmt.Printf("%s MCP server %q → %s\n", verb, mcpInstallName, path)
	fmt.Printf("  command: %s\n  args:    mcp --stdio%s\n", bin, profileSuffix(profile))
	if mcpInstallClient == clientDesktop {
		fmt.Println("\nRestart Claude Desktop for the change to take effect.")
	}
	fmt.Println("Make sure you have logged in (`og login`) so the server has a session.")
	return nil
}

func runMCPUninstall(cmd *cobra.Command, args []string) error {
	path, err := mcpConfigPath(mcpInstallClient, mcpInstallDir)
	if err != nil {
		return err
	}
	cfg, existed, err := readMCPConfig(path)
	if err != nil {
		return err
	}
	if !existed || !removeServer(cfg, mcpInstallName) {
		fmt.Printf("No MCP server %q found in %s — nothing to do.\n", mcpInstallName, path)
		return nil
	}
	if err := backupFile(path); err != nil {
		return err
	}
	if err := writeMCPConfig(path, cfg); err != nil {
		return err
	}
	fmt.Printf("Removed MCP server %q from %s\n", mcpInstallName, path)
	return nil
}

// mcpConfigPath resolves the config file path for the given client. A non-empty
// dir overrides the per-OS default directory; the filename is fixed per client.
func mcpConfigPath(client, dir string) (string, error) {
	var file string
	switch client {
	case clientDesktop:
		file = desktopConfigFile
		if dir == "" {
			d, err := desktopConfigDir()
			if err != nil {
				return "", err
			}
			dir = d
		}
	case clientCode:
		file = codeConfigFile
		if dir == "" {
			dir = "."
		}
	default:
		return "", fmt.Errorf("unknown client %q (use claude-desktop or claude-code)", client)
	}
	return filepath.Join(dir, file), nil
}

// desktopConfigDir returns the Claude Desktop config directory for the current
// OS. Linux has no official Claude Desktop; the community build uses ~/.config.
func desktopConfigDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "Claude"), nil
		}
		return "", fmt.Errorf("APPDATA is not set")
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support", "Claude"), nil
	default: // linux and others
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			return filepath.Join(xdg, "Claude"), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".config", "Claude"), nil
	}
}

// ogBinaryPath returns the absolute, symlink-resolved path of the running og
// binary, so the config points at a binary Claude can launch without a PATH.
func ogBinaryPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locating og binary: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return exe, nil
}

func mcpServerEntry(bin, profile string) map[string]any {
	args := []string{"mcp", "--stdio"}
	if profile != "" {
		args = append(args, "--profile", profile)
	}
	return map[string]any{"command": bin, "args": args}
}

func profileSuffix(profile string) string {
	if profile == "" {
		return ""
	}
	return " --profile " + profile
}

// readMCPConfig reads and parses the config file. A missing file yields an empty
// config and existed=false. A malformed file is an error (we never clobber it).
func readMCPConfig(path string) (cfg map[string]any, existed bool, err error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]any{}, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("reading %s: %w", path, err)
	}
	if len(data) == 0 {
		return map[string]any{}, true, nil
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, true, fmt.Errorf("%s is not valid JSON (refusing to overwrite): %w", path, err)
	}
	return cfg, true, nil
}

// upsertServer sets cfg["mcpServers"][name] = entry, creating the map if needed.
// updated is true if an entry under that name already existed.
func upsertServer(cfg map[string]any, name string, entry map[string]any) (map[string]any, bool) {
	servers, _ := cfg["mcpServers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}
	_, updated := servers[name]
	servers[name] = entry
	cfg["mcpServers"] = servers
	return cfg, updated
}

// removeServer deletes the named entry; returns true if it was present.
func removeServer(cfg map[string]any, name string) bool {
	servers, _ := cfg["mcpServers"].(map[string]any)
	if servers == nil {
		return false
	}
	if _, ok := servers[name]; !ok {
		return false
	}
	delete(servers, name)
	cfg["mcpServers"] = servers
	return true
}

func backupFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s for backup: %w", path, err)
	}
	if err := os.WriteFile(path+".bak", data, 0o644); err != nil {
		return fmt.Errorf("writing backup: %w", err)
	}
	return nil
}

// writeMCPConfig writes cfg as indented JSON atomically (temp + rename), creating
// the parent directory if needed.
func writeMCPConfig(path string, cfg map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	blob, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	blob = append(blob, '\n')

	tmp, err := os.CreateTemp(filepath.Dir(path), ".mcp-config-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(blob); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
