package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/carlosprados/og-cli/internal/output"
	"github.com/carlosprados/og-cli/internal/query"
	"github.com/carlosprados/og-cli/internal/views"
	"github.com/carlosprados/og-cli/pkg/opengate"
	"github.com/spf13/cobra"
)

var devicesCmd = &cobra.Command{
	Use:     "devices",
	Aliases: []string{"dev"},
	Short:   "Manage OpenGate devices",
}

// --- search ---

var devicesSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search devices",
	Long: `Search devices with simple conditions or a raw JSON filter.

Filters apply to BOTH provisioned metadata (provision.*) and the latest
collected value of any datastream (default or organization-specific).

Project fields with -s (append @at for the value timestamp) or with named
views (--view); see 'og views list'. Explicit -s fields win over view fields.

Examples:
  og dev search -w "provision.device.administrativeState eq ACTIVE"
  og dev search -w "provision.device.identifier like sense" --limit 10
  og dev search -w "provision.device.administrativeState eq ACTIVE" -w "provision.device.identifier like sense"
  og dev search -w "wt gt 20" -s wt@at -s provision.device.identifier
  og dev search -w "device.temperature.value gte 50" -w "provision.device.operationalStatus eq NORMAL"
  og dev search --view summary
  og dev search --view summary,power -s wt
  og dev search --filter '{"filter":{"or":[...]}}'`,
	RunE: runDevicesSearch,
}

var (
	devSearchFilter string
	devSearchWhere  []string
	devSearchLimit  int
	devSearchSelect []string
	devSearchView   []string
	devSearchAt     bool
)

func runDevicesSearch(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	c := opengate.New(p.Host, p.Token)

	selectClauses := query.SelectFromFields(devSearchSelect, devSearchAt)
	if len(devSearchView) > 0 {
		reg, err := views.Load()
		if err != nil {
			return err
		}
		selectClauses, err = reg.ResolveSelect(devSearchView, selectClauses)
		if err != nil {
			return err
		}
	}

	filter, err := buildSearchFilter(devSearchWhere, devSearchLimit, selectClauses, devSearchFilter)
	if err != nil {
		return err
	}

	resp, err := c.SearchDevices(filter)
	if err != nil {
		return err
	}

	if len(selectClauses) > 0 {
		return printSelectedDevices(resp.Devices, selectClauses)
	}

	return output.Print(outFmt, resp.Devices,
		[]string{"Identifier", "Name", "Organization", "State"},
		func(data any) [][]string {
			devices := data.([]json.RawMessage)
			rows := make([][]string, len(devices))
			for i, raw := range devices {
				s := opengate.ParseDeviceSummary(raw)
				rows[i] = []string{s.Identifier, s.Name, s.Org, s.Status}
			}
			return rows
		},
	)
}

// buildSearchFilter is shared by all search commands.
func buildSearchFilter(where []string, limit int, sel []query.SelectClause, rawFilter string) (json.RawMessage, error) {
	var conditions []query.Condition
	for _, w := range where {
		// ParseQuery supports "cond AND cond" inside one -w, matching the
		// MCP query parameter semantics.
		cs, err := query.ParseQuery(w)
		if err != nil {
			return nil, err
		}
		conditions = append(conditions, cs...)
	}
	return query.MergeWithRaw(query.SearchParams{
		Conditions: conditions,
		Limit:      limit,
		Select:     sel,
	}, rawFilter)
}

// printSelectedDevices renders devices with one dynamic column per selected
// sub-field (value, at) of each clause.
func printSelectedDevices(devices []json.RawMessage, clauses []query.SelectClause) error {
	type column struct {
		header string
		name   string // datastream name
		sub    string // "value" or "at"
	}
	var columns []column
	for _, c := range clauses {
		for _, f := range c.Fields {
			header := f.Alias
			if header == "" {
				header = query.FieldAlias(c.Name)
			}
			columns = append(columns, column{header: header, name: c.Name, sub: f.Field})
		}
	}

	headers := make([]string, len(columns))
	for i, col := range columns {
		headers[i] = col.header
	}

	return output.Print(outFmt, devices, headers,
		func(data any) [][]string {
			devs := data.([]json.RawMessage)
			rows := make([][]string, len(devs))
			for i, raw := range devs {
				row := make([]string, len(columns))
				for j, col := range columns {
					if col.sub == "at" {
						row[j] = opengate.ExtractFlatAt(raw, col.name)
					} else {
						row[j] = opengate.ExtractFlatValue(raw, col.name)
					}
				}
				rows[i] = row
			}
			return rows
		},
	)
}

// --- get ---

