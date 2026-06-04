package cmd

import (
	"fmt"

	"github.com/carlosprados/og-cli/internal/output"
	"github.com/carlosprados/og-cli/internal/views"
	"github.com/spf13/cobra"
)

var viewsCmd = &cobra.Command{
	Use:   "views",
	Short: "Manage named field views for searches",
	Long: `Views are named field sets that expand into select clauses, so you can ask
for intent instead of memorizing datastream paths:

  og devices search --view summary,power

Built-in views ship with the binary. Add your own in ~/.og/views/*.yaml
(global) or ./.og/views/*.yaml (per project); project views override user
views, which override built-ins.`,
}

var viewsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available views",
	RunE:  runViewsList,
}

var viewsShowCmd = &cobra.Command{
	Use:   "show NAME",
	Short: "Show the fields a view expands to",
	Args:  cobra.ExactArgs(1),
	RunE:  runViewsShow,
}

func runViewsList(cmd *cobra.Command, args []string) error {
	reg, err := views.Load()
	if err != nil {
		return err
	}

	all := reg.All()
	return output.Print(outFmt, all,
		[]string{"Name", "Source", "Description"},
		func(data any) [][]string {
			defs := data.([]views.Definition)
			rows := make([][]string, len(defs))
			for i, d := range defs {
				rows[i] = []string{d.Name, d.Source, d.Description}
			}
			return rows
		},
	)
}

func runViewsShow(cmd *cobra.Command, args []string) error {
	reg, err := views.Load()
	if err != nil {
		return err
	}

	view, ok := reg.Get(args[0])
	if !ok {
		// Reuse the not-found error with typo suggestion.
		_, err := reg.ResolveSelect(args[0:1], nil)
		return err
	}

	fmt.Printf("%s — %s  [%s]\n\n", view.Name, view.Description, view.Source)

	return output.Print(outFmt, view.Fields,
		[]string{"Datastream", "Fields", "Alias"},
		func(data any) [][]string {
			fields := data.([]views.Field)
			rows := make([][]string, len(fields))
			for i, f := range fields {
				projected := "value"
				if f.At {
					projected = "value, at"
				}
				rows[i] = []string{f.Name, projected, f.Alias}
			}
			return rows
		},
	)
}

func init() {
	viewsCmd.AddCommand(viewsListCmd)
	viewsCmd.AddCommand(viewsShowCmd)
	rootCmd.AddCommand(viewsCmd)
}
