package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/carlosprados/og-cli/internal/client"
	"github.com/carlosprados/og-cli/internal/output"
	"github.com/spf13/cobra"
)

var timeseriesCmd = &cobra.Command{
	Use:     "timeseries",
	Aliases: []string{"ts"},
	Short:   "Manage OpenGate time series",
}

// --- list ---

var tsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List time series in the organization",
	RunE:  runTSList,
}

func runTSList(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	orgName, err := resolveOrg(p)
	if err != nil {
		return err
	}
	c := client.New(p.Host, p.Token)

	resp, err := c.ListTimeSeries(orgName)
	if err != nil {
		return err
	}

	return output.Print(outFmt, resp.Timeseries,
		[]string{"Identifier", "Name", "Bucket(s)", "Retention(s)", "Columns"},
		func(data any) [][]string {
			tsList := data.([]client.TimeSeries)
			rows := make([][]string, len(tsList))
			for i, ts := range tsList {
				rows[i] = []string{
					ts.Identifier,
					ts.Name,
					fmt.Sprintf("%d", ts.TimeBucket),
					fmt.Sprintf("%d", ts.Retention),
					fmt.Sprintf("%d", len(ts.Columns)),
				}
			}
			return rows
		},
	)
}

// --- get ---

var tsGetCmd = &cobra.Command{
	Use:   "get <timeseries-id>",
	Short: "Get a time series definition",
	Args:  cobra.ExactArgs(1),
	RunE:  runTSGet,
}

func runTSGet(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	orgName, err := resolveOrg(p)
	if err != nil {
		return err
	}
	c := client.New(p.Host, p.Token)

	ts, err := c.GetTimeSeries(orgName, args[0])
	if err != nil {
		return err
	}

	if outFmt == output.FormatJSON {
		return output.PrintJSON(os.Stdout, ts)
	}

	// Show columns as table
	fmt.Printf("%s — %s\n", ts.Name, ts.Description)
	fmt.Printf("Bucket: %ds, Retention: %ds, Origin: %s\n\n", ts.TimeBucket, ts.Retention, ts.Origin)

	if len(ts.Context) > 0 {
		fmt.Println("Context:")
		return output.Print(outFmt, ts.Context,
			[]string{"Name", "Path", "Filter"},
			func(data any) [][]string {
				cols := data.([]client.TSColumn)
				rows := make([][]string, len(cols))
				for i, c := range cols {
					rows[i] = []string{c.Name, c.Path, c.Filter}
				}
				return rows
			},
		)
	}

	if len(ts.Columns) > 0 {
		fmt.Println("Columns:")
		return output.Print(outFmt, ts.Columns,
			[]string{"Name", "Path", "Aggregation", "Filter"},
			func(data any) [][]string {
				cols := data.([]client.TSColumn)
				rows := make([][]string, len(cols))
				for i, c := range cols {
					rows[i] = []string{c.Name, c.Path, c.AggregationFunction, c.Filter}
				}
				return rows
			},
		)
	}

	return nil
}

// --- create ---

var tsCreateCmd = &cobra.Command{
	Use:   "create -f <file.json>",
	Short: "Create a new time series from a JSON file",
	RunE:  runTSCreate,
}

var tsCreateFile string

func runTSCreate(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	orgName, err := resolveOrg(p)
	if err != nil {
		return err
	}

	body, err := os.ReadFile(tsCreateFile)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	c := client.New(p.Host, p.Token)
	if err := c.CreateTimeSeries(orgName, body); err != nil {
		return err
	}

	fmt.Println("Time series created successfully.")
	return nil
}

// --- update ---

var tsUpdateCmd = &cobra.Command{
	Use:   "update <timeseries-id> -f <file.json>",
	Short: "Update a time series from a JSON file",
	Args:  cobra.ExactArgs(1),
	RunE:  runTSUpdate,
}

var tsUpdateFile string

func runTSUpdate(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	orgName, err := resolveOrg(p)
	if err != nil {
		return err
	}

	body, err := os.ReadFile(tsUpdateFile)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	c := client.New(p.Host, p.Token)
	if err := c.UpdateTimeSeries(orgName, args[0], body); err != nil {
		return err
	}

	fmt.Println("Time series updated successfully.")
	return nil
}

// --- delete ---

