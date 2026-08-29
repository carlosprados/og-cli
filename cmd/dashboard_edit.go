package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/carlosprados/og-cli/v2/internal/canon"
	"github.com/carlosprados/og-cli/v2/internal/config"
	"github.com/carlosprados/og-cli/v2/internal/diff"
	"github.com/carlosprados/og-cli/v2/internal/output"
	"github.com/carlosprados/og-cli/v2/internal/unwrap"
	"github.com/carlosprados/og-cli/v2/pkg/opengate"
	"github.com/spf13/cobra"
)

// The edit half of the dashboard family: `show` for the remote side of one
// file, `diff` for what deploying the directory would change.
//
// Both existed for rules, connector functions and provision functions and for a
// whole workspace, and for nothing in between. That gap was not neutral: og
// routes an unknown subcommand through cobra, which prints the family's help
// and exits 0, so a caller asking `og dashboard diff` got a help page that
// looked like a successful comparison. An editor confirming a deploy against
// that is confirming against nothing.
//
// A widget deliberately gets neither. It is not addressable on the platform on
// its own — it is a grid item of a dashboard — and the smallest unit that can
// be fetched or deployed is the dashboard it sits in. That is the same boundary
// `og workspace watch` already draws when a widget edit resolves to its
// dashboard.

var dashboardShowCmd = newShowCmdSpec(showSpec{
	Kind:  unwrap.KindDashboard,
	Use:   "show <dashboard-id>",
	Short: "Print a dashboard's remote widget code files",
	Examples: []string{
		"og dashboard show multisensor-overview",
		"og dashboard show multisensor-overview --path 01__fulldeviceslist__w-3/columns__2___formatterCode.js",
	},
	Connect: webConnect,
	Files: func(ctx context.Context, c *opengate.Client, _ string, id string) (map[string]string, error) {
		d, err := c.GetDashboard(ctx, id)
		if err != nil {
			return nil, err
		}
		return unwrap.DashboardCodeFiles(d)
	},
	// A dashboard's paths carry a grid index that the platform can change under
	// a local tree, so they are matched by widget identity.
	Resolve: unwrap.ResolveCodePath,
})

// webConnect is the connection workspaces and dashboards need: the Web API,
// which carries its own token, and no organization — they are not scoped by one.
func webConnect(p *config.Profile) (*opengate.Client, string, error) {
	return newWebClient(p), "", nil
}

var dashboardDiffCmd = &cobra.Command{
	Use:     "diff <dashboard-dir>",
	Aliases: []string{"status"},
	Short:   "Compare a local dashboard directory against the platform",
	Long: `Compare an unwrapped dashboard against the one on the platform.

Reads remote → local, so the report is what deploying would do: '+' is something
the deploy would create, '−' something it would delete, '~' something it would
change. Widgets are matched by identity, so moving one is reported as a move
rather than as two rewrites, and unchanged branches are pruned.

Widget JavaScript is compared as a textual diff of the extracted files, the same
ones the editor works on; everything else as a structural diff of the metadata.

This is ` + "`og workspace diff`" + ` narrowed to one dashboard: use it when the
dashboard is what you edited, and the workspace one when you want the whole tree.

  og dashboard diff ws/multisensor-demo/00__multisensor-overview
  og dashboard diff ws/... --name-only
  og dashboard diff ws/... --against prod    # promotion: what differs between tenants?
  og dashboard diff ws/... --exit-code       # CI drift gate

With --exit-code: 1 when there are differences, 0 when there are none, 2 on
failure.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := args[0]
		if err := assertDashboardDir(dir); err != nil {
			return err
		}

		localDash, _, err := unwrap.WrapDashboard(dir, hintWarner())
		if err != nil {
			return err
		}
		if localDash.ID == "" {
			return fmt.Errorf("%s has no _id in dashboard.json — pull it first, or there is nothing remote to compare against",
				filepath.Base(dir))
		}

		// --against compares the same dashboard in another tenant, which is the
		// promotion question rather than the drift one.
		profileName := profile
		scope := canon.SameTenant
		if diffAgainst != "" {
			profileName = diffAgainst
			scope = canon.CrossTenant
		}

		p, err := cfg.ActiveProfile(profileName)
		if err != nil {
			return err
		}
		c := newWebClient(p)

		remoteDash, err := c.GetDashboard(cmd.Context(), localDash.ID)
		if err != nil {
			return err
		}

		// Both sides go through dashboardSide, the same builder the workspace
		// diff uses. That symmetry is what makes the comparison trustworthy:
		// any asymmetry would have to come from the platform rather than from
		// two different readings of the same thing.
		localSide, err := dashboardSide(localDash)
		if err != nil {
			return err
		}
		remoteSide, err := dashboardSide(remoteDash)
		if err != nil {
			return err
		}

		result, err := diff.CompareTree(dir, remoteSide, localSide,
			diff.Options{Scope: scope, ContextLines: diffContext})
		if err != nil {
			return err
		}
		if diffAgainst != "" {
			result.Origin = fmt.Sprintf("compared against profile %s", diffAgainst)
		}

		if outFmt == output.FormatJSON {
			if err := output.PrintEnvelope(os.Stdout, "diff", result); err != nil {
				return err
			}
		} else if !result.Changed() {
			fmt.Println("No differences.")
		} else {
			fmt.Print(result.RenderText(diffNameOnly))
		}

		if diffExitCode && result.Changed() {
			return diffFound()
		}
		return nil
	},
}

func init() {
	addShowFlags(dashboardShowCmd)
	addDiffFlags(dashboardDiffCmd)
	dashboardCmd.AddCommand(dashboardShowCmd)
	dashboardCmd.AddCommand(dashboardDiffCmd)
}
