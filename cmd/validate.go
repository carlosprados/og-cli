package cmd

import (
	"fmt"
	"os"

	"github.com/carlosprados/og-cli/v2/internal/output"
	"github.com/carlosprados/og-cli/v2/internal/unwrap"
	"github.com/carlosprados/og-cli/v2/internal/validate"
	"github.com/spf13/cobra"
)

var validateExitCode bool

// newValidateCmd builds a family's validate subcommand. Purely local: it makes
// no API call, so it works offline and in CI without credentials.
func newValidateCmd(d unwrap.Descriptor, use, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Long: short + `

Runs locally — no API call, no credentials — so it works offline and as a CI
gate. Checks that the metadata is valid JSON, that the declared code files are
present, that brackets in the JavaScript balance, and the per-family rules that
catch an artifact which would deploy but never fire.

It is not a JavaScript parser: the script itself is covered by the generated
typings, which a real type-checker validates in the editor. See 'og typegen'.

Errors mean the artifact is broken. Warnings mean it is deployable but probably
not what was intended — a stray .js file that will not be deployed, a REQUEST
connector function with no operationName.

With --exit-code: 1 when there are errors, 0 when there are none, 2 on failure.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result := validate.Artifact(d, args[0])

			if outFmt == output.FormatJSON {
				if err := output.PrintEnvelope(os.Stdout, "validate", result); err != nil {
					return err
				}
			} else {
				if len(result.Findings) == 0 {
					fmt.Printf("%s: no problems found.\n", args[0])
				} else {
					fmt.Printf("%s\n", args[0])
					for _, f := range result.Findings {
						fmt.Printf("  %s\n", f)
					}
					fmt.Printf("%d error(s), %d warning(s)\n",
						result.Errors(), len(result.Findings)-result.Errors())
				}
			}

			if validateExitCode && !result.OK() {
				return diffFound()
			}
			return nil
		},
	}
}

func addValidateFlags(c *cobra.Command) {
	c.Flags().BoolVar(&validateExitCode, "exit-code", false, "exit 1 when there are errors (2 on failure) — for CI gates")
}
