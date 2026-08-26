package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/carlosprados/og-cli/v2/internal/output"
	"github.com/carlosprados/og-cli/v2/internal/unwrap"
	"github.com/carlosprados/og-cli/v2/pkg/opengate"
	"github.com/spf13/cobra"
)

var provisionCmd = &cobra.Command{
	Use:     "provision",
	Aliases: []string{"pf"},
	Short:   "Manage OpenGate provision functions (provision processors)",
	Long: `Manage OpenGate provision functions — "provision processors" in the API (organization-scoped).

A provision function is a JavaScript script that transforms inbound rows (typically
from an Excel sheet) into provisioning actions: create/update/delete assets, devices,
subscriptions and subscribers. The script must implement two functions:
  normalizeRawObject(rawObject)   — validate + shape one inbound row
  actionsPlanning(normalizedObject) — return the array of ODM actions to apply

The code lives in the scriptProcessor.script field. The pull/wrap/deploy verbs mirror
the rules and connector lifecycle: pull a provision function into an editable directory
(provisionfunction.json + scriptProcessor__script.js), edit the JS locally with your
IDE, and deploy it back.

Execution closes the loop:
  plan  — dry-run the first N rows of an Excel file, returns the action plan as JSON
          WITHOUT mutating any data (ideal for iterating on a script)
  bulk  — run the full provisioning from an Excel file (returns a bulk process id)
  bulk-status / bulk-details — track a bulk run and download its result Excel

Examples:
  og provision list --org sensehat
  og provision pull <pp-id> --dir provision/
  og provision deploy provision/<pp-slug> --update
  og provision plan <pp-id> --file data.xlsx --rows 3
  og provision bulk <pp-id> --file data.xlsx
  og provision bulk-status <bulk-id>
  og provision bulk-details <bulk-id> --out result.xlsx`,
}

var (
	ppCreateFile   string
	ppUpdateFile   string
	ppPullDir      string
	ppPullForce    bool
	ppWrapOut      string
	ppDeployUpdate bool
	ppPlanFile     string
	ppPlanRows     int
	ppBulkFile     string
	ppDetailsOut   string
)

// --- list ---

var provisionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List provision functions in the organization",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, orgName, err := provisionClient()
		if err != nil {
			return err
		}
		items, err := c.ListProvisionProcessors(cmd.Context(), orgName)
		if err != nil {
			return err
		}

		return output.Print(outFmt, items,
			[]string{"Identifier", "Name", "Sheet", "HeaderRow", "ResultColumn"},
			func(data any) [][]string {
				pps := data.([]json.RawMessage)
				rows := make([][]string, len(pps))
				for i, raw := range pps {
					s := opengate.ParseProvisionProcessorSummary(raw)
					rows[i] = []string{s.ProvisionProcessorID, s.Name, s.SheetName, s.HeaderRow, s.ResultColumnName}
				}
				return rows
			},
		)
	},
}

// --- get ---

var provisionGetCmd = &cobra.Command{
	Use:   "get <pp-id>",
	Short: "Get a provision function by identifier",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, orgName, err := provisionClient()
		if err != nil {
			return err
		}
		data, err := c.GetProvisionProcessor(cmd.Context(), orgName, args[0])
		if err != nil {
			return err
		}
		return printJSON(data)
	},
}

// --- create / update / delete ---

var provisionCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a provision function from a JSON file",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, orgName, err := provisionClient()
		if err != nil {
			return err
		}
		body, err := os.ReadFile(ppCreateFile)
		if err != nil {
			return err
		}
		if _, err := c.CreateProvisionProcessor(cmd.Context(), orgName, body); err != nil {
			return err
		}
		fmt.Println("Provision function created successfully.")
		return nil
	},
}

