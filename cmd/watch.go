package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/carlosprados/og-cli/v2/internal/basestate"
	"github.com/carlosprados/og-cli/v2/internal/config"
	"github.com/carlosprados/og-cli/v2/internal/unwrap"
	"github.com/carlosprados/og-cli/v2/internal/validate"
	"github.com/carlosprados/og-cli/v2/internal/watch"
	"github.com/carlosprados/og-cli/v2/pkg/opengate"
	"github.com/spf13/cobra"
)

var (
	watchDryRun          bool
	watchNoValidate      bool
	watchAllowProduction bool
	watchDebounce        time.Duration
	watchJSON            bool
)

// deployer pushes an artifact payload. Each family supplies one.
type deployer func(ctx context.Context, c *opengate.Client, org, id string, body json.RawMessage) error

// watchSpec is what a family provides to get a watch subcommand.
type watchSpec struct {
	Descriptor unwrap.Descriptor
	Use        string
	Short      string
	Fetch      fetcher
	Deploy     deployer
	// Channel reports the channel in play, for the event log.
	Channel func() string
}

// watchEvent is one NDJSON line under --json, for an editor or a log collector.
type watchEvent struct {
	Time   string `json:"time"`
	Event  string `json:"event"`
	Kind   string `json:"kind"`
	Dir    string `json:"dir"`
	ID     string `json:"id,omitempty"`
	State  string `json:"state,omitempty"`
	Detail string `json:"detail,omitempty"`
}

func newWatchCmd(spec watchSpec) *cobra.Command {
	return &cobra.Command{
		Use:   spec.Use,
		Short: spec.Short,
		Long: spec.Short + `

Watches a directory tree and deploys each artifact when its files change. This
is the only og command that writes to the platform without a decision per
action, so it is deliberately cautious:

  • A changed file resolves to the SMALLEST deployable unit — the artifact
    directory it lives in, not the whole tree.
  • A burst of events becomes one deploy. Editors replace files rather than
    writing them, so one save produces three or four events.
  • Every artifact is validated before it is pushed. --no-validate must be
    explicit; deploying a syntax error into a live connector function is an
    incident, not a typo.
  • A CONFLICT — the remote changed since you pulled, and so did you — refuses
    to deploy. There is no --force: the whole point is that overwriting
    someone else's edit should not be one keystroke away.
  • Against a profile marked production: true, it refuses to start without
    --allow-production.

Start with --dry-run to see what it would do.

Examples:
  og rules watch rules/ --org sensehat --dry-run
  og rules watch rules/ --org sensehat
  og rules watch rules/ --org sensehat --json | tee watch.ndjson`,
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
			orgName, err := resolveOrg(p)
			if err != nil {
				return err
			}

			if err := guardProduction(p); err != nil {
				return err
			}
			warnSessionSharing(spec.Descriptor.Kind)

			c := opengate.New(p.Host, p.Token, p.ClientOptions()...)
			channel := ""
			if spec.Channel != nil {
				channel = spec.Channel()
			}

			w, err := watch.New(watch.Options{
				Root:     root,
				Debounce: watchDebounce,
				IsTarget: func(dir string) bool {
					_, err := os.Stat(filepath.Join(dir, spec.Descriptor.MetaFile))
					return err == nil
				},
			})
			if err != nil {
				return err
			}
			defer func() { _ = w.Close() }()

			emit := watchEmitter()
			emit(watchEvent{Event: "started", Kind: string(spec.Descriptor.Kind), Dir: root,
				Detail: fmt.Sprintf("%d directories, org %s, debounce %s, dry-run %t",
					w.Watched(), orgName, watchDebounce, watchDryRun)})

			return w.Run(cmd.Context(), func(target watch.Target) error {
				return deployTarget(cmd.Context(), spec, c, orgName, channel, string(target), emit)
			})
		},
	}
}

