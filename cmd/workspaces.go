package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/carlosprados/og-cli/internal/client"
	"github.com/carlosprados/og-cli/internal/output"
	"github.com/carlosprados/og-cli/internal/unwrap"
	"github.com/spf13/cobra"
)

var workspaceCmd = &cobra.Command{
	Use:     "workspace",
	Aliases: []string{"ws", "workspaces"},
	Short:   "Manage OpenGate workspaces (Web API)",
	Long: `Manage workspaces from the OpenGate Web API.

Workspaces are the top-level container in the UI configuration hierarchy.
Each workspace owns one or more dashboards.`,
}

// --- list ---

var workspaceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List workspaces",
	RunE:  runWorkspaceList,
}

var workspaceListFull bool

func runWorkspaceList(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	c := newWebClient(p)

	wss, err := c.ListWorkspaces(workspaceListFull)
	if err != nil {
		return err
	}

	return output.Print(outFmt, wss,
		[]string{"ID", "Name", "Owner", "Dashboards", "Domains"},
		func(data any) [][]string {
			items := data.([]client.Workspace)
			rows := make([][]string, len(items))
			for i, w := range items {
				rows[i] = []string{
					w.ID,
					w.Name,
					w.Owner,
					fmt.Sprintf("%d", len(w.Dashboards)),
					fmt.Sprintf("%d", len(w.Domains)),
				}
			}
			return rows
		},
	)
}

// --- get ---

var workspaceGetCmd = &cobra.Command{
	Use:   "get <workspace-id>",
	Short: "Get a workspace by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkspaceGet,
}

var workspaceGetFull bool

func runWorkspaceGet(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	c := newWebClient(p)

	w, err := c.GetWorkspace(args[0], workspaceGetFull)
	if err != nil {
		return err
	}

	return output.Print(outFmt, w,
		[]string{"Field", "Value"},
		func(data any) [][]string {
			w := data.(*client.Workspace)
			return [][]string{
				{"ID", w.ID},
				{"Name", w.Name},
				{"Owner", w.Owner},
				{"Icon", w.Icon},
				{"Color", w.Color},
				{"Dashboards", fmt.Sprintf("%d", len(w.Dashboards))},
				{"Domains", fmt.Sprintf("%d", len(w.Domains))},
				{"Users", fmt.Sprintf("%d", len(w.Users))},
				{"Actions", fmt.Sprintf("%d", len(w.Actions))},
				{"LastAccess", w.LastAccess},
			}
		},
	)
}

// --- export ---

var workspaceExportCmd = &cobra.Command{
	Use:   "export <workspace-id>",
	Short: "Export a workspace as JSON",
	Long: `Export a workspace using the dedicated /workspaces/export endpoint.
The resulting JSON can be passed to "og workspace import" on another tenant.

Output destinations (precedence: --out > --dir > stdout):
  --out file.json          write to that exact path
  --dir backups/           write to backups/<workspace-id>.json (auto-naming)

Use --full to fetch with dashboards inlined instead (GET /workspaces/{id}?full=1).`,
	Args: cobra.ExactArgs(1),
	RunE: runWorkspaceExport,
}

var (
	workspaceExportOut  string
	workspaceExportDir  string
	workspaceExportFull bool
)

func runWorkspaceExport(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	c := newWebClient(p)

	data, err := fetchWorkspaceExportData(c, args[0], workspaceExportFull)
	if err != nil {
		return err
	}

	dest, err := resolveOutputPath(workspaceExportOut, workspaceExportDir, args[0])
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
	fmt.Printf("Workspace exported to %s\n", dest)
	return nil
}

func fetchWorkspaceExportData(c *client.Client, id string, full bool) ([]byte, error) {
	if full {
		w, err := c.GetWorkspace(id, true)
		if err != nil {
			return nil, err
		}
		data, err := json.MarshalIndent(w, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshaling workspace: %w", err)
		}
		return data, nil
	}
	return c.ExportWorkspace(id)
}

