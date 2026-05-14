package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/carlosprados/og-cli/internal/client"
	"github.com/carlosprados/og-cli/internal/output"
	"github.com/carlosprados/og-cli/internal/unwrap"
	"github.com/spf13/cobra"
)

var dashboardCmd = &cobra.Command{
	Use:     "dashboard",
	Aliases: []string{"dash", "dashboards"},
	Short:   "Manage OpenGate dashboards (Web API)",
	Long: `Manage dashboards from the OpenGate Web API.

Every dashboard belongs to exactly one workspace. Use "og workspace list"
to discover workspace IDs and then list dashboards filtered by workspace.`,
}

// --- list ---

var dashboardListCmd = &cobra.Command{
	Use:   "list",
	Short: "List dashboards (optionally filtered by workspace)",
	Long: `List dashboards.

Without flags: iterates all workspaces with ?full=1 and shows their dashboards.
With --workspace <id>: shows dashboards of that workspace only.`,
	RunE: runDashboardList,
}

var dashboardListWorkspace string

type dashboardRow struct {
	WorkspaceID   string
	WorkspaceName string
	DashboardID   string
	Title         string
	Owner         string
}

func runDashboardList(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	c := newWebClient(p)

	var rows []dashboardRow
	if dashboardListWorkspace != "" {
		w, err := c.GetWorkspace(dashboardListWorkspace, true)
		if err != nil {
			return err
		}
		rows = collectDashboardRows(w)
	} else {
		wss, err := c.ListWorkspaces(true)
		if err != nil {
			return err
		}
		for i := range wss {
			rows = append(rows, collectDashboardRows(&wss[i])...)
		}
	}

	return output.Print(outFmt, rows,
		[]string{"Workspace", "Workspace ID", "Dashboard ID", "Title", "Owner"},
		func(data any) [][]string {
			items := data.([]dashboardRow)
			out := make([][]string, len(items))
			for i, r := range items {
				out[i] = []string{r.WorkspaceName, r.WorkspaceID, r.DashboardID, r.Title, r.Owner}
			}
			return out
		},
	)
}

func collectDashboardRows(w *client.Workspace) []dashboardRow {
	rows := make([]dashboardRow, 0, len(w.Dashboards))
	for _, wd := range w.Dashboards {
		if wd.Dashboard == nil {
			rows = append(rows, dashboardRow{
				WorkspaceID:   w.ID,
				WorkspaceName: w.Name,
				DashboardID:   wd.ID,
			})
			continue
		}
		rows = append(rows, dashboardRow{
			WorkspaceID:   w.ID,
			WorkspaceName: w.Name,
			DashboardID:   wd.Dashboard.ID,
			Title:         wd.Dashboard.Title,
			Owner:         wd.Dashboard.Owner,
		})
	}
	return rows
}

// --- get ---

var dashboardGetCmd = &cobra.Command{
	Use:   "get <dashboard-id>",
	Short: "Get a dashboard by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runDashboardGet,
}

func runDashboardGet(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	c := newWebClient(p)

	d, err := c.GetDashboard(args[0])
	if err != nil {
		return err
	}

	return output.Print(outFmt, d,
		[]string{"Field", "Value"},
		func(data any) [][]string {
			d := data.(*client.Dashboard)
			return [][]string{
				{"ID", d.ID},
				{"Title", d.Title},
				{"Owner", d.Owner},
				{"Workspace", d.Workspaces},
				{"Icon", d.Icon},
				{"Widgets", fmt.Sprintf("%d", len(d.Grid))},
				{"LastAccess", d.LastAccess},
			}
		},
	)
}

// --- export ---

var dashboardExportCmd = &cobra.Command{
	Use:   "export <dashboard-id>",
	Short: "Export a dashboard as JSON",
	Long: `Export a dashboard using the dedicated /dashboards/export endpoint.
The resulting JSON can be passed to "og dashboard import" on another tenant.

Output destinations (precedence: --out > --dir > stdout):
  --out file.json          write to that exact path
  --dir backups/           write to backups/<dashboard-id>.json (auto-naming)

Use --full to fetch via GET /dashboards/{id} instead.`,
	Args: cobra.ExactArgs(1),
	RunE: runDashboardExport,
}

var (
	dashboardExportOut  string
	dashboardExportDir  string
	dashboardExportFull bool
)

func runDashboardExport(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	c := newWebClient(p)

	data, err := fetchDashboardExportData(c, args[0], dashboardExportFull)
	if err != nil {
		return err
	}

	dest, err := resolveOutputPath(dashboardExportOut, dashboardExportDir, args[0])
	if err != nil {
		return err
	}

	if dest == "" {
		fmt.Println(string(data))
		return nil
	}
	if err := writeExportFile(dest, data); err != nil {
		return err
	}
	fmt.Printf("Dashboard exported to %s\n", dest)
	return nil
}