var tsDeleteCmd = &cobra.Command{
	Use:   "delete <timeseries-id>",
	Short: "Delete a time series",
	Args:  cobra.ExactArgs(1),
	RunE:  runTSDelete,
}

func runTSDelete(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	orgName, err := resolveOrg(p)
	if err != nil {
		return err
	}

	c := client.New(p.Host, p.Token)
	if err := c.DeleteTimeSeries(orgName, args[0]); err != nil {
		return err
	}

	fmt.Println("Time series deleted successfully.")
	return nil
}

// --- data ---

var tsDataCmd = &cobra.Command{
	Use:   "data <timeseries-id>",
	Short: "Query data from a time series",
	Long: `Query collected data from a time series with optional filtering.

Examples:
  og ts data <id>
  og ts data <id> -w "Prov Identifier eq MyDevice1"
  og ts data <id> --limit 20
  og ts data <id> --sort EntityAscBucketDesc`,
	Args: cobra.ExactArgs(1),
	RunE: runTSData,
}

var (
	tsDataFilter string
	tsDataWhere  []string
	tsDataLimit  int
	tsDataSort   string
)

func runTSData(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	orgName, err := resolveOrg(p)
	if err != nil {
		return err
	}
	c := client.New(p.Host, p.Token)

	filter, err := buildSearchFilter(tsDataWhere, tsDataLimit, nil, tsDataFilter)
	if err != nil {
		return err
	}

	// Inject sort if provided
	if tsDataSort != "" && filter != nil {
		filter, err = injectSort(filter, tsDataSort)
		if err != nil {
			return err
		}
	} else if tsDataSort != "" {
		filter = json.RawMessage(fmt.Sprintf(`{"sort":"%s"}`, tsDataSort))
	}

	resp, err := c.QueryTimeSeriesData(orgName, args[0], filter)
	if err != nil {
		return err
	}

	if outFmt == output.FormatJSON {
		return output.PrintJSON(os.Stdout, resp)
	}

	// Dynamic table from columns + data
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

// --- export ---

var tsExportCmd = &cobra.Command{
	Use:   "export <timeseries-id>",
	Short: "Trigger Parquet export of a time series",
	Args:  cobra.ExactArgs(1),
	RunE:  runTSExport,
}

func runTSExport(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	orgName, err := resolveOrg(p)
	if err != nil {
		return err
	}

	c := client.New(p.Host, p.Token)
	if err := c.ExportTimeSeries(orgName, args[0], nil); err != nil {
		return err
	}

	fmt.Println("Export triggered successfully.")
	return nil
}

// injectSort adds a sort field to an existing filter JSON.
func injectSort(filter json.RawMessage, sort string) (json.RawMessage, error) {
	var m map[string]any
	if err := json.Unmarshal(filter, &m); err != nil {
		return nil, err
	}
	m["sort"] = sort
	return json.Marshal(m)
}

// --- init ---

func init() {
	tsDataCmd.Flags().StringArrayVarP(&tsDataWhere, "where", "w", nil, `filter condition: "column op value" (repeatable)`)
	tsDataCmd.Flags().IntVar(&tsDataLimit, "limit", 0, "max number of rows")
	tsDataCmd.Flags().StringVar(&tsDataSort, "sort", "", "sort identifier defined in the time series")
	tsDataCmd.Flags().StringVar(&tsDataFilter, "filter", "", "raw search filter as JSON (overrides -w)")

	tsCreateCmd.Flags().StringVarP(&tsCreateFile, "file", "f", "", "path to JSON file with time series definition")
	tsCreateCmd.MarkFlagRequired("file")

	tsUpdateCmd.Flags().StringVarP(&tsUpdateFile, "file", "f", "", "path to JSON file with time series definition")
	tsUpdateCmd.MarkFlagRequired("file")

	timeseriesCmd.AddCommand(tsListCmd)
	timeseriesCmd.AddCommand(tsGetCmd)
	timeseriesCmd.AddCommand(tsCreateCmd)
	timeseriesCmd.AddCommand(tsUpdateCmd)
	timeseriesCmd.AddCommand(tsDeleteCmd)
	timeseriesCmd.AddCommand(tsDataCmd)
	timeseriesCmd.AddCommand(tsExportCmd)

	rootCmd.AddCommand(timeseriesCmd)
}