// resolveOutputPath picks the destination file given --out and --dir.
// Returns "" when output should go to stdout.
func resolveOutputPath(outFlag, dirFlag, id string) (string, error) {
	if outFlag != "" {
		return outFlag, nil
	}
	if dirFlag != "" {
		if err := os.MkdirAll(dirFlag, 0o755); err != nil {
			return "", fmt.Errorf("creating directory %s: %w", dirFlag, err)
		}
		return filepath.Join(dirFlag, id+".json"), nil
	}
	return "", nil
}

func writeExportFile(dest string, data []byte) error {
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return fmt.Errorf("writing file %s: %w", dest, err)
	}
	return nil
}

// --- export-all ---

var workspaceExportAllCmd = &cobra.Command{
	Use:   "export-all --dir <dir>",
	Short: "Export every workspace to <dir>/<id>.json",
	Long: `Export every workspace accessible to the current user, writing one
JSON file per workspace into the given directory (named <workspace-id>.json).

Useful for full UI backups before tenant migrations.`,
	RunE: runWorkspaceExportAll,
}

var (
	workspaceExportAllDir  string
	workspaceExportAllFull bool
)

func runWorkspaceExportAll(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	c := newWebClient(p)

	if err := os.MkdirAll(workspaceExportAllDir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", workspaceExportAllDir, err)
	}

	wss, err := c.ListWorkspaces(false)
	if err != nil {
		return err
	}

	var exported, failed int
	for _, w := range wss {
		data, err := fetchWorkspaceExportData(c, w.ID, workspaceExportAllFull)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ %s: %v\n", w.ID, err)
			failed++
			continue
		}
		dest := filepath.Join(workspaceExportAllDir, w.ID+".json")
		if err := writeExportFile(dest, data); err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ %s: %v\n", w.ID, err)
			failed++
			continue
		}
		fmt.Printf("  ✓ %s → %s\n", w.ID, dest)
		exported++
	}

	fmt.Printf("\n%d workspace(s) exported, %d failed.\n", exported, failed)
	if failed > 0 {
		return fmt.Errorf("%d workspace(s) failed to export", failed)
	}
	return nil
}

// --- deploy ---

var workspaceDeployCmd = &cobra.Command{
	Use:   "deploy <workspace-dir>",
	Short: "Wrap + import in one step (skip the intermediate JSON file)",
	Long: `Deploy an unwrapped workspace directory to OpenGate in a single step.

Equivalent to:
  og workspace wrap <dir> --out tmp.json
  og workspace import -f tmp.json [--update]

Without --update, performs a fresh create (POST workspace shell + POST each
dashboard + PUT workspace with layout refs). With --update, replays the same
multi-phase flow as PUTs against the existing IDs.`,
	Args: cobra.ExactArgs(1),
	RunE: runWorkspaceDeploy,
}

var workspaceDeployUpdate bool

func runWorkspaceDeploy(cmd *cobra.Command, args []string) error {
	dir := args[0]
	if err := assertWorkspaceDir(dir); err != nil {
		return err
	}

	p, err := activeProfile()
	if err != nil {
		return err
	}

	w, err := unwrap.Wrap(dir)
	if err != nil {
		return fmt.Errorf("wrapping %s: %w", dir, err)
	}
	if w.ID == "" {
		return fmt.Errorf("workspace.json in %s has no _id", dir)
	}

	c := newWebClient(p)

	if workspaceDeployUpdate {
		if err := c.UpdateWorkspaceDeep(w); err != nil {
			return err
		}
		fmt.Printf("Workspace %s deployed (updated workspace + %d dashboard(s)).\n", w.ID, countEmbeddedDashboards(w))
		return nil
	}

	if err := c.ImportWorkspaceDeep(w); err != nil {
		if isDuplicateKeyError(err) {
			return fmt.Errorf("%w\n\nThe workspace _id already exists. Re-run with --update to overwrite it (and its dashboards) via PUT", err)
		}
		return err
	}
	fmt.Printf("Workspace %s deployed (created workspace + %d dashboard(s)).\n", w.ID, countEmbeddedDashboards(w))
	return nil
}