var provisionUpdateCmd = &cobra.Command{
	Use:   "update <pp-id>",
	Short: "Update a provision function from a JSON file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, orgName, err := provisionClient()
		if err != nil {
			return err
		}
		body, err := os.ReadFile(ppUpdateFile)
		if err != nil {
			return err
		}
		if err := c.UpdateProvisionProcessor(cmd.Context(), orgName, args[0], body); err != nil {
			return err
		}
		fmt.Println("Provision function updated successfully.")
		return nil
	},
}

var provisionDeleteCmd = &cobra.Command{
	Use:   "delete <pp-id>",
	Short: "Delete a provision function",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := confirmDestructive(fmt.Sprintf("delete provision function %q", args[0])); err != nil {
			return err
		}
		c, orgName, err := provisionClient()
		if err != nil {
			return err
		}
		if err := c.DeleteProvisionProcessor(cmd.Context(), orgName, args[0]); err != nil {
			return err
		}
		fmt.Println("Provision function deleted successfully.")
		return nil
	},
}

// --- pull / pull-all / wrap / deploy ---

var provisionPullCmd = &cobra.Command{
	Use:     "pull <pp-id>",
	Aliases: []string{"unwrap"},
	Short:   "Pull a provision function into an editable directory (provisionfunction.json + scriptProcessor__script.js)",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, orgName, err := provisionClient()
		if err != nil {
			return err
		}
		raw, err := c.GetProvisionProcessor(cmd.Context(), orgName, args[0])
		if err != nil {
			return err
		}
		dir, err := unwrapArtifactTo(unwrap.ProvisionFunctionDescriptor(), raw, ppPullDir, &unwrap.Options{Force: ppPullForce, Warn: hintWarner()})
		if err != nil {
			return err
		}
		fmt.Printf("Provision function unwrapped to %s\n", dir)

		if p, perr := activeProfile(); perr == nil {
			d := unwrap.ProvisionFunctionDescriptor()
			recordBase(d.Kind, opengate.ParseProvisionProcessorSummary(raw).ProvisionProcessorID, d.NameOf(raw),
				dir, ppPullDir, raw, syncTarget(p, orgName, ""))
		}
		return nil
	},
}

var provisionPullAllCmd = &cobra.Command{
	Use:     "pull-all",
	Aliases: []string{"unwrap-all"},
	Short:   "Pull every provision function of the organization into <dir>/<pp-slug>/",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, orgName, err := provisionClient()
		if err != nil {
			return err
		}
		items, err := c.ListProvisionProcessors(cmd.Context(), orgName)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			fmt.Println("No provision functions found.")
			return nil
		}
		// One Options for the whole batch: slug deduplication only works when
		// every artifact sees the slugs its siblings already claimed.
		opts := &unwrap.Options{Force: ppPullForce, Warn: hintWarner()}
		count := 0
		p, _ := activeProfile()
		for _, raw := range items {
			dir, err := unwrapArtifactTo(unwrap.ProvisionFunctionDescriptor(), raw, ppPullDir, opts)
			if err != nil {
				return err
			}
			fmt.Printf("  %s\n", dir)
			if p != nil {
				d := unwrap.ProvisionFunctionDescriptor()
				recordBase(d.Kind, opengate.ParseProvisionProcessorSummary(raw).ProvisionProcessorID, d.NameOf(raw),
					dir, ppPullDir, raw, syncTarget(p, orgName, ""))
			}
			count++
		}
		fmt.Printf("%d provision functions unwrapped to %s\n", count, ppPullDir)
		return nil
	},
}

var provisionWrapCmd = &cobra.Command{
	Use:   "wrap <pp-dir>",
	Short: "Rebuild a provision function JSON from an unwrapped directory (no upload)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := unwrap.WrapProvisionProcessor(args[0], hintWarner())
		if err != nil {
			return err
		}
		if ppWrapOut != "" {
			if err := os.WriteFile(ppWrapOut, data, 0o644); err != nil {
				return err
			}
			fmt.Printf("Provision function written to %s\n", ppWrapOut)
			return nil
		}
		fmt.Println(string(data))
		return nil
	},
}

