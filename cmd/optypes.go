package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/carlosprados/og-cli/internal/output"
	"github.com/carlosprados/og-cli/pkg/opengate"
	"github.com/spf13/cobra"
)

var optypesCmd = &cobra.Command{
	Use:     "optypes",
	Aliases: []string{"operation-types"},
	Short:   "Manage OpenGate operation type definitions",
	Long: `Manage operation TYPE definitions (the catalog of operations that jobs can launch).

This is the DDL side of operations: define a new operation with its parameters
schema, then launch it on devices with 'og jobs create'.

Examples:
  og optypes catalog
  og optypes search
  og optypes get CALIBRATE_SENSOR --org sensehat
  og optypes create --org sensehat -f optype.json
  og optypes delete CALIBRATE_SENSOR --org sensehat`,
}

var (
	optypeCreateFile string
	optypeUpdateFile string
)

var optypesCatalogCmd = &cobra.Command{
	Use:   "catalog",
	Short: "List the catalog of predefined operation types",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := activeProfile()
		if err != nil {
			return err
		}
		c := opengate.New(p.Host, p.Token)
		data, err := c.OpTypesCatalog(cmd.Context())
		if err != nil {
			return err
		}
		return printOpTypeList(data)
	},
}

var optypesSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search operation types",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := activeProfile()
		if err != nil {
			return err
		}
		c := opengate.New(p.Host, p.Token)
		data, err := c.SearchOpTypes(cmd.Context(), nil)
		if err != nil {
			return err
		}
		return printOpTypeList(data)
	},
}

var optypesGetCmd = &cobra.Command{
	Use:   "get <operation-name>",
	Short: "Get an operation type definition",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, orgName, err := rulesClient()
		if err != nil {
			return err
		}
		data, err := c.GetOpType(cmd.Context(), orgName, args[0])
		if err != nil {
			return err
		}
		return printJSON(data)
	},
}

var optypesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an operation type from a JSON file",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, orgName, err := rulesClient()
		if err != nil {
			return err
		}
		body, err := os.ReadFile(optypeCreateFile)
		if err != nil {
			return err
		}
		if _, err := c.CreateOpType(cmd.Context(), orgName, body); err != nil {
			return err
		}
		fmt.Println("Operation type created successfully.")
		return nil
	},
}

var optypesUpdateCmd = &cobra.Command{
	Use:   "update <operation-name>",
	Short: "Update an operation type from a JSON file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, orgName, err := rulesClient()
		if err != nil {
			return err
		}
		body, err := os.ReadFile(optypeUpdateFile)
		if err != nil {
			return err
		}
		if err := c.UpdateOpType(cmd.Context(), orgName, args[0], body); err != nil {
			return err
		}
		fmt.Println("Operation type updated successfully.")
		return nil
	},
}

var optypesDeleteCmd = &cobra.Command{
	Use:   "delete <operation-name>",
	Short: "Delete an operation type",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := confirmDestructive(fmt.Sprintf("delete operation type %q", args[0])); err != nil {
			return err
		}
		c, orgName, err := rulesClient()
		if err != nil {
			return err
		}
		if err := c.DeleteOpType(cmd.Context(), orgName, args[0]); err != nil {
			return err
		}
		fmt.Println("Operation type deleted successfully.")
		return nil
	},
}

// printOpTypeList renders an operation type list response as a table. The API
// wraps results in different envelopes (catalog vs search), so try the known
// shapes before falling back to raw JSON.
func printOpTypeList(data json.RawMessage) error {
	var list []json.RawMessage
	if json.Unmarshal(data, &list) != nil {
		// Not a bare array — try known envelope keys.
		var envelope map[string]json.RawMessage
		if err := json.Unmarshal(data, &envelope); err != nil {
			return printJSON(data)
		}
		for _, key := range []string{"operations", "operationTypes", "catalog"} {
			if raw, ok := envelope[key]; ok {
				if json.Unmarshal(raw, &list) == nil {
					break
				}
			}
		}
	}
	if list == nil {
		return printJSON(data)
	}

	return output.Print(outFmt, list,
		[]string{"Name", "Title", "ApplicableTo", "Description"},
		func(d any) [][]string {
			items := d.([]json.RawMessage)
			rows := make([][]string, len(items))
			for i, raw := range items {
				s := opengate.ParseOpTypeSummary(raw)
				rows[i] = []string{s.Name, s.Title, strings.Join(s.ApplicableTo, ","), s.Description}
			}
			return rows
		},
	)
}

func init() {
	optypesCreateCmd.Flags().StringVarP(&optypeCreateFile, "file", "f", "", "path to JSON file with operation type definition")
	optypesCreateCmd.MarkFlagRequired("file")

	optypesUpdateCmd.Flags().StringVarP(&optypeUpdateFile, "file", "f", "", "path to JSON file with operation type definition")
	optypesUpdateCmd.MarkFlagRequired("file")

	optypesCmd.AddCommand(optypesCatalogCmd)
	optypesCmd.AddCommand(optypesSearchCmd)
	optypesCmd.AddCommand(optypesGetCmd)
	optypesCmd.AddCommand(optypesCreateCmd)
	optypesCmd.AddCommand(optypesUpdateCmd)
	optypesCmd.AddCommand(optypesDeleteCmd)

	rootCmd.AddCommand(optypesCmd)
}
