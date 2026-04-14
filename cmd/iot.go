package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/carlosprados/og-cli/internal/client"
	"github.com/spf13/cobra"
)

var iotCmd = &cobra.Command{
	Use:   "iot",
	Short: "Device integration (South API)",
}

// --- collect ---

var iotCollectCmd = &cobra.Command{
	Use:   "collect <device-id> <datastream-id> <value>",
	Short: "Send a single data point to a device",
	Long: `Send a single value to a datastream on a device via the South API (X-ApiKey auth).

The API key is obtained automatically from the login response and stored in the profile.

Examples:
  og iot collect sense-001 wt 25.3
  og iot collect sense-001 wp 1013
  og iot collect sense-001 mystream "hello world"`,
	Args: cobra.ExactArgs(3),
	RunE: runIoTCollect,
}

// --- collect-file ---

var iotCollectFileCmd = &cobra.Command{
	Use:   "collect-file <device-id> -f <file.json>",
	Short: "Send IoT data from a JSON file",
	Long: `Send a full IoT payload to a device from a JSON file.

The JSON must follow the OpenGate collection format:
  {"version":"1.0.0","datastreams":[{"id":"temp","datapoints":[{"value":25}]}]}`,
	Args: cobra.ExactArgs(1),
	RunE: runIoTCollectFile,
}

var collectFile string

func runIoTCollect(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	if p.APIKey == "" {
		return fmt.Errorf("no API key found. Run 'og login' first to obtain one")
	}

	deviceID := args[0]
	datastreamID := args[1]
	rawValue := args[2]

	// Try to parse as number, bool, or keep as string
	value := parseValue(rawValue)

	if err := client.CollectSimple(p.Host, p.APIKey, deviceID, datastreamID, value); err != nil {
		return err
	}

	fmt.Printf("Sent %v to %s/%s\n", value, deviceID, datastreamID)
	return nil
}

func runIoTCollectFile(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	if p.APIKey == "" {
		return fmt.Errorf("no API key found. Run 'og login' first to obtain one")
	}

	deviceID := args[0]

	data, err := os.ReadFile(collectFile)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	var payload client.IoTPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("parsing IoT payload: %w", err)
	}

	if err := client.CollectIoT(p.Host, p.APIKey, deviceID, payload); err != nil {
		return err
	}

	fmt.Printf("Sent IoT data to %s (%d datastreams)\n", deviceID, len(payload.Datastreams))
	return nil
}

func parseValue(s string) any {
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		return v
	}
	if v, err := strconv.ParseBool(s); err == nil {
		return v
	}
	return s
}

func init() {
	iotCollectFileCmd.Flags().StringVarP(&collectFile, "file", "f", "", "path to JSON file with IoT payload")
	iotCollectFileCmd.MarkFlagRequired("file")

	iotCmd.AddCommand(iotCollectCmd)
	iotCmd.AddCommand(iotCollectFileCmd)
	rootCmd.AddCommand(iotCmd)
}