// assertWorkspaceDir verifies dir looks like a workspace unwrap directory
// (contains workspace.json). When the user accidentally passes a dashboard
// directory, surface a clear hint pointing at og dashboard deploy.
func assertWorkspaceDir(dir string) error {
	if _, err := os.Stat(filepath.Join(dir, "workspace.json")); err == nil {
		return nil
	}
	if _, err := os.Stat(filepath.Join(dir, "dashboard.json")); err == nil {
		return fmt.Errorf("%s looks like a dashboard directory — use `og dashboard deploy` instead", dir)
	}
	return fmt.Errorf("%s does not contain workspace.json — not a workspace unwrap directory", dir)
}

// --- unwrap ---

// prepareUnwrapTarget makes sure the destination workspace dir exists and is
// in a known state. When force is true, the existing dir is wiped first; when
// false and the dir is non-empty, an error is returned to avoid mixing stale
// folders with fresh ones (which produces ghost dashboards in the wrap).
func prepareUnwrapTarget(dir string, force bool) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading destination: %w", err)
	}
	if len(entries) == 0 {
		return nil
	}
	if force {
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("removing existing destination: %w", err)
		}
		return nil
	}
	return fmt.Errorf("destination %s already exists and is not empty (use --force to overwrite)", dir)
}

var workspaceUnwrapCmd = &cobra.Command{
	Use:   "unwrap <workspace-id>",
	Short: "Unwrap a workspace into an editable directory tree",
	Long: `Unwrap a workspace into a directory tree designed for inspection
and edition with an IDE (or by an AI agent).

Structure produced:

  <dir>/<workspace-slug>/
    workspace.json                   # workspace metadata (dashboards stripped)
    <dashboard-slug>/
      dashboard.json                 # dashboard metadata (grid stripped)
      <NN>__<type>__<wid>/           # one folder per widget, NN preserves order
        widget.json                  # grid item, JS fields removed from config
        formatter.js                 # if the widget had a "formatter" field
        columns__0__formatter.js     # nested fields keep the keypath in the name

JavaScript code is extracted by name (formatter, script, operation, code, fn,
expression) or by a content heuristic. Use "og workspace wrap" to rebuild a
single workspace JSON ready for import.`,
	Args: cobra.ExactArgs(1),
	RunE: runWorkspaceUnwrap,
}

var (
	workspaceUnwrapDir   string
	workspaceUnwrapForce bool
)

func runWorkspaceUnwrap(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	c := newWebClient(p)

	w, err := c.GetWorkspace(args[0], true)
	if err != nil {
		return err
	}

	wsSlugs := make(map[string]bool)
	wsSlug := unwrap.DedupedSlug(w.Name, w.ID, wsSlugs)
	wsDir := filepath.Join(workspaceUnwrapDir, wsSlug)
	if err := prepareUnwrapTarget(wsDir, workspaceUnwrapForce); err != nil {
		return err
	}

	return unwrapOneWorkspace(c, w, wsDir)
}

// unwrapOneWorkspace performs the full unwrap including fetching each
// dashboard's full grid (the simplified embedded version has no widgets).
//
// Dashboard folders are prefixed with NN__ to preserve the array order, so
// the inverse wrap can re-assemble dashboards in their original sequence.
func unwrapOneWorkspace(c *client.Client, w *client.Workspace, wsDir string) error {
	if _, err := unwrap.Unwrap(w, wsDir); err != nil {
		return err
	}

	fmt.Printf("  ✓ workspace → %s\n", wsDir)

	dashSlugs := make(map[string]bool)
	width := dashIndexWidth(len(w.Dashboards))
	for i, wd := range w.Dashboards {
		if wd.Dashboard == nil {
			continue
		}
		dashSlug := indexedDashSlug(i, width, wd.Dashboard.Title, wd.Dashboard.ID, dashSlugs)
		dashDir := filepath.Join(wsDir, dashSlug)

		fullDash, err := c.GetDashboard(wd.Dashboard.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "    ✗ dashboard %s: %v\n", wd.Dashboard.ID, err)
			continue
		}
		layout := wd
		layout.Dashboard = nil
		if err := unwrap.UnwrapDashboardFull(fullDash, &layout, dashDir); err != nil {
			fmt.Fprintf(os.Stderr, "    ✗ dashboard %s: %v\n", wd.Dashboard.ID, err)
			continue
		}
		fmt.Printf("    ✓ dashboard %s (%d widgets) → %s\n", wd.Dashboard.ID, len(fullDash.Grid), dashDir)
	}
	return nil
}

