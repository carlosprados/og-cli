package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// share subcommands for workspaces and dashboards. Kept in a separate file:
// the share lists are REPLACED on every call (empty lists = unshare), which
// is the platform's own semantics for PUT .../share.

var (
	wsShareUsers     []string
	wsShareDomains   []string
	dashShareUsers   []string
	dashShareDomains []string
)

var workspaceShareCmd = &cobra.Command{
	Use:   "share <workspace-id>",
	Short: "Share a workspace with users and/or domains",
	Long: `Share a workspace so other users see it in their UI.

The given lists REPLACE the current sharing (platform semantics): run with the
full list every time, or with no flags to unshare completely.

Examples:
  og workspace share _my_ws --user claudia@amplia.es
  og workspace share _my_ws --user a@x.com --user b@x.com --domain partners
  og workspace share _my_ws --unshare`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runShare(args[0], "workspace", wsShareUsers, wsShareDomains, cmd)
	},
}

var dashboardShareCmd = &cobra.Command{
	Use:   "share <dashboard-id>",
	Short: "Share a dashboard with users and/or domains",
	Long: `Share a single dashboard so other users see it in their UI.

The given lists REPLACE the current sharing. Use --unshare to clear.

Examples:
  og dashboard share <dash-id> --user claudia@amplia.es
  og dashboard share <dash-id> --unshare`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runShare(args[0], "dashboard", dashShareUsers, dashShareDomains, cmd)
	},
}

func runShare(id, kind string, users, domains []string, cmd *cobra.Command) error {
	unshare, _ := cmd.Flags().GetBool("unshare")
	if !unshare && len(users) == 0 && len(domains) == 0 {
		return fmt.Errorf("nothing to share: pass --user/--domain, or --unshare to clear")
	}
	if unshare {
		users, domains = nil, nil
	}

	p, err := activeProfile()
	if err != nil {
		return err
	}
	c := newWebClient(p)

	if kind == "workspace" {
		if _, err := c.ShareWorkspace(id, users, domains); err != nil {
			return err
		}
	} else {
		if _, err := c.ShareDashboard(id, users, domains); err != nil {
			return err
		}
	}

	if unshare {
		fmt.Printf("%s %s unshared.\n", capitalize(kind), id)
		return nil
	}
	fmt.Printf("%s %s shared with users=%v domains=%v\n", capitalize(kind), id, users, domains)
	return nil
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return string(s[0]-32) + s[1:]
}

func init() {
	workspaceShareCmd.Flags().StringArrayVar(&wsShareUsers, "user", nil, "user email to share with (repeatable)")
	workspaceShareCmd.Flags().StringArrayVar(&wsShareDomains, "domain", nil, "domain to share with (repeatable)")
	workspaceShareCmd.Flags().Bool("unshare", false, "clear all sharing")
	workspaceCmd.AddCommand(workspaceShareCmd)

	dashboardShareCmd.Flags().StringArrayVar(&dashShareUsers, "user", nil, "user email to share with (repeatable)")
	dashboardShareCmd.Flags().StringArrayVar(&dashShareDomains, "domain", nil, "domain to share with (repeatable)")
	dashboardShareCmd.Flags().Bool("unshare", false, "clear all sharing")
	dashboardCmd.AddCommand(dashboardShareCmd)
}
