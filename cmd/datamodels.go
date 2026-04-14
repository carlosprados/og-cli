package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/carlosprados/og-cli/internal/client"
	"github.com/carlosprados/og-cli/internal/output"
	"github.com/spf13/cobra"
)

var datamodelsCmd = &cobra.Command{
	Use:     "datamodels",
	Aliases: []string{"dm"},
	Short:   "Manage OpenGate data models",
}

// --- search ---

var datamodelsSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search data models",
	Long: `Search data models with simple conditions or a raw JSON filter.

Examples:
  og dm search -w "datamodels.identifier like teliot"
  og dm search -w "datamodels.organizationName eq sensehat" --limit 5
  og dm search --filter '{"filter":{"or":[...]}}'`,
	RunE: runDatamodelsSearch,
}

var (
	dmSearchFilter string
	dmSearchWhere  []string
	dmSearchLimit  int
)

func runDatamodelsSearch(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	c := client.New(p.Host, p.Token)

	filter, err := buildSearchFilter(dmSearchWhere, dmSearchLimit, nil, dmSearchFilter)
	if err != nil {
		return err
	}

	resp, err := c.SearchDatamodels(filter)
	if err != nil {
		return err
	}

	return output.Print(outFmt, resp.Datamodels,
		[]string{"Identifier", "Organization", "Name", "Version"},
		func(data any) [][]string {
			dms := data.([]client.Datamodel)
			rows := make([][]string, len(dms))
			for i, dm := range dms {
				rows[i] = []string{dm.Identifier, dm.OrganizationName, dm.Name, dm.Version}
			}
			return rows
		},
	)
}

// --- get ---

var datamodelsGetCmd = &cobra.Command{
	Use:   "get <datamodel-id>",
	Short: "Get a data model by identifier",
	Args:  cobra.ExactArgs(1),
	RunE:  runDatamodelsGet,
}

func runDatamodelsGet(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	orgName, err := resolveOrg(p)
	if err != nil {
		return err
	}
	c := client.New(p.Host, p.Token)

	dm, err := c.GetDatamodel(orgName, args[0])
	if err != nil {
		return err
	}

	return output.Print(outFmt, dm,
		[]string{"Category", "Datastream", "Name", "Period", "Schema", "Access"},
		func(data any) [][]string {
			d := data.(*client.Datamodel)
			var rows [][]string
			for _, cat := range d.Categories {
				if len(cat.Datastreams) == 0 {
					rows = append(rows, []string{cat.Identifier, "", "", "", "", ""})
					continue
				}
				for _, ds := range cat.Datastreams {
					schemaType := schemaTypeString(ds.Schema)
					rows = append(rows, []string{
						cat.Identifier,
						ds.Identifier,
						ds.Name,
						ds.Period,
						schemaType,
						ds.Access,
					})
				}
			}
			return rows
		},
	)
}

func schemaTypeString(schema json.RawMessage) string {
	if schema == nil {
		return ""
	}
	var s struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(schema, &s) == nil && s.Type != "" {
		return s.Type
	}
	return string(schema)
}

// --- create ---

var datamodelsCreateCmd = &cobra.Command{
	Use:   "create -f <file.json>",
	Short: "Create a new data model from a JSON file",
	RunE:  runDatamodelsCreate,
}

var createFile string

func runDatamodelsCreate(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	orgName, err := resolveOrg(p)
	if err != nil {
		return err
	}

	body, err := os.ReadFile(createFile)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	c := client.New(p.Host, p.Token)
	if err := c.CreateDatamodel(orgName, body); err != nil {
		return err
	}

	fmt.Println("Datamodel created successfully.")
	return nil
}

// --- update ---

var datamodelsUpdateCmd = &cobra.Command{
	Use:   "update <datamodel-id> -f <file.json>",
	Short: "Update an existing data model from a JSON file",
	Args:  cobra.ExactArgs(1),
	RunE:  runDatamodelsUpdate,
}

var updateFile string

func runDatamodelsUpdate(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	orgName, err := resolveOrg(p)
	if err != nil {
		return err
	}

	body, err := os.ReadFile(updateFile)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	c := client.New(p.Host, p.Token)
	if err := c.UpdateDatamodel(orgName, args[0], body); err != nil {
		return err
	}

	fmt.Println("Datamodel updated successfully.")
	return nil
}

// --- delete ---

var datamodelsDeleteCmd = &cobra.Command{
	Use:   "delete <datamodel-id>",
	Short: "Delete a data model",
	Args:  cobra.ExactArgs(1),
	RunE:  runDatamodelsDelete,
}

func runDatamodelsDelete(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	orgName, err := resolveOrg(p)
	if err != nil {
		return err
	}

	c := client.New(p.Host, p.Token)
	if err := c.DeleteDatamodel(orgName, args[0]); err != nil {
		return err
	}

	fmt.Println("Datamodel deleted successfully.")
	return nil
}

// --- init ---

func init() {
	datamodelsSearchCmd.Flags().StringArrayVarP(&dmSearchWhere, "where", "w", nil, `filter condition: "field op value" (repeatable)`)
	datamodelsSearchCmd.Flags().IntVar(&dmSearchLimit, "limit", 0, "max number of results")
	datamodelsSearchCmd.Flags().StringVar(&dmSearchFilter, "filter", "", "raw search filter as JSON (overrides -w)")

	datamodelsCreateCmd.Flags().StringVarP(&createFile, "file", "f", "", "path to JSON file with datamodel definition")
	datamodelsCreateCmd.MarkFlagRequired("file")

	datamodelsUpdateCmd.Flags().StringVarP(&updateFile, "file", "f", "", "path to JSON file with datamodel definition")
	datamodelsUpdateCmd.MarkFlagRequired("file")

	datamodelsCmd.AddCommand(datamodelsSearchCmd)
	datamodelsCmd.AddCommand(datamodelsGetCmd)
	datamodelsCmd.AddCommand(datamodelsCreateCmd)
	datamodelsCmd.AddCommand(datamodelsUpdateCmd)
	datamodelsCmd.AddCommand(datamodelsDeleteCmd)

	rootCmd.AddCommand(datamodelsCmd)
}