// dashIndexWidth returns 2 for up to 100 dashboards, 3 for 1000, etc.
func dashIndexWidth(n int) int {
	if n <= 1 {
		return 2
	}
	return max(len(fmt.Sprintf("%d", n-1)), 2)
}

// indexedDashSlug prefixes the deduped slug with NN__ to encode array order.
func indexedDashSlug(index, width int, title, id string, taken map[string]bool) string {
	base := unwrap.DedupedSlug(title, id, taken)
	return fmt.Sprintf("%0*d__%s", width, index, base)
}

// --- unwrap-all ---

var workspaceUnwrapAllCmd = &cobra.Command{
	Use:   "unwrap-all --dir <dir>",
	Short: "Unwrap every workspace into <dir>/<workspace-slug>/...",
	RunE:  runWorkspaceUnwrapAll,
}

var (
	workspaceUnwrapAllDir   string
	workspaceUnwrapAllForce bool
)

func runWorkspaceUnwrapAll(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	c := newWebClient(p)

	wss, err := c.ListWorkspaces(true)
	if err != nil {
		return err
	}

	wsSlugs := make(map[string]bool)
	var ok, failed int
	for i := range wss {
		w := &wss[i]
		wsSlug := unwrap.DedupedSlug(w.Name, w.ID, wsSlugs)
		wsDir := filepath.Join(workspaceUnwrapAllDir, wsSlug)
		if err := prepareUnwrapTarget(wsDir, workspaceUnwrapAllForce); err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ workspace %s: %v\n", w.ID, err)
			failed++
			continue
		}
		if err := unwrapOneWorkspace(c, w, wsDir); err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ workspace %s: %v\n", w.ID, err)
			failed++
			continue
		}
		ok++
	}
	fmt.Printf("\n%d workspace(s) unwrapped, %d failed.\n", ok, failed)
	if failed > 0 {
		return fmt.Errorf("%d workspace(s) failed", failed)
	}
	return nil
}

// --- unwrap-file ---

var workspaceUnwrapFileCmd = &cobra.Command{
	Use:   "unwrap-file <workspace.json>",
	Short: "Unwrap a workspace from a local JSON file (no API call)",
	Long: `Unwrap a workspace into a directory tree from a JSON file already on disk.

Useful when you have an export produced by "og workspace export" or downloaded
from the OpenGate web UI. Accepts either the raw workspace object or the
{"workspaces":[...]} wrapper produced by the /workspaces/export endpoint.`,
	Args: cobra.ExactArgs(1),
	RunE: runWorkspaceUnwrapFile,
}

var (
	workspaceUnwrapFileDir   string
	workspaceUnwrapFileForce bool
)