var provisionDeployCmd = &cobra.Command{
	Use:   "deploy <pp-dir>",
	Short: "Wrap + upload a provision function in one step (POST, or PUT with --update)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, orgName, err := provisionClient()
		if err != nil {
			return err
		}
		body, err := unwrap.WrapProvisionProcessor(args[0], hintWarner())
		if err != nil {
			return err
		}
		if p, perr := activeProfile(); perr == nil {
			warnIfMovedTarget(args[0], syncTarget(p, orgName, ""))
		}

		if ppDeployUpdate {
			var pp struct {
				ProvisionProcessorID string `json:"provisionProcessorId"`
			}
			if err := json.Unmarshal(body, &pp); err != nil || pp.ProvisionProcessorID == "" {
				return fmt.Errorf("--update requires a 'provisionProcessorId' field in provisionfunction.json (pull the provision function first)")
			}
			if err := c.UpdateProvisionProcessor(cmd.Context(), orgName, pp.ProvisionProcessorID, body); err != nil {
				return err
			}
			fmt.Printf("Provision function %s updated successfully.\n", pp.ProvisionProcessorID)
			return nil
		}

		if _, err := c.CreateProvisionProcessor(cmd.Context(), orgName, body); err != nil {
			return err
		}
		fmt.Println("Provision function created successfully.")
		return nil
	},
}

// --- plan / bulk / bulk-status / bulk-details ---

var provisionPlanCmd = &cobra.Command{
	Use:   "plan <pp-id>",
	Short: "Dry-run a provision function against the first N rows of an Excel file (no data mutated)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, orgName, err := provisionClient()
		if err != nil {
			return err
		}
		data, err := c.PlanProvisionBulk(cmd.Context(), orgName, args[0], ppPlanFile, ppPlanRows)
		if err != nil {
			return err
		}
		return printJSON(data)
	},
}

var provisionBulkCmd = &cobra.Command{
	Use:   "bulk <pp-id>",
	Short: "Execute a full provisioning bulk from an Excel file (returns the bulk process id)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, orgName, err := provisionClient()
		if err != nil {
			return err
		}
		bulkID, err := c.RunProvisionBulk(cmd.Context(), orgName, args[0], ppBulkFile)
		if err != nil {
			return err
		}
		fmt.Printf("Bulk process started: %s\n", bulkID)
		fmt.Printf("Track it with: og provision bulk-status %s\n", bulkID)
		return nil
	},
}

var provisionBulkStatusCmd = &cobra.Command{
	Use:   "bulk-status <bulk-id>",
	Short: "Read the status summary of a bulk process",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, orgName, err := provisionClient()
		if err != nil {
			return err
		}
		data, err := c.GetProvisionBulkStatus(cmd.Context(), orgName, args[0])
		if err != nil {
			return err
		}
		return printJSON(data)
	},
}

var provisionBulkDetailsCmd = &cobra.Command{
	Use:   "bulk-details <bulk-id>",
	Short: "Download the result Excel of a finished bulk process",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, orgName, err := provisionClient()
		if err != nil {
			return err
		}
		data, ready, err := c.GetProvisionBulkDetails(cmd.Context(), orgName, args[0])
		if err != nil {
			return err
		}
		if !ready {
			fmt.Println("Bulk process not finished yet — no details available (try again later).")
			return nil
		}
		out := ppDetailsOut
		if out == "" {
			out = args[0] + ".xlsx"
		}
		if err := os.WriteFile(out, data, 0o644); err != nil {
			return err
		}
		fmt.Printf("Bulk details written to %s\n", out)
		return nil
	},
}

// --- helpers ---

func provisionClient() (*opengate.Client, string, error) {
	p, err := activeProfile()
	if err != nil {
		return nil, "", err
	}
	orgName, err := resolveOrg(p)
	if err != nil {
		return nil, "", err
	}
	return opengate.New(p.Host, p.Token, p.ClientOptions()...), orgName, nil
}

