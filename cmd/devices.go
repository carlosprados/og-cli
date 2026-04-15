package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/carlosprados/og-cli/internal/client"
	"github.com/carlosprados/og-cli/internal/output"
	"github.com/carlosprados/og-cli/internal/query"
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

Examples:
  og dev search -w "provision.device.administrativeState eq ACTIVE"
  og dev search -w "provision.device.identifier like sense" --limit 10
  og dev search -w "provision.device.administrativeState eq ACTIVE" -w "provision.device.identifier like sense"
  og dev search -w "wt gt 20"
  og dev search -w "device.temperature.value gte 50" -w "provision.device.operationalStatus eq NORMAL"
  og dev search --filter '{"filter":{"or":[...]}}'`,
	RunE: runDevicesSearch,
}

var (
	devSearchFilter string
	devSearchWhere  []string
	devSearchLimit  int
	devSearchSelect []string
)

func runDevicesSearch(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	c := client.New(p.Host, p.Token)

	filter, err := buildSearchFilter(devSearchWhere, devSearchLimit, devSearchSelect, devSearchFilter)
	if err != nil {
		return err
	}

	resp, err := c.SearchDevices(filter)
	if err != nil {
		return err
	}

	if len(devSearchSelect) > 0 {
		return printSelectedDevices(resp.Devices, devSearchSelect)
	}

	return output.Print(outFmt, resp.Devices,
		[]string{"Identifier", "Name", "Organization", "State"},
		func(data any) [][]string {
			devices := data.([]json.RawMessage)
			rows := make([][]string, len(devices))
			for i, raw := range devices {
				s := client.ParseDeviceSummary(raw)
				rows[i] = []string{s.Identifier, s.Name, s.Org, s.Status}
			}
			return rows
		},
	)
}

// buildSearchFilter is shared by all search commands.
func buildSearchFilter(where []string, limit int, selectFields []string, rawFilter string) (json.RawMessage, error) {
	var conditions []query.Condition
	for _, w := range where {
		c, err := query.ParseCondition(w)
		if err != nil {
			return nil, err
		}
		conditions = append(conditions, c)
	}
	return query.MergeWithRaw(query.SearchParams{
		Conditions: conditions,
		Limit:      limit,
		Select:     selectFields,
	}, rawFilter)
}

// printSelectedDevices renders devices with dynamic columns from -s fields.
func printSelectedDevices(devices []json.RawMessage, fields []string) error {
	headers := make([]string, len(fields))
	for i, f := range fields {
		headers[i] = query.FieldAlias(f)
	}

	return output.Print(outFmt, devices, headers,
		func(data any) [][]string {
			devs := data.([]json.RawMessage)
			rows := make([][]string, len(devs))
			for i, raw := range devs {
				row := make([]string, len(fields))
				for j, field := range fields {
					row[j] = client.ExtractFlatValue(raw, field)
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
	c := client.New(p.Host, p.Token)

	data, err := c.GetDevice(orgName, args[0])
	if err != nil {
		return err
	}

	// Device JSON is complex (flattened format), always output as JSON for get
	if outFmt == output.FormatTable {
		s := client.ParseDeviceSummary(data)
		return output.Print(outFmt, s,
			[]string{"Identifier", "Name", "Organization", "State"},
			func(d any) [][]string {
				ds := d.(client.DeviceSummary)
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

	c := client.New(p.Host, p.Token)
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

	c := client.New(p.Host, p.Token)
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

	c := client.New(p.Host, p.Token)
	if err := c.DeleteDevice(orgName, args[0]); err != nil {
		return err
	}

	fmt.Println("Device deleted successfully.")
	return nil
}

// --- init ---

func init() {
	devicesSearchCmd.Flags().StringArrayVarP(&devSearchWhere, "where", "w", nil, `filter condition: "field op value" (repeatable)`)
	devicesSearchCmd.Flags().StringArrayVarP(&devSearchSelect, "select", "s", nil, "fields to return (repeatable, e.g. -s provision.device.identifier -s wt)")
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
