package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/carlosprados/og-cli/v2/internal/output"
	"github.com/carlosprados/og-cli/v2/internal/unwrap"
	"github.com/carlosprados/og-cli/v2/pkg/opengate"
	"github.com/spf13/cobra"
)

// `og <family> show <id> --path <file>` prints one remote code file, raw.
//
// It exists for the VS Code extension, which renders diffs natively: it
// registers a TextDocumentContentProvider for an `og-remote:` scheme and hands
// the two URIs to vscode.diff(). The provider needs the remote side of a single
// file on stdout, with nothing wrapped around it — no JSON envelope, no table,
// no trailing summary.
//
// That is also why this is not a flag on `get`. `get` prints the artifact's
// payload through the shared formatter and honours -o json; this prints bytes.
// Overloading one command with two output contracts would make both worse.

var showPath string

func newShowCmd(d unwrap.Descriptor, fetch fetcher, use, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Long: short + `

With --path, writes that file's remote content to stdout and nothing else, so it
can be redirected or piped. Without it, lists the files this artifact carries.

The names are the ones ` + "`pull`" + ` writes on disk, so a path from the local tree
addresses the same file remotely.

Examples:
  og rules show <rule-id> --org sensehat
  og rules show <rule-id> --org sensehat --path javascript.js
  og rules show <rule-id> --org sensehat --path javascript.js > /tmp/remote.js`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := activeProfile()
			if err != nil {
				return err
			}
			orgName, err := resolveOrg(p)
			if err != nil {
				return err
			}

			c := opengate.New(p.Host, p.Token, p.ClientOptions()...)
			payload, err := fetch(cmd.Context(), c, orgName, args[0])
			if err != nil {
				return err
			}

			files, err := remoteCodeFiles(d, payload)
			if err != nil {
				return err
			}

			if showPath == "" {
				return listRemoteCodeFiles(d, args[0], files)
			}

			content, ok := files[showPath]
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

// remoteCodeFiles extracts a payload's code files using the family's contract —
// the same extraction `pull` performs, so the names match the local tree.
func remoteCodeFiles(d unwrap.Descriptor, payload json.RawMessage) (map[string]string, error) {
	var node any
	if err := json.Unmarshal(payload, &node); err != nil {
		return nil, fmt.Errorf("parsing the remote payload: %w", err)
	}
	meta, _ := node.(map[string]any)
	_, files, _ := d.Contract(meta).Extract(node, nil)
	return files, nil
}

func listRemoteCodeFiles(d unwrap.Descriptor, id string, files map[string]string) error {
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
		}{Kind: string(d.Kind), ID: id, Files: []entry{}}
		for _, n := range names {
			out.Files = append(out.Files, entry{File: n, Bytes: len(files[n]), Lines: len(splitLines(files[n]))})
		}
		return output.PrintEnvelope(os.Stdout, "show", out)
	}

	if len(names) == 0 {
		fmt.Printf("%s %s carries no code files.\n", d.Kind, id)
		return nil
	}
	fmt.Printf("%s %s:\n", d.Kind, id)
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
