package cmd

import (
	"fmt"

	ogmcp "github.com/carlosprados/og-cli/internal/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the MCP (Model Context Protocol) server",
	Long:  "Starts an MCP server that exposes OpenGate API operations as LLM tools.",
	RunE:  runMCP,
}

var (
	mcpStdio       bool
	mcpHTTP        string
	mcpMultiTenant bool
)

func init() {
	mcpCmd.Flags().BoolVar(&mcpStdio, "stdio", true, "serve over stdio (default)")
	mcpCmd.Flags().StringVar(&mcpHTTP, "http", "", "serve over HTTP at the given address (e.g. :8080)")
	mcpCmd.Flags().BoolVar(&mcpMultiTenant, "multi-tenant", false,
		"HTTP only: take credentials per request from headers (Authorization: Bearer, X-OG-Web-Token, X-OG-Api-Key) instead of the startup profile. Requires TLS in front. Drops the login tool.")
	rootCmd.AddCommand(mcpCmd)
}

func runMCP(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}

	if mcpHTTP != "" {
		mode := "single-tenant"
		if mcpMultiTenant {
			mode = "multi-tenant (per-request header credentials)"
		}
		fmt.Printf("Starting MCP HTTP server on %s [%s]\n", mcpHTTP, mode)
		return ogmcp.ServeHTTP(mcpHTTP, p.Host, p.Token, p.WebToken, p.APIKey, mcpMultiTenant)
	}

	if mcpMultiTenant {
		return fmt.Errorf("--multi-tenant requires --http (stdio is inherently single-user)")
	}
	return ogmcp.ServeStdio(p.Host, p.Token, p.WebToken, p.APIKey)
}