// deployTarget is the whole per-save pipeline: wrap, validate, classify, push.
func deployTarget(ctx context.Context, spec watchSpec, c *opengate.Client,
	orgName, channel, dir string, emit func(watchEvent)) error {

	kind := string(spec.Descriptor.Kind)
	ev := watchEvent{Kind: kind, Dir: dir}

	if !watchNoValidate {
		result := validate.Artifact(spec.Descriptor, dir)
		if !result.OK() {
			detail := ""
			for _, f := range result.Findings {
				if f.Severity == validate.Error {
					detail = f.String()
					break
				}
			}
			ev.Event, ev.Detail = "invalid", detail
			emit(ev)
			return nil
		}
	}

	body, err := spec.Descriptor.Wrap(dir, nil)
	if err != nil {
		ev.Event, ev.Detail = "error", err.Error()
		emit(ev)
		return nil
	}

	id := artifactID(spec.Descriptor, body)
	ev.ID = id
	if id == "" {
		ev.Event, ev.Detail = "skipped", "no identifier — watch only updates existing artifacts"
		emit(ev)
		return nil
	}

	// Fetch the remote to classify before writing. This is the check that makes
	// watch safe to leave running: without it, every save is a blind overwrite.
	remote, ferr := spec.Fetch(ctx, c, orgName, id)
	if ferr != nil {
		ev.Event, ev.Detail = "error", fmt.Sprintf("cannot read the remote artifact: %v", ferr)
		emit(ev)
		return nil
	}

	if store, ok := basestate.Find(dir); ok {
		cmp, _, cerr := store.ClassifyArtifact(spec.Descriptor.Kind, dir, body, remote)
		if cerr == nil {
			ev.State = cmp.State.String()
			if cmp.State == basestate.Conflict {
				ev.Event = "refused"
				ev.Detail = "the remote changed since you pulled — deploying would discard it. Run diff, then pull or resolve by hand."
				emit(ev)
				return nil
			}
			if cmp.State == basestate.Clean {
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

	if err := spec.Deploy(ctx, c, orgName, id, body); err != nil {
		ev.Event, ev.Detail = "failed", err.Error()
		emit(ev)
		return nil
	}

	// Record the new remote state as the base, or the next save is classified
	// against a snapshot that is now two versions old and reports a conflict
	// against your own work.
	if store, ok := basestate.Find(dir); ok {
		_ = store.Record(spec.Descriptor.Kind, id, spec.Descriptor.NameOf(body), dir, body,
			basestate.Target{Profile: profileName(), Org: orgName, Channel: channel})
	}

	ev.Event = "deployed"
	emit(ev)
	return nil
}

// guardProduction refuses to start against a profile marked production.
func guardProduction(p *config.Profile) error {
	if !p.Production || watchAllowProduction {
		return nil
	}
	return fmt.Errorf("profile %q is marked production: true — watch deploys on every save; "+
		"pass --allow-production to do it anyway", profileName())
}

// warnSessionSharing warns only where it applies.
//
// og re-signs transparently on a 401, which invalidates the browser session for
// the same user — OpenGate allows one. With watch running, the loop is: save,
// re-sign, the developer's browser session dies, they reload, the CLI token
// dies. But the re-sign path is the Web API client, used only by workspaces and
// dashboards; rules, connector functions and provision functions go through the
// North API and never re-sign. A warning printed everywhere is a warning nobody
// reads.
func warnSessionSharing(kind unwrap.Kind) {
	if kind != unwrap.KindWorkspace && kind != unwrap.KindDashboard {
		return
	}
	fmt.Fprintln(os.Stderr,
		"  warning: og re-signs on a 401, and OpenGate allows one web session per user.\n"+
			"           While watching, that can invalidate your browser session repeatedly.\n"+
			"           Use a dedicated service account for the CLI.")
}

func profileName() string {
	if profile != "" {
		return profile
	}
	if cfg != nil {
		return cfg.DefaultProfile
	}
	return ""
}

// watchEmitter returns a printer for watch events: NDJSON under --json, one
// human line otherwise.
func watchEmitter() func(watchEvent) {
	return func(ev watchEvent) {
		ev.Time = time.Now().UTC().Format(time.RFC3339)
		if watchJSON {
			// NDJSON: one compact object per line. Indented output would be
			// unreadable to the line-by-line consumers this is for — an editor
			// extension, a log collector.
			line, err := json.Marshal(ev)
			if err != nil {
				return
			}
			_, _ = fmt.Fprintf(os.Stdout, "%s\n", line)
			return
		}
		line := fmt.Sprintf("%s  %-13s %s", time.Now().Format("15:04:05"), ev.Event, filepath.Base(ev.Dir))
		if ev.State != "" {
			line += fmt.Sprintf("  [%s]", ev.State)
		}
		if ev.Detail != "" {
			line += "  " + ev.Detail
		}
		fmt.Println(line)
	}
}

func addWatchFlags(c *cobra.Command) {
	c.Flags().BoolVar(&watchDryRun, "dry-run", false, "report what would be deployed without writing anything")
	c.Flags().BoolVar(&watchNoValidate, "no-validate", false, "deploy without validating first (not recommended)")
	c.Flags().BoolVar(&watchAllowProduction, "allow-production", false, "allow watching a profile marked production: true")
	c.Flags().DurationVar(&watchDebounce, "debounce", watch.DefaultDebounce, "how long to wait for a burst of file events to settle")
	c.Flags().BoolVar(&watchJSON, "json", false, "emit NDJSON events, one per line")
}