var devicesGetCmd = &cobra.Command{
	Use:   "get <device-id>",
	Short: "Get a device by identifier",
	Args:  cobra.ExactArgs(1),
	RunE:  runDevicesGet,
}

func runDevicesGet(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	orgName, err := resolveOrg(p)
	if err != nil {
		return err
	}
	c := opengate.New(p.Host, p.Token)

	data, err := c.GetDevice(orgName, args[0])
	if err != nil {
		return err
	}

	// Device JSON is complex (flattened format), always output as JSON for get
	if outFmt == output.FormatTable {
		s := opengate.ParseDeviceSummary(data)
		return output.Print(outFmt, s,
			[]string{"Identifier", "Name", "Organization", "State"},
			func(d any) [][]string {
				ds := d.(opengate.DeviceSummary)
				return [][]string{{ds.Identifier, ds.Name, ds.Org, ds.Status}}
			},
		)
	}

	var pretty json.RawMessage
	if json.Unmarshal(data, &pretty) == nil {
		return output.PrintJSON(os.Stdout, pretty)
	}
	fmt.Println(string(data))
	return nil
}

// --- create ---

var devicesCreateCmd = &cobra.Command{
	Use:   "create -f <file.json>",
	Short: "Create a new device from a JSON file",
	RunE:  runDevicesCreate,
}

var devCreateFile string

func runDevicesCreate(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	orgName, err := resolveOrg(p)
	if err != nil {
		return err
	}

	body, err := os.ReadFile(devCreateFile)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	c := opengate.New(p.Host, p.Token)
	if err := c.CreateDevice(orgName, body); err != nil {
		return err
	}

	fmt.Println("Device created successfully.")
	return nil
}

// --- update ---

var devicesUpdateCmd = &cobra.Command{
	Use:   "update <device-id> -f <file.json>",
	Short: "Update an existing device from a JSON file",
	Args:  cobra.ExactArgs(1),
	RunE:  runDevicesUpdate,
}

var devUpdateFile string

func runDevicesUpdate(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	orgName, err := resolveOrg(p)
	if err != nil {
		return err
	}

	body, err := os.ReadFile(devUpdateFile)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	c := opengate.New(p.Host, p.Token)
	if err := c.UpdateDevice(orgName, args[0], body); err != nil {
		return err
	}

	fmt.Println("Device updated successfully.")
	return nil
}

// --- delete ---

var devicesDeleteCmd = &cobra.Command{
	Use:   "delete <device-id>",
	Short: "Delete a device",
	Args:  cobra.ExactArgs(1),
	RunE:  runDevicesDelete,
}

func runDevicesDelete(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	orgName, err := resolveOrg(p)
	if err != nil {
		return err
	}

	c := opengate.New(p.Host, p.Token)
	if err := c.DeleteDevice(orgName, args[0]); err != nil {
		return err
	}

	fmt.Println("Device deleted successfully.")
	return nil
}

// --- init ---

func init() {
	devicesSearchCmd.Flags().StringArrayVarP(&devSearchWhere, "where", "w", nil, `filter condition: "field op value" (repeatable)`)
	devicesSearchCmd.Flags().StringArrayVarP(&devSearchSelect, "select", "s", nil, "fields to return (repeatable; append @at for the timestamp, e.g. -s wt@at)")
	devicesSearchCmd.Flags().BoolVar(&devSearchAt, "at", false, "include the at timestamp for every selected field")
	devicesSearchCmd.Flags().StringSliceVar(&devSearchView, "view", nil, "named views to project (comma-separated or repeatable, e.g. --view summary,power); see 'og views list'")
	devicesSearchCmd.Flags().IntVar(&devSearchLimit, "limit", 0, "max number of results")
	devicesSearchCmd.Flags().StringVar(&devSearchFilter, "filter", "", "raw search filter as JSON (overrides -w)")

	devicesCreateCmd.Flags().StringVarP(&devCreateFile, "file", "f", "", "path to JSON file with device definition")
	devicesCreateCmd.MarkFlagRequired("file")

	devicesUpdateCmd.Flags().StringVarP(&devUpdateFile, "file", "f", "", "path to JSON file with device definition")
	devicesUpdateCmd.MarkFlagRequired("file")

	devicesCmd.AddCommand(devicesSearchCmd)
	devicesCmd.AddCommand(devicesGetCmd)
	devicesCmd.AddCommand(devicesCreateCmd)
	devicesCmd.AddCommand(devicesUpdateCmd)
	devicesCmd.AddCommand(devicesDeleteCmd)

	rootCmd.AddCommand(devicesCmd)
}
