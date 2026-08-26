package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/carlosprados/og-cli/v2/internal/canon"
	"github.com/carlosprados/og-cli/v2/internal/diff"
	"github.com/carlosprados/og-cli/v2/internal/output"
	"github.com/carlosprados/og-cli/v2/internal/unwrap"
	"github.com/carlosprados/og-cli/v2/pkg/opengate"
	"github.com/spf13/cobra"
)

// Flags shared by every family's diff subcommand.
var (
	diffNameOnly bool
	diffExitCode bool
	diffAgainst  string
	diffContext  int
)

// fetcher retrieves one artifact's payload by identifier. Each family supplies
// one, closing over its own client and scope.
type fetcher func(ctx context.Context, c *opengate.Client, org, id string) (json.RawMessage, error)

// diffSpec is what a family provides to get a diff subcommand.
type diffSpec struct {
	Descriptor unwrap.Descriptor
	// Channel is the channel flag's value, for families scoped by channel.
	Channel func() string
	// IDKey reads the identifier out of the local payload.
	Fetch fetcher
	// Use is the subcommand's usage line, e.g. "diff <rule-dir>".
	Use   string
	Short string
	Long  string
}

// newDiffCmd builds a family's diff subcommand. One implementation for every
// family: the differences are the descriptor, the client and the scope.
func newDiffCmd(spec diffSpec) *cobra.Command {
	return &cobra.Command{
		Use:   spec.Use,
		Short: spec.Short,
		Long:  spec.Long,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			artifactDir := args[0]

			local, err := spec.Descriptor.Wrap(artifactDir, hintWarner())
			if err != nil {
				return err
			}

			id := artifactID(spec.Descriptor, local)
			if id == "" {
				return fmt.Errorf("%s has no %s in %s — pull it first, or there is nothing remote to compare against",
					filepath.Base(artifactDir), spec.Descriptor.IDKey, spec.Descriptor.MetaFile)
			}

			// --against compares the same artifact in another tenant, which is
			// the promotion question: what differs between staging and prod?
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
			orgName, err := resolveOrg(p)
			if err != nil {
				return err
			}

			c := opengate.New(p.Host, p.Token, p.ClientOptions()...)
			remote, err := spec.Fetch(cmd.Context(), c, orgName, id)
			if err != nil {
				return err
			}

			result, err := diff.Compare(spec.Descriptor, artifactDir, local, remote,
				diff.Options{Scope: scope, ContextLines: diffContext})
			if err != nil {
				return err
			}
			if diffAgainst != "" {
				result.Origin = fmt.Sprintf("compared against profile %s (org %s)", diffAgainst, orgName)
			}

			if err := emitDiff(result); err != nil {
				return err
			}
			if diffExitCode && result.Changed() {
				return diffFound()
			}
			return nil
		},
	}
}

// artifactID reads the identifier from a local payload using the family's key.
func artifactID(d unwrap.Descriptor, payload json.RawMessage) string {
	var node map[string]any
	if json.Unmarshal(payload, &node) != nil {
		return ""
	}
	s, _ := node[d.IDKey].(string)
	return s
}

// emitDiff writes one result in the requested format.
func emitDiff(result diff.Result) error {
	if outFmt == output.FormatJSON {
		return output.PrintEnvelope(os.Stdout, "diff", result)
	}
	if !result.Changed() {
		fmt.Printf("No differences.")
		if result.Ignored != "" {
			fmt.Printf(" (%s)", result.Ignored)
		}
		fmt.Println()
		return nil
	}
	fmt.Print(result.RenderText(diffNameOnly))
	return nil
}

// addDiffFlags installs the shared flags on a family's diff subcommand.
func addDiffFlags(c *cobra.Command) {
	c.Flags().BoolVar(&diffNameOnly, "name-only", false, "list the artifact and its state, without the differences")
	c.Flags().BoolVar(&diffExitCode, "exit-code", false, "exit 1 when there are differences, 0 when there are none (2 on error) — for CI drift gates")
	c.Flags().StringVar(&diffAgainst, "against", "", "compare against another profile instead of the active one (cross-tenant: identifiers and ownership are ignored)")
	c.Flags().IntVar(&diffContext, "context", 3, "lines of context around each code change")
}
