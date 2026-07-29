package cmd

import (
	"fmt"
	"os"

	ogmcp "github.com/carlosprados/og-cli/v2/internal/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the MCP (Model Context Protocol) server",
	Long: `Starts an MCP server that exposes OpenGate API operations to an LLM.

Surface modes (control how much context the tool definitions cost):
  --lean              expose just og_exec/og_help (the whole CLI through 2 tools);
                      the model discovers the surface on demand. Smallest footprint.
  --toolsets <names>  expose named tool groups. Aliases: observe (curated read core),
                      readonly (all read), all (everything); or specific groups such
                      as "devices,alarms-ops". Use og_toolsets to list them.
  --all-tools         alias of --toolsets all.

Per-transport defaults when no surface flag is given:
  stdio  -> --lean        (shell-capable, trusted, single user)
  http   -> --toolsets observe`,
	RunE: runMCP,
}

var (
	mcpStdio       bool
	mcpHTTP        string
	mcpMultiTenant bool
	mcpLean        bool
	mcpToolsets    []string
	mcpAllTools    bool
)

func init() {
	mcpCmd.Flags().BoolVar(&mcpStdio, "stdio", true, "serve over stdio (default)")
	mcpCmd.Flags().StringVar(&mcpHTTP, "http", "", "serve over HTTP at the given address (e.g. :8080)")
	mcpCmd.Flags().BoolVar(&mcpMultiTenant, "multi-tenant", false,
		"HTTP only: take credentials per request from headers (Authorization: Bearer, X-OG-Web-Token, X-OG-Api-Key) instead of the startup profile. Requires TLS in front. Drops the login tool.")
	mcpCmd.Flags().BoolVar(&mcpLean, "lean", false,
		"expose only og_exec/og_help (exec-over-CLI) instead of named tools — smallest token footprint. Default for stdio.")
	mcpCmd.Flags().StringSliceVar(&mcpToolsets, "toolsets", nil,
		"named tool groups to expose: observe, readonly, all, or specific groups (e.g. devices,alarms-ops). Default for HTTP: observe.")
	mcpCmd.Flags().BoolVar(&mcpAllTools, "all-tools", false, "expose all tools (alias of --toolsets all)")
	rootCmd.AddCommand(mcpCmd)
}

// mcpSurface resolves the surface intent from the flags, applying the
// per-transport default when no surface flag is set.
func mcpSurface(isHTTP bool) (lean bool, toolsetNames []string) {
	switch {
	case mcpAllTools:
		return false, []string{"all"}
	case len(mcpToolsets) > 0:
		return false, mcpToolsets
	case mcpLean:
		return true, nil
	default:
		if isHTTP {
			return false, []string{"observe"}
		}
		return true, nil
	}
}

func runMCP(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}

	isHTTP := mcpHTTP != ""
	lean, toolsetNames := mcpSurface(isHTTP)

	opts, err := ogmcp.ResolveOptions(ogmcp.Options{
		Host:        p.Host,
		Token:       p.Token,
		WebToken:    p.WebToken,
		APIKey:      p.APIKey,
		MultiTenant: mcpMultiTenant,

		ClientOptions: p.ClientOptions(),
	}, lean, toolsetNames)
	if err != nil {
		return err
	}

	if isHTTP {
		mode := "single-tenant"
		if mcpMultiTenant {
			mode = "multi-tenant (per-request header credentials)"
		}
		fmt.Printf("Starting MCP HTTP server on %s [%s, %s]\n", mcpHTTP, mode, surfaceLabel(lean, toolsetNames))
		if lean && mcpMultiTenant {
			fmt.Fprintln(os.Stderr, "Warning: --lean over multi-tenant HTTP exposes og_exec (arbitrary og subcommands) to remote callers. Ensure the per-request credentials are suitably scoped (e.g. a read-only user).")
		}
		return ogmcp.ServeHTTP(mcpHTTP, opts)
	}

	if mcpMultiTenant {
		return fmt.Errorf("--multi-tenant requires --http (stdio is inherently single-user)")
	}
	return ogmcp.ServeStdio(opts)
}

func surfaceLabel(lean bool, toolsetNames []string) string {
	if lean {
		return "lean (og_exec/og_help)"
	}
	if len(toolsetNames) == 0 {
		return "toolsets: observe"
	}
	return "toolsets: " + fmt.Sprintf("%v", toolsetNames)
}