// --- init ---

var provisionDiffCmd = newDiffCmd(diffSpec{
	Descriptor: unwrap.ProvisionFunctionDescriptor(),
	Use:        "diff <pf-dir>",
	Short:      "Compare a local provision function directory against the platform",
	Long: `Compare a locally-edited provision function against the one on the platform.

Metadata is reported structurally and the script textually. See
'og rules diff --help' for the state markers.

Worth running before every deploy here: a bad provision function corrupts entity
data at bulk scale.

Examples:
  og provision diff provision/createUpdate --org sensehat
  og provision diff provision/createUpdate --against production
  og provision diff provision/createUpdate --exit-code -o json`,
	Fetch: func(ctx context.Context, c *opengate.Client, org, id string) (json.RawMessage, error) {
		return c.GetProvisionProcessor(ctx, org, id)
	},
})

func init() {
	provisionCmd.AddCommand(provisionDiffCmd)
	addDiffFlags(provisionDiffCmd)
	provisionCreateCmd.Flags().StringVarP(&ppCreateFile, "file", "f", "", "path to JSON file with provision function definition")
	provisionCreateCmd.MarkFlagRequired("file")

	provisionUpdateCmd.Flags().StringVarP(&ppUpdateFile, "file", "f", "", "path to JSON file with provision function definition")
	provisionUpdateCmd.MarkFlagRequired("file")

	provisionPullCmd.Flags().StringVar(&ppPullDir, "dir", "provision", "destination directory")
	provisionPullCmd.Flags().BoolVar(&ppPullForce, "force", false, "overwrite existing destination")
	provisionPullAllCmd.Flags().StringVar(&ppPullDir, "dir", "provision", "destination directory")
	provisionPullAllCmd.Flags().BoolVar(&ppPullForce, "force", false, "overwrite existing destinations")

	provisionWrapCmd.Flags().StringVar(&ppWrapOut, "out", "", "write provision function JSON to this file (default: stdout)")

	provisionDeployCmd.Flags().BoolVar(&ppDeployUpdate, "update", false, "update an existing provision function (PUT) instead of creating (POST)")

	provisionPlanCmd.Flags().StringVar(&ppPlanFile, "file", "", "path to the Excel file (.xlsx/.xls) with the rows to plan")
	provisionPlanCmd.Flags().IntVar(&ppPlanRows, "rows", 1, "number of entries (rows) to plan")
	provisionPlanCmd.MarkFlagRequired("file")

	provisionBulkCmd.Flags().StringVar(&ppBulkFile, "file", "", "path to the Excel file (.xlsx/.xls) with the rows to provision")
	provisionBulkCmd.MarkFlagRequired("file")

	provisionBulkDetailsCmd.Flags().StringVar(&ppDetailsOut, "out", "", "write the result Excel to this file (default: <bulk-id>.xlsx)")

	provisionCmd.AddCommand(provisionListCmd)
	provisionCmd.AddCommand(provisionGetCmd)
	provisionCmd.AddCommand(provisionCreateCmd)
	provisionCmd.AddCommand(provisionUpdateCmd)
	provisionCmd.AddCommand(provisionDeleteCmd)
	provisionCmd.AddCommand(provisionPullCmd)
	provisionCmd.AddCommand(provisionPullAllCmd)
	provisionCmd.AddCommand(provisionWrapCmd)
	provisionCmd.AddCommand(provisionDeployCmd)
	provisionCmd.AddCommand(provisionPlanCmd)
	provisionCmd.AddCommand(provisionBulkCmd)
	provisionCmd.AddCommand(provisionBulkStatusCmd)
	provisionCmd.AddCommand(provisionBulkDetailsCmd)

	rootCmd.AddCommand(provisionCmd)
}