func runWorkspaceUnwrapFile(cmd *cobra.Command, args []string) error {
	raw, err := os.ReadFile(args[0])
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	w, err := parseWorkspaceFromBytes(raw)
	if err != nil {
		return err
	}

	wsSlugs := make(map[string]bool)
	wsSlug := unwrap.DedupedSlug(w.Name, w.ID, wsSlugs)
	wsDir := filepath.Join(workspaceUnwrapFileDir, wsSlug)

	if err := prepareUnwrapTarget(wsDir, workspaceUnwrapFileForce); err != nil {
		return err
	}

	if _, err := unwrap.Unwrap(w, wsDir); err != nil {
		return err
	}
	fmt.Printf("  ✓ workspace → %s\n", wsDir)

	// Each embedded dashboard already carries its full grid because the file
	// came from an export (not a /workspaces/?full=1 GET which omits grids).
	// We promote each DashboardSimplified to a full Dashboard so that
	// UnwrapDashboardFull can split the grid into widget folders.
	dashSlugs := make(map[string]bool)
	width := dashIndexWidth(len(w.Dashboards))
	for i, wd := range w.Dashboards {
		if wd.Dashboard == nil {
			continue
		}
		dashSlug := indexedDashSlug(i, width, wd.Dashboard.Title, wd.Dashboard.ID, dashSlugs)
		dashDir := filepath.Join(wsDir, dashSlug)

		fullDash := simplifiedToFullStruct(wd.Dashboard)
		layout := wd
		layout.Dashboard = nil
		if err := unwrap.UnwrapDashboardFull(fullDash, &layout, dashDir); err != nil {
			fmt.Fprintf(os.Stderr, "    ✗ dashboard %s: %v\n", wd.Dashboard.ID, err)
			continue
		}
		fmt.Printf("    ✓ dashboard %s (%d widgets) → %s\n", wd.Dashboard.ID, len(fullDash.Grid), dashDir)
	}
	return nil
}

// parseWorkspaceFromBytes accepts either a raw Workspace JSON or the
// {"workspaces":[...]} wrapper produced by /api/workspaces/export/{id}.
func parseWorkspaceFromBytes(raw []byte) (*client.Workspace, error) {
	// Try wrapper first.
	var wrapper struct {
		Workspaces []client.Workspace `json:"workspaces"`
	}
	if err := json.Unmarshal(raw, &wrapper); err == nil && len(wrapper.Workspaces) > 0 {
		return &wrapper.Workspaces[0], nil
	}

	// Fall back to raw Workspace object.
	var w client.Workspace
	if err := json.Unmarshal(raw, &w); err != nil {
		return nil, fmt.Errorf("parsing workspace JSON: %w", err)
	}
	if w.Name == "" && len(w.Dashboards) == 0 {
		return nil, fmt.Errorf("file does not contain a recognisable workspace")
	}
	return &w, nil
}

// simplifiedToFullStruct lifts a DashboardSimplified into a full Dashboard,
// preserving the Grid which (post the Grid field addition) lives directly on
// the simplified struct after unmarshalling an export payload.
func simplifiedToFullStruct(s *client.DashboardSimplified) *client.Dashboard {
	return &client.Dashboard{
		ID:              s.ID,
		AltID:           s.AltID,
		Title:           s.Title,
		Description:     s.Description,
		Icon:            s.Icon,
		IconType:        s.IconType,
		Owner:           s.Owner,
		Workspaces:      s.Workspaces,
		Users:           s.Users,
		Workgroups:      s.Workgroups,
		AllowedProfiles: s.AllowedProfiles,
		Domains:         s.Domains,
		LastAccess:      s.LastAccess,
		Editable:        s.Editable,
		BackgroundImage: s.BackgroundImage,
		BannerImage:     s.BannerImage,
		Version:         s.Version,
		ExtraConfig:     s.ExtraConfig,
		Grid:            s.Grid,
		TemplateConfig:  s.TemplateConfig,
	}
}

// --- wrap ---

var workspaceWrapCmd = &cobra.Command{
	Use:   "wrap <workspace-dir>",
	Short: "Rebuild a workspace JSON from an unwrapped directory tree",
	Long: `Rebuild the workspace JSON from a directory previously produced by
"og workspace unwrap". The output JSON is suitable for "og workspace import"
on the same or a different tenant.

By default, the rebuilt JSON is written to stdout. Use --out to write to a file.`,
	Args: cobra.ExactArgs(1),
	RunE: runWorkspaceWrap,
}

var workspaceWrapOut string