func fetchDashboardExportData(c *client.Client, id string, full bool) ([]byte, error) {
	if full {
		d, err := c.GetDashboard(id)
		if err != nil {
			return nil, err
		}
		data, err := json.MarshalIndent(d, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshaling dashboard: %w", err)
		}
		return data, nil
	}
	return c.ExportDashboard(id)
}

// --- export-all ---

var dashboardExportAllCmd = &cobra.Command{
	Use:   "export-all --dir <dir>",
	Short: "Export every dashboard to <dir>/<id>.json",
	Long: `Export every dashboard accessible to the current user, writing one
JSON file per dashboard into the given directory (named <dashboard-id>.json).

Iterates all workspaces (with embedded dashboards) and exports each dashboard
in turn. Use --workspace <id> to limit to dashboards of a single workspace.`,
	RunE: runDashboardExportAll,
}

var (
	dashboardExportAllDir       string
	dashboardExportAllWorkspace string
	dashboardExportAllFull      bool
)

func runDashboardExportAll(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	c := newWebClient(p)

	if err := os.MkdirAll(dashboardExportAllDir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dashboardExportAllDir, err)
	}

	var rows []dashboardRow
	if dashboardExportAllWorkspace != "" {
		w, err := c.GetWorkspace(dashboardExportAllWorkspace, true)
		if err != nil {
			return err
		}
		rows = collectDashboardRows(w)
	} else {
		wss, err := c.ListWorkspaces(true)
		if err != nil {
			return err
		}
		for i := range wss {
			rows = append(rows, collectDashboardRows(&wss[i])...)
		}
	}

	var exported, failed int
	for _, r := range rows {
		if r.DashboardID == "" {
			continue
		}
		data, err := fetchDashboardExportData(c, r.DashboardID, dashboardExportAllFull)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ %s: %v\n", r.DashboardID, err)
			failed++
			continue
		}
		dest := filepath.Join(dashboardExportAllDir, r.DashboardID+".json")
		if err := writeExportFile(dest, data); err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ %s: %v\n", r.DashboardID, err)
			failed++
			continue
		}
		fmt.Printf("  ✓ %s (%s) → %s\n", r.DashboardID, r.Title, dest)
		exported++
	}

	fmt.Printf("\n%d dashboard(s) exported, %d failed.\n", exported, failed)
	if failed > 0 {
		return fmt.Errorf("%d dashboard(s) failed to export", failed)
	}
	return nil
}

// --- unwrap / pull ---

var dashboardUnwrapCmd = &cobra.Command{
	Use:     "unwrap <dashboard-id>",
	Aliases: []string{"pull"},
	Short:   "Unwrap a single dashboard into an editable directory tree (alias: pull)",
	Long: `Unwrap a single dashboard into a directory tree designed for inspection
and edition with an IDE (or by an AI agent).

Structure produced:

  <dir>/<dashboard-slug>/
    dashboard.json                   # dashboard metadata (grid stripped)
    <NN>__<type>__<wid>/             # one folder per widget, NN preserves order
      widget.json                    # grid item, JS fields removed from config
      formatter.js                   # if the widget had a "formatter" field
      columns__0__formatter.js       # nested fields keep the keypath in the name

Use "og dashboard wrap" or "og dashboard deploy" to rebuild and re-upload.`,
	Args: cobra.ExactArgs(1),
	RunE: runDashboardUnwrap,
}

var (
	dashboardUnwrapDir   string
	dashboardUnwrapForce bool
)

func runDashboardUnwrap(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	c := newWebClient(p)

	d, err := c.GetDashboard(args[0])
	if err != nil {
		return err
	}

	taken := make(map[string]bool)
	slug := unwrap.DedupedSlug(d.Title, d.ID, taken)
	dashDir := filepath.Join(dashboardUnwrapDir, slug)

	if err := prepareUnwrapTarget(dashDir, dashboardUnwrapForce); err != nil {
		return err
	}

	if err := unwrap.UnwrapDashboardFull(d, nil, dashDir); err != nil {
		return err
	}
	fmt.Printf("  ✓ dashboard %s (%d widgets) → %s\n", d.ID, len(d.Grid), dashDir)
	return nil
}

// --- wrap ---

var dashboardWrapCmd = &cobra.Command{
	Use:   "wrap <dashboard-dir>",
	Short: "Rebuild a dashboard JSON from an unwrapped directory tree",
	Long: `Rebuild a dashboard JSON from a directory previously produced by
"og dashboard unwrap" (or by "og workspace unwrap" — pass one of the
NN__<slug> subfolders).

By default, the rebuilt JSON is written to stdout. Use --out to write to a file.`,
	Args: cobra.ExactArgs(1),
	RunE: runDashboardWrap,
}

