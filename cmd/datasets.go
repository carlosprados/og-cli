package cmd

import (
	"fmt"
	"os"

	"github.com/carlosprados/og-cli/internal/client"
	"github.com/carlosprados/og-cli/internal/output"
	"github.com/spf13/cobra"
)

var datasetsCmd = &cobra.Command{
	Use:     "datasets",
	Aliases: []string{"ds"},
	Short:   "Manage OpenGate datasets",
}

// --- list ---

var dsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List datasets in the organization",
	RunE:  runDSList,
}

func runDSList(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	orgName, err := resolveOrg(p)
	if err != nil {
		return err
	}
	c := client.New(p.Host, p.Token)

	resp, err := c.ListDatasets(orgName)
	if err != nil {
		return err
	}

	return output.Print(outFmt, resp.Datasets,
		[]string{"Identifier", "Name", "Description", "Columns"},
		func(data any) [][]string {
			dsList := data.([]client.Dataset)
			rows := make([][]string, len(dsList))
			for i, ds := range dsList {
				rows[i] = []string{ds.Identifier, ds.Name, ds.Description, fmt.Sprintf("%d", len(ds.Columns))}
			}
			return rows
		},
	)
}

// --- get ---

var dsGetCmd = &cobra.Command{
	Use:   "get <dataset-id>",
	Short: "Get a dataset definition",
	Args:  cobra.ExactArgs(1),
	RunE:  runDSGet,
}

func runDSGet(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	orgName, err := resolveOrg(p)
	if err != nil {
		return err
	}
	c := client.New(p.Host, p.Token)

	ds, err := c.GetDataset(orgName, args[0])
	if err != nil {
		return err
	}

	if outFmt == output.FormatJSON {
		return output.PrintJSON(os.Stdout, ds)
	}

	fmt.Printf("%s — %s\n\n", ds.Name, ds.Description)

	return output.Print(outFmt, ds.Columns,
		[]string{"Name", "Path", "Filter", "Sort"},
		func(data any) [][]string {
			cols := data.([]client.DSColumn)
			rows := make([][]string, len(cols))
			for i, c := range cols {
				rows[i] = []string{c.Name, c.Path, c.Filter, fmt.Sprintf("%v", c.Sort)}
			}
			return rows
		},
	)
}

// --- create ---

var dsCreateCmd = &cobra.Command{
	Use:   "create -f <file.json>",
	Short: "Create a new dataset from a JSON file",
	RunE:  runDSCreate,
}

var dsCreateFile string

func runDSCreate(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	orgName, err := resolveOrg(p)
	if err != nil {
		return err
	}

	body, err := os.ReadFile(dsCreateFile)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	c := client.New(p.Host, p.Token)
	if err := c.CreateDataset(orgName, body); err != nil {
		return err
	}

	fmt.Println("Dataset created successfully.")
	return nil
}

// --- update ---

var dsUpdateCmd = &cobra.Command{
	Use:   "update <dataset-id> -f <file.json>",
	Short: "Update a dataset from a JSON file",
	Args:  cobra.ExactArgs(1),
	RunE:  runDSUpdate,
}

var dsUpdateFile string

func runDSUpdate(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	orgName, err := resolveOrg(p)
	if err != nil {
		return err
	}

	body, err := os.ReadFile(dsUpdateFile)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	c := client.New(p.Host, p.Token)
	if err := c.UpdateDataset(orgName, args[0], body); err != nil {
		return err
	}

	fmt.Println("Dataset updated successfully.")
	return nil
}

// --- delete ---

var dsDeleteCmd = &cobra.Command{
	Use:   "delete <dataset-id>",
	Short: "Delete a dataset",
	Args:  cobra.ExactArgs(1),
	RunE:  runDSDelete,
}

func runDSDelete(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	orgName, err := resolveOrg(p)
	if err != nil {
		return err
	}

	c := client.New(p.Host, p.Token)
	if err := c.DeleteDataset(orgName, args[0]); err != nil {
		return err
	}

	fmt.Println("Dataset deleted successfully.")
	return nil
}

// --- data ---

var dsDataCmd = &cobra.Command{
	Use:   "data <dataset-id>",
	Short: "Query data from a dataset",
	Long: `Query data from a dataset with optional filtering.

Examples:
  og ds data <id>
  og ds data <id> -w "Prov Identifier eq MyDevice1"
  og ds data <id> --limit 20`,
	Args: cobra.ExactArgs(1),
	RunE: runDSData,
}

var (
	dsDataFilter string
	dsDataWhere  []string
	dsDataLimit  int
)

func runDSData(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	orgName, err := resolveOrg(p)
	if err != nil {
		return err
	}
	c := client.New(p.Host, p.Token)

	filter, err := buildSearchFilter(dsDataWhere, dsDataLimit, nil, dsDataFilter)
	if err != nil {
		return err
	}

	resp, err := c.QueryDatasetData(orgName, args[0], filter)
	if err != nil {
		return err
	}

	if outFmt == output.FormatJSON {
		return output.PrintJSON(os.Stdout, resp)
	}

	if len(resp.Columns) == 0 {
		fmt.Println("No data.")
		return nil
	}

	rows := make([][]string, len(resp.Data))
	for i, row := range resp.Data {
		cells := make([]string, len(row))
		for j, val := range row {
			cells[j] = fmt.Sprintf("%v", val)
		}
		rows[i] = cells
	}

	output.PrintTable(os.Stdout, resp.Columns, rows)
	return nil
}

// --- init ---

func init() {
	dsDataCmd.Flags().StringArrayVarP(&dsDataWhere, "where", "w", nil, `filter condition: "column op value" (repeatable)`)
	dsDataCmd.Flags().IntVar(&dsDataLimit, "limit", 0, "max number of rows")
	dsDataCmd.Flags().StringVar(&dsDataFilter, "filter", "", "raw search filter as JSON (overrides -w)")

	dsCreateCmd.Flags().StringVarP(&dsCreateFile, "file", "f", "", "path to JSON file with dataset definition")
	dsCreateCmd.MarkFlagRequired("file")

	dsUpdateCmd.Flags().StringVarP(&dsUpdateFile, "file", "f", "", "path to JSON file with dataset definition")
	dsUpdateCmd.MarkFlagRequired("file")

	datasetsCmd.AddCommand(dsListCmd)
	datasetsCmd.AddCommand(dsGetCmd)
	datasetsCmd.AddCommand(dsCreateCmd)
	datasetsCmd.AddCommand(dsUpdateCmd)
	datasetsCmd.AddCommand(dsDeleteCmd)
	datasetsCmd.AddCommand(dsDataCmd)

	rootCmd.AddCommand(datasetsCmd)
}
