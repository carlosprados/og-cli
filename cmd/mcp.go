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
	mcpStdio   bool
	mcpHTTP    string
)

func init() {
	mcpCmd.Flags().BoolVar(&mcpStdio, "stdio", true, "serve over stdio (default)")
	mcpCmd.Flags().StringVar(&mcpHTTP, "http", "", "serve over HTTP at the given address (e.g. :8080)")
	rootCmd.AddCommand(mcpCmd)
}

func runMCP(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}

	if mcpHTTP != "" {
		fmt.Printf("Starting MCP HTTP server on %s\n", mcpHTTP)
		return ogmcp.ServeHTTP(mcpHTTP, p.Host, p.Token, p.APIKey)
	}

	return ogmcp.ServeStdio(p.Host, p.Token, p.APIKey)
}