var dashboardWrapOut string

func runDashboardWrap(cmd *cobra.Command, args []string) error {
	if err := assertDashboardDir(args[0]); err != nil {
		return err
	}

	full, _, err := unwrap.WrapDashboard(args[0])
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(full, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling dashboard: %w", err)
	}
	if dashboardWrapOut == "" {
		fmt.Println(string(data))
		return nil
	}
	if err := os.WriteFile(dashboardWrapOut, data, 0o644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}
	fmt.Printf("Dashboard JSON written to %s\n", dashboardWrapOut)
	return nil
}

// --- deploy ---

var dashboardDeployCmd = &cobra.Command{
	Use:   "deploy <dashboard-dir>",
	Short: "Wrap + import a single dashboard in one step",
	Long: `Deploy an unwrapped dashboard directory (one of the NN__<slug> folders
inside a workspace unwrap) to OpenGate in a single step.

Without --update: POST /api/dashboards (creates a new dashboard).
With --update:    PUT  /api/dashboards/{id} (overwrites in place).

Use --workspace <id> to override the target workspace ID in the payload
(useful for migrating a dashboard between tenants). Ignored with --update.`,
	Args: cobra.ExactArgs(1),
	RunE: runDashboardDeploy,
}

var (
	dashboardDeployUpdate    bool
	dashboardDeployWorkspace string
)

func runDashboardDeploy(cmd *cobra.Command, args []string) error {
	dir := args[0]
	if err := assertDashboardDir(dir); err != nil {
		return err
	}

	p, err := activeProfile()
	if err != nil {
		return err
	}

	full, _, err := unwrap.WrapDashboard(dir)
	if err != nil {
		return fmt.Errorf("wrapping %s: %w", dir, err)
	}
	if full.ID == "" {
		return fmt.Errorf("dashboard.json in %s has no _id", dir)
	}

	body, err := json.Marshal(full)
	if err != nil {
		return fmt.Errorf("marshaling dashboard: %w", err)
	}

	c := newWebClient(p)

	if dashboardDeployUpdate {
		if err := c.UpdateDashboard(full.ID, body); err != nil {
			return err
		}
		fmt.Printf("Dashboard %s deployed (%d widget(s) updated).\n", full.ID, len(full.Grid))
		return nil
	}

	if _, err := c.CreateDashboard(body, dashboardDeployWorkspace); err != nil {
		if isDuplicateKeyError(err) {
			return fmt.Errorf("%w\n\nThe dashboard _id already exists. Re-run with --update to overwrite it via PUT", err)
		}
		return err
	}
	fmt.Printf("Dashboard %s deployed (%d widget(s) created).\n", full.ID, len(full.Grid))
	return nil
}

func assertDashboardDir(dir string) error {
	if _, err := os.Stat(filepath.Join(dir, "dashboard.json")); err == nil {
		return nil
	}
	if _, err := os.Stat(filepath.Join(dir, "workspace.json")); err == nil {
		return fmt.Errorf("%s looks like a workspace directory — use `og workspace deploy` instead", dir)
	}
	return fmt.Errorf("%s does not contain dashboard.json — not a dashboard unwrap directory", dir)
}

// --- import ---

var dashboardImportCmd = &cobra.Command{
	Use:   "import -f <file.json> [--workspace <id>] [--update]",
	Short: "Import (create or update) a dashboard from a JSON file",
	Long: `Import a dashboard from a JSON file.

By default, the file is POSTed as a new dashboard. If the JSON contains an
"_id" that already exists, OpenGate returns HTTP 400 with a duplicate-key
error; re-run with --update to overwrite the existing dashboard via PUT.

If --workspace is provided, the "workspaces" field of the payload is overridden
with the given workspace ID — useful for migrating a dashboard from one tenant
or workspace to another. --workspace is ignored when combined with --update.`,
	RunE: runDashboardImport,
}

var (
	dashboardImportFile      string
	dashboardImportWorkspace string
	dashboardImportUpdate    bool
)

func runDashboardImport(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}

	body, err := os.ReadFile(dashboardImportFile)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	c := newWebClient(p)

	if dashboardImportUpdate {
		id, err := extractIDFromJSON(body)
		if err != nil {
			return fmt.Errorf("--update requires the file to contain an _id: %w", err)
		}
		if err := c.UpdateDashboard(id, body); err != nil {
			return err
		}
		fmt.Printf("Dashboard %s updated successfully.\n", id)
		return nil
	}

	resp, err := c.CreateDashboard(body, dashboardImportWorkspace)
	if err != nil {
		if isDuplicateKeyError(err) {
			return fmt.Errorf("%w\n\nThe dashboard _id already exists. Re-run with --update to overwrite it via PUT", err)
		}
		return err
	}

	if len(resp) > 0 && outFmt == output.FormatJSON {
		fmt.Println(string(resp))
		return nil
	}
	fmt.Println("Dashboard imported successfully.")
	return nil
}

