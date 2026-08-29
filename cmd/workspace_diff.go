package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/carlosprados/og-cli/v2/internal/canon"
	"github.com/carlosprados/og-cli/v2/internal/diff"
	"github.com/carlosprados/og-cli/v2/internal/output"
	"github.com/carlosprados/og-cli/v2/internal/unwrap"
	"github.com/carlosprados/og-cli/v2/pkg/opengate"
	"github.com/spf13/cobra"
)

// `og workspace diff` is the one family whose diff could not reuse newDiffCmd.
//
// Not because the engine is unsuited — it is family-agnostic and a workspace
// payload canonicalizes like any other — but because a workspace is a tree.
// Flattening it produces paths like
// `dashboards[0].dashboard.grid[2].definition.config.columns[4]._formatterCode`,
// which is technically the answer and practically unreadable. So both sides are
// decomposed into the same shape the directory tree already has — workspace,
// dashboard, widget — and compared level by level.
//
// Both sides go through the SAME builders below. That symmetry is what makes
// the comparison trustworthy: the local tree is wrapped back into the client's
// own types first, so any asymmetry would have to come from the platform rather
// than from two different readings of the same thing.

var workspaceDiffCmd = &cobra.Command{
	Use:     "diff <workspace-dir>",
	Aliases: []string{"status"},
	Short:   "Compare a local workspace tree against the platform",
	Long: `Compare an unwrapped workspace against the one on the platform.

Reads remote → local, so the report is what deploying would do: '+' is something
the deploy would create, '−' something it would delete, '~' something it would
change. Dashboards and widgets are matched by identity, so moving a widget is
reported as a move rather than as two rewrites, and unchanged branches are
pruned — what remains is the path to what actually differs.

Widget JavaScript is compared as a textual diff of the extracted files, the same
ones the editor works on; everything else as a structural diff of the metadata.

  og workspace diff ws/multisensor-demo
  og workspace diff ws/multisensor-demo --name-only
  og workspace diff ws/multisensor-demo --against prod    # promotion: what differs between tenants?
  og workspace diff ws/multisensor-demo --exit-code       # CI drift gate

With --exit-code: 1 when there are differences, 0 when there are none, 2 on
failure.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := args[0]

		localWS, err := unwrap.Wrap(dir, hintWarner())
		if err != nil {
			return err
		}
		if localWS.ID == "" {
			return fmt.Errorf("%s has no _id in workspace.json — pull it first, or there is nothing remote to compare against",
				filepath.Base(dir))
		}

		// --against compares the same workspace in another tenant, which is the
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
		// Workspaces and dashboards go through the Web API, which carries its own
		// token — the North one is not enough.
		c := newWebClient(p)

		remoteWS, err := c.GetWorkspace(cmd.Context(), localWS.ID, true)
		if err != nil {
			return err
		}

		localSide, err := localWorkspaceSide(dir, localWS)
		if err != nil {
			return err
		}
		remoteSide, err := remoteWorkspaceSide(cmd.Context(), c, remoteWS)
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

// localWorkspaceSide builds the local half of the tree.
//
// unwrap.Wrap gives the workspace with its dashboards as layout entries only,
// so each dashboard directory is wrapped again for its grid — the same call
// `og workspace deploy` makes, which is what keeps the diff honest about what
// a deploy would send.
func localWorkspaceSide(dir string, ws *opengate.Workspace) (diff.Side, error) {
	side := diff.Side{Kind: "workspace", Name: ws.Name, ID: ws.ID}

	meta, err := workspaceMeta(ws)
	if err != nil {
		return side, err
	}
	side.Meta = meta

	names, err := dashboardDirs(dir)
	if err != nil {
		return side, err
	}
	for _, name := range names {
		dash, _, err := unwrap.WrapDashboard(filepath.Join(dir, name), hintWarner())
		if err != nil {
			return side, fmt.Errorf("dashboard %s: %w", name, err)
		}
		child, err := dashboardSide(dash)
		if err != nil {
			return side, fmt.Errorf("dashboard %s: %w", name, err)
		}
		side.Children = append(side.Children, child)
	}
	return side, nil
}

// remoteWorkspaceSide builds the remote half, fetching each dashboard for its
// grid — the workspace response carries only a simplified reference.
func remoteWorkspaceSide(ctx context.Context, c *opengate.Client, ws *opengate.Workspace) (diff.Side, error) {
	side := diff.Side{Kind: "workspace", Name: ws.Name, ID: ws.ID}

	meta, err := workspaceMeta(ws)
	if err != nil {
		return side, err
	}
	side.Meta = meta

	for _, wd := range ws.Dashboards {
		if wd.Dashboard == nil {
			continue
		}
		full, err := c.GetDashboard(ctx, wd.Dashboard.ID)
		if err != nil {
			return side, fmt.Errorf("fetching dashboard %s: %w", wd.Dashboard.ID, err)
		}
		child, err := dashboardSide(full)
		if err != nil {
			return side, fmt.Errorf("dashboard %s: %w", wd.Dashboard.ID, err)
		}
		side.Children = append(side.Children, child)
	}
	return side, nil
}

// workspaceMeta is the workspace with its dashboards removed: they are compared
// as children, and leaving them in would report every nested change twice.
func workspaceMeta(ws *opengate.Workspace) (json.RawMessage, error) {
	clone := *ws
	clone.Dashboards = nil
	return json.Marshal(clone)
}

// dashboardSide turns one dashboard into a node with its widgets beneath it.
func dashboardSide(d *opengate.Dashboard) (diff.Side, error) {
	side := diff.Side{Kind: "dashboard", Name: d.Title, ID: d.ID}

	clone := *d
	clone.Grid = nil
	meta, err := json.Marshal(clone)
	if err != nil {
		return side, err
	}
	side.Meta = meta

	for i, item := range d.Grid {
		w, err := widgetSide(i, item)
		if err != nil {
			return side, err
		}
		side.Children = append(side.Children, w)
	}
	return side, nil
}

// widgetSide splits one grid item into its metadata and its extracted code.
//
// The extraction is unwrap's, not a second implementation of it: the files this
// produces are the ones on disk, so a code diff is a diff of what the developer
// edits rather than of a 1500-character JSON string.
func widgetSide(index int, item opengate.GridItem) (diff.Side, error) {
	side := diff.Side{Kind: "widget", ID: item.I, Name: widgetLabel(index, item)}

	var configTree any
	if item.Definition != nil && len(item.Definition.Config) > 0 {
		if err := json.Unmarshal(item.Definition.Config, &configTree); err != nil {
			return side, fmt.Errorf("widget %s: decoding config: %w", item.I, err)
		}
	}
	cleaned, files, _ := unwrap.WidgetContract().Extract(configTree, nil)
	side.Code = files

	clone := item
	if item.Definition != nil {
		defClone := *item.Definition
		raw, err := json.Marshal(cleaned)
		if err != nil {
			return side, err
		}
		defClone.Config = raw
		clone.Definition = &defClone
	}
	meta, err := json.Marshal(clone)
	if err != nil {
		return side, err
	}
	side.Meta = meta
	return side, nil
}

// widgetLabel names a widget the way `og workspace pull` names its directory:
// grid position, widget type, widget id. The position is the one in the grid
// being reported, so on a reordered dashboard the label shows where the widget
// ends up rather than the folder it currently sits in — which is the question a
// diff is answering.
func widgetLabel(index int, item opengate.GridItem) string {
	kind := "widget"
	if item.Definition != nil && item.Definition.Type != "" {
		kind = item.Definition.Type
	}
	id := item.I
	if id == "" && item.Definition != nil {
		id = item.Definition.Wid
	}
	if id == "" {
		return fmt.Sprintf("%02d__%s", index, kind)
	}
	return fmt.Sprintf("%02d__%s__%s", index, kind, id)
}

// dashboardDirs lists a workspace directory's dashboard folders, in order.
// Dot-directories are metadata — the .og/ sync cache, .git, editor state.
func dashboardDirs(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("listing %s: %w", dir, err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names, nil
}

func init() {
	addDiffFlags(workspaceDiffCmd)
	workspaceCmd.AddCommand(workspaceDiffCmd)
}
