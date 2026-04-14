package cmd

import (
	"fmt"
	"os"

	"github.com/carlosprados/og-cli/internal/client"
	"github.com/carlosprados/og-cli/internal/output"
	"github.com/spf13/cobra"
)

var alarmsCmd = &cobra.Command{
	Use:     "alarms",
	Aliases: []string{"al"},
	Short:   "Manage OpenGate alarms",
}

// --- search ---

var alarmsSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search alarms",
	Long: `Search alarms with simple conditions or a raw JSON filter.

Examples:
  og alarms search
  og alarms search -w "alarm.severity eq CRITICAL"
  og alarms search -w "alarm.status eq OPEN" -w "alarm.severity eq URGENT"
  og alarms search -w "alarm.entityIdentifier like sense" --limit 10`,
	RunE: runAlarmsSearch,
}

var (
	alSearchFilter string
	alSearchWhere  []string
	alSearchLimit  int
)

func runAlarmsSearch(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	c := client.New(p.Host, p.Token)

	filter, err := buildSearchFilter(alSearchWhere, alSearchLimit, nil, alSearchFilter)
	if err != nil {
		return err
	}

	resp, err := c.SearchAlarms(filter)
	if err != nil {
		return err
	}

	return output.Print(outFmt, resp.Alarms,
		[]string{"Severity", "Status", "Name", "Entity", "Rule", "Opening Date"},
		func(data any) [][]string {
			alarms := data.([]client.Alarm)
			rows := make([][]string, len(alarms))
			for i, a := range alarms {
				rows[i] = []string{a.Severity, a.Status, a.Name, a.EntityIdentifier, a.Rule, a.OpeningDate}
			}
			return rows
		},
	)
}

// --- summary ---

var alarmsSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Show alarm summary (counts by severity, status, rule, name)",
	Long: `Get aggregated alarm counts grouped by severity, status, rule, and name.

Examples:
  og alarms summary
  og alarms summary -w "alarm.status eq OPEN"`,
	RunE: runAlarmsSummary,
}

var (
	alSummaryFilter string
	alSummaryWhere  []string
)

func runAlarmsSummary(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	c := client.New(p.Host, p.Token)

	filter, err := buildSearchFilter(alSummaryWhere, 0, nil, alSummaryFilter)
	if err != nil {
		return err
	}

	resp, err := c.SummaryAlarms(filter)
	if err != nil {
		return err
	}

	if outFmt == output.FormatJSON {
		return output.PrintJSON(os.Stdout, resp)
	}

	fmt.Printf("Total alarms: %d\n\n", resp.Summary.Count)
	for _, group := range resp.Summary.SummaryGroup {
		for groupName, g := range group {
			fmt.Printf("%s (%d):\n", groupName, g.Count)
			for _, entry := range g.List {
				fmt.Printf("  %-20s %d\n", entry.Name, entry.Count)
			}
			fmt.Println()
		}
	}
	return nil
}

// --- attend ---

var alarmsAttendCmd = &cobra.Command{
	Use:   "attend <alarm-id> [alarm-id...]",
	Short: "Mark alarms as attended",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runAlarmsAttend,
}

var attendNotes string

func runAlarmsAttend(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	c := client.New(p.Host, p.Token)

	resp, err := c.AttendAlarms(args, attendNotes)
	if err != nil {
		return err
	}

	fmt.Printf("Attended: %d, Errors: %d\n", resp.Result.Successful, resp.Result.Error.Count)
	return nil
}

// --- close ---

var alarmsCloseCmd = &cobra.Command{
	Use:   "close <alarm-id> [alarm-id...]",
	Short: "Close alarms",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runAlarmsClose,
}

var closeNotes string

func runAlarmsClose(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	c := client.New(p.Host, p.Token)

	resp, err := c.CloseAlarms(args, closeNotes)
	if err != nil {
		return err
	}

	fmt.Printf("Closed: %d, Errors: %d\n", resp.Result.Successful, resp.Result.Error.Count)
	return nil
}

// --- init ---

func init() {
	alarmsSearchCmd.Flags().StringArrayVarP(&alSearchWhere, "where", "w", nil, `filter condition: "field op value" (repeatable)`)
	alarmsSearchCmd.Flags().IntVar(&alSearchLimit, "limit", 0, "max number of results")
	alarmsSearchCmd.Flags().StringVar(&alSearchFilter, "filter", "", "raw search filter as JSON (overrides -w)")

	alarmsSummaryCmd.Flags().StringArrayVarP(&alSummaryWhere, "where", "w", nil, `filter condition: "field op value" (repeatable)`)
	alarmsSummaryCmd.Flags().StringVar(&alSummaryFilter, "filter", "", "raw search filter as JSON (overrides -w)")

	alarmsAttendCmd.Flags().StringVar(&attendNotes, "notes", "", "notes for the attend action")
	alarmsCloseCmd.Flags().StringVar(&closeNotes, "notes", "", "notes for the close action")

	alarmsCmd.AddCommand(alarmsSearchCmd)
	alarmsCmd.AddCommand(alarmsSummaryCmd)
	alarmsCmd.AddCommand(alarmsAttendCmd)
	alarmsCmd.AddCommand(alarmsCloseCmd)

	rootCmd.AddCommand(alarmsCmd)
}