// --- update ---

var dashboardUpdateCmd = &cobra.Command{
	Use:   "update <dashboard-id> -f <file.json>",
	Short: "Update an existing dashboard from a JSON file",
	Args:  cobra.ExactArgs(1),
	RunE:  runDashboardUpdate,
}

var dashboardUpdateFile string

func runDashboardUpdate(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}

	body, err := os.ReadFile(dashboardUpdateFile)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	c := newWebClient(p)
	if err := c.UpdateDashboard(args[0], body); err != nil {
		return err
	}

	fmt.Println("Dashboard updated successfully.")
	return nil
}

// --- delete ---

var dashboardDeleteCmd = &cobra.Command{
	Use:   "delete <dashboard-id>",
	Short: "Delete a dashboard",
	Args:  cobra.ExactArgs(1),
	RunE:  runDashboardDelete,
}

func runDashboardDelete(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}

	c := newWebClient(p)
	if err := c.DeleteDashboard(args[0]); err != nil {
		return err
	}

	fmt.Println("Dashboard deleted successfully.")
	return nil
}

// --- init ---

func init() {
	dashboardListCmd.Flags().StringVar(&dashboardListWorkspace, "workspace", "", "filter by workspace ID")

	dashboardExportCmd.Flags().StringVar(&dashboardExportOut, "out", "", "write export JSON to this file (default: stdout)")
	dashboardExportCmd.Flags().StringVar(&dashboardExportDir, "dir", "", "write export to <dir>/<dashboard-id>.json (auto-naming)")
	dashboardExportCmd.Flags().BoolVar(&dashboardExportFull, "full", false, "use GET /dashboards/{id} instead of /dashboards/export/{id}")

	dashboardImportCmd.Flags().StringVarP(&dashboardImportFile, "file", "f", "", "path to JSON file with dashboard definition")
	dashboardImportCmd.Flags().StringVar(&dashboardImportWorkspace, "workspace", "", "override the target workspace ID in the payload (ignored with --update)")
	dashboardImportCmd.Flags().BoolVar(&dashboardImportUpdate, "update", false, "update an existing dashboard (PUT) instead of creating (POST)")
	_ = dashboardImportCmd.MarkFlagRequired("file")

	dashboardUpdateCmd.Flags().StringVarP(&dashboardUpdateFile, "file", "f", "", "path to JSON file with dashboard definition")
	_ = dashboardUpdateCmd.MarkFlagRequired("file")

	dashboardDeployCmd.Flags().BoolVar(&dashboardDeployUpdate, "update", false, "update an existing dashboard (PUT) instead of creating (POST)")
	dashboardDeployCmd.Flags().StringVar(&dashboardDeployWorkspace, "workspace", "", "override the target workspace ID in the payload (ignored with --update)")

	dashboardUnwrapCmd.Flags().StringVar(&dashboardUnwrapDir, "dir", "", "destination directory (required)")
	dashboardUnwrapCmd.Flags().BoolVar(&dashboardUnwrapForce, "force", false, "overwrite destination if it already exists")
	_ = dashboardUnwrapCmd.MarkFlagRequired("dir")

	dashboardWrapCmd.Flags().StringVar(&dashboardWrapOut, "out", "", "write rebuilt JSON to this file (default: stdout)")

	dashboardExportAllCmd.Flags().StringVar(&dashboardExportAllDir, "dir", "", "destination directory (required)")
	dashboardExportAllCmd.Flags().StringVar(&dashboardExportAllWorkspace, "workspace", "", "limit to dashboards of this workspace")
	dashboardExportAllCmd.Flags().BoolVar(&dashboardExportAllFull, "full", false, "use GET /dashboards/{id} instead of /dashboards/export/{id}")
	_ = dashboardExportAllCmd.MarkFlagRequired("dir")

	dashboardCmd.AddCommand(dashboardListCmd)
	dashboardCmd.AddCommand(dashboardGetCmd)
	dashboardCmd.AddCommand(dashboardExportCmd)
	dashboardCmd.AddCommand(dashboardExportAllCmd)
	dashboardCmd.AddCommand(dashboardImportCmd)
	dashboardCmd.AddCommand(dashboardUpdateCmd)
	dashboardCmd.AddCommand(dashboardDeleteCmd)
	dashboardCmd.AddCommand(dashboardDeployCmd)
	dashboardCmd.AddCommand(dashboardUnwrapCmd)
	dashboardCmd.AddCommand(dashboardWrapCmd)

	rootCmd.AddCommand(dashboardCmd)
}
