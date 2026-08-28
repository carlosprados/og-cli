package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/carlosprados/og-cli/v2/internal/basestate"
	"github.com/carlosprados/og-cli/v2/internal/unwrap"
	"github.com/carlosprados/og-cli/v2/internal/watch"
	"github.com/carlosprados/og-cli/v2/pkg/opengate"
	"github.com/spf13/cobra"
)

// `og workspace watch` deploys DASHBOARDS, not workspaces.
//
// The watch loop's rule is that a changed file resolves to the smallest
// deployable unit, and for a widget edit that is the dashboard the widget sits
// in — a whole-workspace deploy on every keystroke is exactly the blast radius
// the rule exists to avoid. Dashboard directories are also deeper than the
// workspace directory, so the watcher's nearest-target resolution picks them
// without any special casing.
//
// Editing workspace.json is therefore reported and skipped rather than
// deployed: changing a workspace's name, colour or sharing is a deliberate act,
// not part of an edit loop, and `og workspace deploy` is the command for it.

var workspaceWatchCmd = &cobra.Command{
	Use:   "watch <dir>",
	Short: "Deploy each dashboard as its files change",
	Long: `Watch a pulled workspace tree and deploy each dashboard when its files change.

The unit of deployment is the DASHBOARD: editing a widget's JavaScript deploys
the dashboard that widget belongs to, not the whole workspace. Edits to
workspace.json are reported and skipped — use 'og workspace deploy' for those.

The same guards as every other watch apply:

  • A burst of file events becomes one deploy.
  • A CONFLICT — the remote changed since you pulled, and so did you — refuses
    to deploy, with no --force.
  • Against a profile marked production: true, it refuses to start without
    --allow-production.

Conflict detection needs the snapshot 'og workspace pull' records in .og/. A
dashboard pulled before that existed, or imported from an export file, has none
and is reported as unknown rather than silently overwritten.

Start with --dry-run.

Examples:
  og workspace watch ws/ --dry-run
  og workspace watch ws/
  og workspace watch ws/ --json | tee watch.ndjson`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root := args[0]
		if info, err := os.Stat(root); err != nil || !info.IsDir() {
			return fmt.Errorf("%s is not a directory", root)
		}

		p, err := activeProfile()
		if err != nil {
			return err
		}
		if err := guardProduction(p); err != nil {
			return err
		}
		// Workspaces and dashboards are the Web API families, the only ones
		// where og's transparent re-sign can evict the developer's browser
		// session — which is precisely the loop a watch would run in.
		warnSessionSharing(unwrap.KindDashboard)

		c := newWebClient(p)

		w, err := watch.New(watch.Options{
			Root:     root,
			Debounce: watchDebounce,
			IsTarget: func(dir string) bool {
				_, err := os.Stat(filepath.Join(dir, "dashboard.json"))
				return err == nil
			},
		})
		if err != nil {
			return err
		}
		defer func() { _ = w.Close() }()

		emit := watchEmitter()
		emit(watchEvent{Event: "started", Kind: string(unwrap.KindDashboard), Dir: root,
			Detail: fmt.Sprintf("%d directories, debounce %s, dry-run %t",
				w.Watched(), watchDebounce, watchDryRun)})

		return w.Run(cmd.Context(), func(target watch.Target) error {
			return deployDashboardTarget(cmd.Context(), c, string(target), emit)
		})
	},
}

// deployDashboardTarget is the per-save pipeline for one dashboard.
func deployDashboardTarget(ctx context.Context, c *opengate.Client, dir string, emit func(watchEvent)) error {
	ev := watchEvent{Kind: string(unwrap.KindDashboard), Dir: dir}

	// Wrapping is the validation. It refuses a malformed widget directory
	// rather than silently dropping the widget, which is the check that matters
	// here — there is no separate validator for a hierarchical family.
	dash, _, err := unwrap.WrapDashboard(dir, nil)
	if err != nil {
		ev.Event, ev.Detail = "invalid", err.Error()
		emit(ev)
		return nil
	}

	ev.ID = dash.ID
	if dash.ID == "" {
		ev.Event, ev.Detail = "skipped", "no _id in dashboard.json — watch only updates existing dashboards"
		emit(ev)
		return nil
	}

	body, err := json.Marshal(dash)
	if err != nil {
		ev.Event, ev.Detail = "error", err.Error()
		emit(ev)
		return nil
	}

	// Read the remote before writing. Without this every save is a blind
	// overwrite, which is the thing that makes leaving a watch running unsafe.
	remoteDash, err := c.GetDashboard(ctx, dash.ID)
	if err != nil {
		ev.Event, ev.Detail = "error", fmt.Sprintf("cannot read the remote dashboard: %v", err)
		emit(ev)
		return nil
	}
	remote, err := json.Marshal(remoteDash)
	if err != nil {
		ev.Event, ev.Detail = "error", err.Error()
		emit(ev)
		return nil
	}

	if store, ok := basestate.Find(dir); ok {
		cmp, _, cerr := store.ClassifyArtifact(unwrap.KindDashboard, dir, body, remote)
		if cerr == nil {
			ev.State = cmp.State.String()
			switch cmp.State {
			case basestate.Conflict:
				ev.Event = "refused"
				ev.Detail = "the remote changed since you pulled — deploying would discard it. Run `og workspace diff`, then pull or resolve by hand."
				emit(ev)
				return nil
			case basestate.Clean:
				ev.Event, ev.Detail = "unchanged", "nothing to deploy"
				emit(ev)
				return nil
			}
		}
	}

	if watchDryRun {
		ev.Event, ev.Detail = "would-deploy", "dry run"
		emit(ev)
		return nil
	}

	if err := c.UpdateDashboard(ctx, dash.ID, body); err != nil {
		ev.Event, ev.Detail = "failed", err.Error()
		emit(ev)
		return nil
	}

	// Re-read and record what the platform now holds. Recording the payload we
	// SENT would drift from what it stored — the platform stamps fields — and
	// the next save would classify against a base that never existed remotely.
	if store, ok := basestate.Find(dir); ok {
		if fresh, ferr := c.GetDashboard(ctx, dash.ID); ferr == nil {
			if raw, merr := json.Marshal(fresh); merr == nil {
				_ = store.Record(unwrap.KindDashboard, dash.ID, dash.Title, dir, raw,
					syncTarget(nil, "", ""))
			}
		}
	}

	ev.Event = "deployed"
	emit(ev)
	return nil
}

func init() {
	addWatchFlags(workspaceWatchCmd)
	workspaceCmd.AddCommand(workspaceWatchCmd)
}
