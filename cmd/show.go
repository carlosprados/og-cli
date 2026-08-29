package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/carlosprados/og-cli/v2/internal/config"
	"github.com/carlosprados/og-cli/v2/internal/output"
	"github.com/carlosprados/og-cli/v2/internal/unwrap"
	"github.com/carlosprados/og-cli/v2/pkg/opengate"
	"github.com/spf13/cobra"
)

// `og <family> show <id> --path <file>` prints one remote code file, raw.
//
// It exists for the editor plugins, which render diffs natively: og.nvim opens
// the content in a scratch buffer beside the file, og-vscode registers a
// TextDocumentContentProvider for an `og-remote:` scheme and hands the two URIs
// to vscode.diff(). Both need the remote side of a single file on stdout, with
// nothing wrapped around it — no JSON envelope, no table, no trailing summary.
//
// That is also why this is not a flag on `get`. `get` prints the artifact's
// payload through the shared formatter and honours -o json; this prints bytes.
// Overloading one command with two output contracts would make both worse.
//
// Families differ in three things and this is a table of them: how to reach the
// platform, how to turn what comes back into files, and how a requested path is
// matched against them. The flat families answer all three from their
// descriptor; a dashboard is a tree and answers them its own way.

var showPath string

// showSpec is what a family provides to get a show subcommand.
type showSpec struct {
	Kind unwrap.Kind

	// Use is the subcommand's usage line, e.g. "show <rule-id>".
	Use   string
	Short string
	// Examples are printed under the description. Each family addresses its own
	// artifacts, and a shared block of examples would be wrong for all but one.
	Examples []string

	// Connect builds the client this family talks through and the scope it
	// needs: the North API with an organization for the flat families, the Web
	// API with none for dashboards.
	Connect func(p *config.Profile) (*opengate.Client, string, error)

	// Files returns the artifact's code files, keyed by the path `pull` writes
	// them to, so a path from the local tree addresses the same file remotely.
	Files func(ctx context.Context, c *opengate.Client, org, id string) (map[string]string, error)

	// Resolve matches a requested path against those files. Optional: without
	// one the match is exact, which is all a flat family needs.
	Resolve func(files map[string]string, requested string) (string, bool)
}

// northConnect is the flat families' connection: the North API, scoped to an
// organization.
func northConnect(p *config.Profile) (*opengate.Client, string, error) {
	orgName, err := resolveOrg(p)
	if err != nil {
		return nil, "", err
	}
	return opengate.New(p.Host, p.Token, p.ClientOptions()...), orgName, nil
}

// descriptorFiles turns a flat family's fetcher into a Files function using the
// family's own code contract — the same extraction `pull` performs, so the
// names match the local tree.
func descriptorFiles(d unwrap.Descriptor, fetch fetcher) func(context.Context, *opengate.Client, string, string) (map[string]string, error) {
	return func(ctx context.Context, c *opengate.Client, org, id string) (map[string]string, error) {
		payload, err := fetch(ctx, c, org, id)
		if err != nil {
			return nil, err
		}
		return remoteCodeFiles(d, payload)
	}
}

// remoteCodeFiles extracts a flat payload's code files using the family's
// contract — the same extraction `pull` performs, so the names match the local
// tree.
func remoteCodeFiles(d unwrap.Descriptor, payload json.RawMessage) (map[string]string, error) {
	var node any
	if err := json.Unmarshal(payload, &node); err != nil {
		return nil, fmt.Errorf("parsing the remote payload: %w", err)
	}
	meta, _ := node.(map[string]any)
	_, files, _ := d.Contract(meta).Extract(node, nil)
	return files, nil
}

// newShowCmd builds a flat family's show subcommand from its descriptor.
func newShowCmd(d unwrap.Descriptor, fetch fetcher, use, short string, examples ...string) *cobra.Command {
	return newShowCmdSpec(showSpec{
		Kind:     d.Kind,
		Use:      use,
		Short:    short,
		Examples: examples,
		Connect:  northConnect,
		Files:    descriptorFiles(d, fetch),
	})
}

func newShowCmdSpec(spec showSpec) *cobra.Command {
	long := spec.Short + `

With --path, writes that file's remote content to stdout and nothing else, so it
can be redirected or piped. Without it, lists the files this artifact carries.

The names are the ones ` + "`pull`" + ` writes on disk, so a path from the local tree
addresses the same file remotely.`
	if len(spec.Examples) > 0 {
		long += "\n\nExamples:\n  " + strings.Join(spec.Examples, "\n  ")
	}

	return &cobra.Command{
		Use:   spec.Use,
		Short: spec.Short,
		Long:  long,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := activeProfile()
			if err != nil {
				return err
			}
			c, orgName, err := spec.Connect(p)
			if err != nil {
				return err
			}

			files, err := spec.Files(cmd.Context(), c, orgName, args[0])
			if err != nil {
				return err
			}

			if showPath == "" {
				return listRemoteCodeFiles(spec.Kind, args[0], files)
			}

			resolve := spec.Resolve
			if resolve == nil {
				resolve = exactPath
			}
			content, ok := resolve(files, showPath)
			if !ok {
				return fmt.Errorf("%s carries no file %q\n  it has: %s",
					args[0], showPath, joinOrNone(sortedFileNames(files)))
			}
			// Raw, unterminated: the caller is a diff view or a redirect, and an
			// added newline would show as a spurious last-line change.
			_, err = os.Stdout.WriteString(content)
			return err
		},
	}
}

func exactPath(files map[string]string, requested string) (string, bool) {
	content, ok := files[requested]
	return content, ok
}

func listRemoteCodeFiles(kind unwrap.Kind, id string, files map[string]string) error {
	names := sortedFileNames(files)

	if outFmt == output.FormatJSON {
		type entry struct {
			File  string `json:"file"`
			Bytes int    `json:"bytes"`
			Lines int    `json:"lines"`
		}
		out := struct {
			Kind  string  `json:"kind"`
			ID    string  `json:"id"`
			Files []entry `json:"files"`
		}{Kind: string(kind), ID: id, Files: []entry{}}
		for _, n := range names {
			out.Files = append(out.Files, entry{File: n, Bytes: len(files[n]), Lines: len(splitLines(files[n]))})
		}
		return output.PrintEnvelope(os.Stdout, "show", out)
	}

	if len(names) == 0 {
		fmt.Printf("%s %s carries no code files.\n", kind, id)
		return nil
	}
	fmt.Printf("%s %s:\n", kind, id)
	for _, n := range names {
		fmt.Printf("  %-40s %6d bytes, %d lines\n", n, len(files[n]), len(splitLines(files[n])))
	}
	fmt.Printf("\nUse --path <file> to print one of them.\n")
	return nil
}

func sortedFileNames(files map[string]string) []string {
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func joinOrNone(names []string) string {
	if len(names) == 0 {
		return "no code files at all"
	}
	out := ""
	for i, n := range names {
		if i > 0 {
			out += ", "
		}
		out += n
	}
	return out
}

// addShowFlags installs the shared flag on a family's show subcommand.
func addShowFlags(c *cobra.Command) {
	c.Flags().StringVar(&showPath, "path", "",
		"print this file's remote content to stdout, raw (default: list the files)")
}