func runWorkspaceWrap(cmd *cobra.Command, args []string) error {
	w, err := unwrap.Wrap(args[0])
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling workspace: %w", err)
	}
	if workspaceWrapOut == "" {
		fmt.Println(string(data))
		return nil
	}
	if err := os.WriteFile(workspaceWrapOut, data, 0o644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}
	fmt.Printf("Workspace JSON written to %s\n", workspaceWrapOut)
	return nil
}

// --- import ---

var workspaceImportCmd = &cobra.Command{
	Use:   "import -f <file.json> [--update]",
	Short: "Import (create or update) a workspace from a JSON file",
	Long: `Import a workspace from a JSON file.

By default, the file is POSTed as a new workspace. If the JSON contains an
"_id" that already exists in the tenant, OpenGate returns HTTP 400 with a
duplicate-key error; re-run with --update to overwrite the existing workspace
via PUT instead.

This makes the export → unwrap → wrap → import cycle frictionless when
re-deploying changes to the same workspace.`,
	RunE: runWorkspaceImport,
}

var (
	workspaceImportFile   string
	workspaceImportUpdate bool
)

func runWorkspaceImport(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}

	body, err := os.ReadFile(workspaceImportFile)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	var w client.Workspace
	if err := json.Unmarshal(body, &w); err != nil {
		return fmt.Errorf("parsing workspace JSON: %w", err)
	}
	if w.ID == "" {
		return fmt.Errorf("workspace JSON has no _id")
	}

	c := newWebClient(p)

	if workspaceImportUpdate {
		if err := c.UpdateWorkspaceDeep(&w); err != nil {
			return err
		}
		fmt.Printf("Workspace %s updated successfully (workspace + %d dashboard(s)).\n", w.ID, countEmbeddedDashboards(&w))
		return nil
	}

	if err := c.ImportWorkspaceDeep(&w); err != nil {
		if isDuplicateKeyError(err) {
			return fmt.Errorf("%w\n\nThe workspace _id already exists. Re-run with --update to overwrite it (and its dashboards) via PUT", err)
		}
		return err
	}
	fmt.Printf("Workspace %s imported successfully (workspace + %d dashboard(s)).\n", w.ID, countEmbeddedDashboards(&w))
	return nil
}

func countEmbeddedDashboards(w *client.Workspace) int {
	n := 0
	for _, wd := range w.Dashboards {
		if wd.Dashboard != nil {
			n++
		}
	}
	return n
}

// extractIDFromJSON pulls the "_id" (or "id") top-level field from a JSON body.
func extractIDFromJSON(body []byte) (string, error) {
	var probe struct {
		ID    string `json:"_id"`
		AltID string `json:"id"`
	}
	if err := json.Unmarshal(body, &probe); err != nil {
		return "", fmt.Errorf("parsing JSON: %w", err)
	}
	if probe.ID != "" {
		return probe.ID, nil
	}
	if probe.AltID != "" {
		return probe.AltID, nil
	}
	return "", fmt.Errorf("no _id or id field found")
}

// isDuplicateKeyError detects the MongoDB-backed "11000 duplicate key" error
// surfaced by OpenGate as HTTP 400.
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "11000") || strings.Contains(msg, "duplicate key")
}

// --- update ---

var workspaceUpdateCmd = &cobra.Command{
	Use:   "update <workspace-id> -f <file.json>",
	Short: "Update an existing workspace from a JSON file",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkspaceUpdate,
}

var workspaceUpdateFile string

func runWorkspaceUpdate(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}

	body, err := os.ReadFile(workspaceUpdateFile)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	c := newWebClient(p)
	if err := c.UpdateWorkspace(args[0], body); err != nil {
		return err
	}

	fmt.Println("Workspace updated successfully.")
	return nil
}

// --- delete ---

var workspaceDeleteCmd = &cobra.Command{
	Use:   "delete <workspace-id>",
	Short: "Delete a workspace",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkspaceDelete,
}

func runWorkspaceDelete(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}

	c := newWebClient(p)
	if err := c.DeleteWorkspace(args[0]); err != nil {
		return err
	}

	fmt.Println("Workspace deleted successfully.")
	return nil
}

// --- init ---

func init() {
	workspaceListCmd.Flags().BoolVar(&workspaceListFull, "full", false, "include embedded dashboards in each workspace")
	workspaceGetCmd.Flags().BoolVar(&workspaceGetFull, "full", false, "include embedded dashboards")

	workspaceExportCmd.Flags().StringVar(&workspaceExportOut, "out", "", "write export JSON to this file (default: stdout)")
	workspaceExportCmd.Flags().StringVar(&workspaceExportDir, "dir", "", "write export to <dir>/<workspace-id>.json (auto-naming)")
	workspaceExportCmd.Flags().BoolVar(&workspaceExportFull, "full", false, "use GET /workspaces/{id}?full=1 instead of /workspaces/export/{id}")

	workspaceImportCmd.Flags().StringVarP(&workspaceImportFile, "file", "f", "", "path to JSON file with workspace definition")
	workspaceImportCmd.Flags().BoolVar(&workspaceImportUpdate, "update", false, "update an existing workspace (PUT) instead of creating (POST)")
	_ = workspaceImportCmd.MarkFlagRequired("file")

	workspaceUpdateCmd.Flags().StringVarP(&workspaceUpdateFile, "file", "f", "", "path to JSON file with workspace definition")
	_ = workspaceUpdateCmd.MarkFlagRequired("file")

	workspaceExportAllCmd.Flags().StringVar(&workspaceExportAllDir, "dir", "", "destination directory (required)")
	workspaceExportAllCmd.Flags().BoolVar(&workspaceExportAllFull, "full", false, "use GET /workspaces/{id}?full=1 instead of /workspaces/export/{id}")
	_ = workspaceExportAllCmd.MarkFlagRequired("dir")

	workspaceUnwrapCmd.Flags().StringVar(&workspaceUnwrapDir, "dir", "", "destination directory (required)")
	workspaceUnwrapCmd.Flags().BoolVar(&workspaceUnwrapForce, "force", false, "overwrite destination if it already exists")
	_ = workspaceUnwrapCmd.MarkFlagRequired("dir")

	workspaceUnwrapAllCmd.Flags().StringVar(&workspaceUnwrapAllDir, "dir", "", "destination directory (required)")
	workspaceUnwrapAllCmd.Flags().BoolVar(&workspaceUnwrapAllForce, "force", false, "overwrite each workspace destination if it already exists")
	_ = workspaceUnwrapAllCmd.MarkFlagRequired("dir")

	workspaceUnwrapFileCmd.Flags().StringVar(&workspaceUnwrapFileDir, "dir", "", "destination directory (required)")
	workspaceUnwrapFileCmd.Flags().BoolVar(&workspaceUnwrapFileForce, "force", false, "overwrite destination if it already exists")
	_ = workspaceUnwrapFileCmd.MarkFlagRequired("dir")

	workspaceWrapCmd.Flags().StringVar(&workspaceWrapOut, "out", "", "write rebuilt JSON to this file (default: stdout)")

	workspaceDeployCmd.Flags().BoolVar(&workspaceDeployUpdate, "update", false, "update an existing workspace (PUT) instead of creating (POST)")

	workspaceCmd.AddCommand(workspaceListCmd)
	workspaceCmd.AddCommand(workspaceGetCmd)
	workspaceCmd.AddCommand(workspaceExportCmd)
	workspaceCmd.AddCommand(workspaceExportAllCmd)
	workspaceCmd.AddCommand(workspaceUnwrapCmd)
	workspaceCmd.AddCommand(workspaceUnwrapAllCmd)
	workspaceCmd.AddCommand(workspaceUnwrapFileCmd)
	workspaceCmd.AddCommand(workspaceWrapCmd)
	workspaceCmd.AddCommand(workspaceDeployCmd)
	workspaceCmd.AddCommand(workspaceImportCmd)
	workspaceCmd.AddCommand(workspaceUpdateCmd)
	workspaceCmd.AddCommand(workspaceDeleteCmd)

	rootCmd.AddCommand(workspaceCmd)
}
