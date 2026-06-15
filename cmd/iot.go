package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/carlosprados/og-cli/pkg/opengate"
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

var (
	collectFile    string
	rawRoute       string
	rawBody        string
	rawFile        string
	rawContentType string
)

// --- collect-raw (HTTP custom south route — connector function trigger) ---

var iotCollectRawCmd = &cobra.Command{
	Use:   "collect-raw <device-id> --route <path>",
	Short: "POST a raw body to a connector function's HTTP south route",
	Long: `Trigger a COLLECTION/RESPONSE connector function over its HTTP south route.

Unlike 'collect' (which posts a structured payload to collect/iot and bypasses
connector functions), this POSTs a raw body to /south/v80/devices/<id>/<route>,
where <route> is the connector function's HTTP southCriteria path. The CF then
transforms the body and emits datapoints (verify the result with 'og devices search').

Body comes from --body or -f <file>. X-ApiKey auth (from the profile).

Examples:
  og iot collect-raw charlie-01 --route ogcli-demo --body '{"raw":21,"id":"abc"}'
  og iot collect-raw charlie-01 --route sensors/temp -f reading.json
  og iot collect-raw charlie-01 --route raw/feed --body 'PLAIN TEXT' --content-type text/plain`,
	Args: cobra.ExactArgs(1),
	RunE: runIoTCollectRaw,
}

func runIoTCollectRaw(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	if p.APIKey == "" {
		return fmt.Errorf("no API key found. Run 'og login' first to obtain one")
	}
	deviceID := args[0]

	var body []byte
	switch {
	case rawFile != "":
		data, err := os.ReadFile(rawFile)
		if err != nil {
			return fmt.Errorf("reading file: %w", err)
		}
		body = data
	case rawBody != "":
		body = []byte(rawBody)
	default:
		return fmt.Errorf("provide a body with --body or -f <file>")
	}

	data, status, err := opengate.CollectRaw(p.Host, p.APIKey, deviceID, rawRoute, body, rawContentType)
	if err != nil {
		return err
	}
	fmt.Printf("Posted to %s/%s (HTTP %d)\n", deviceID, strings.TrimPrefix(rawRoute, "/"), status)
	if len(data) > 0 {
		fmt.Println(string(data))
	}
	return nil
}

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

	if err := opengate.CollectSimple(p.Host, p.APIKey, deviceID, datastreamID, value); err != nil {
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

	var payload opengate.IoTPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("parsing IoT payload: %w", err)
	}

	if err := opengate.CollectIoT(p.Host, p.APIKey, deviceID, payload); err != nil {
		return err
	}

	fmt.Printf("Sent IoT data to %s (%d datastreams)\n", deviceID, len(payload.Datastreams))
	return nil
}

// --- MQTT south client ---

var (
	mqttHost        string
	mqttPort        int
	mqttTLS         bool
	mqttInsecure    bool
	mqttQoS         int
	mqttTopic       string
	mqttFile        string
	mqttRaw         string
	mqttRepeat      int
	mqttInterval    time.Duration
	mqttCount       int
	mqttReqTopic    string
	mqttRespTopic   string
	mqttResult      string
	mqttDescription string
	mqttRefreshData string
)

var iotPublishCmd = &cobra.Command{
	Use:   "publish <device-id> [<datastream-id> <value>]",
	Short: "Publish data to OpenGate over MQTT (South plane)",
	Long: `Publish a message to the OpenGate MQTT broker as a device.

Auth uses the device id as username and the profile API key as password. The
default topic is odm/iot/<device-id>; override --topic to publish to a connector
function's custom south route (topics are NOT fixed when connector functions
define their own southCriterias).

Body sources (pick one):
  positional <datastream-id> <value>  build a collection payload for one datastream
  -f <file.json>                      send a full IoT payload (version/datastreams)
  --raw '<string>'                    publish the literal body verbatim (custom CF routes)

Examples:
  og iot publish sense-001 temperature 21.5
  og iot publish sense-001 -f payload.json
  og iot publish sense-001 --topic my/cf/route --raw '{"raw":21,"id":"abc"}'`,
	Args: cobra.MinimumNArgs(1),
	RunE: runIoTPublish,
}

var iotSubscribeCmd = &cobra.Command{
	Use:   "subscribe <device-id>",
	Short: "Subscribe to an MQTT topic and print messages live (Ctrl-C to stop)",
	Long: `Connect to the OpenGate MQTT broker and print every message on a topic.

Default topic is odm/request/<device-id> (incoming operations); override --topic
to observe any topic, including a connector function's custom south route. Useful
for debugging connector functions and operation flows.

Examples:
  og iot subscribe sense-001                       # watch incoming operations
  og iot subscribe sense-001 --topic odm/iot/sense-001
  og iot subscribe sense-001 --topic my/cf/route --count 5`,
	Args: cobra.ExactArgs(1),
	RunE: runIoTSubscribe,
}

var iotDeviceCmd = &cobra.Command{
	Use:   "device <device-id>",
	Short: "Act as a virtual MQTT device: receive operations and auto-answer them",
	Long: `Run og as a virtual OpenGate device over MQTT.

Subscribes to odm/request/<device-id>, and for every operation request received
it publishes an acknowledging response to odm/response/<device-id> (echoing the
operation name and id). Runs until Ctrl-C.

With --refresh-data, a REFRESH_INFO operation additionally publishes that payload
file to odm/iot/<device-id> to fulfil the info request.

Examples:
  og iot device sense-001
  og iot device sense-001 --result ERROR --description "simulated failure"
  og iot device sense-001 --refresh-data refresh.json`,
	Args: cobra.ExactArgs(1),
	RunE: runIoTDevice,
}

func mqttClientFor(deviceID string) (*opengate.MQTTClient, error) {
	p, err := activeProfile()
	if err != nil {
		return nil, err
	}
	if p.APIKey == "" {
		return nil, fmt.Errorf("no API key found. Run 'og login' first to obtain one")
	}
	host := mqttHost
	if host == "" {
		host = opengate.MQTTHostFromProfile(p.Host)
	}
	return opengate.NewMQTTClient(host, mqttPort, mqttTLS, mqttInsecure, deviceID, p.APIKey)
}

func runIoTPublish(cmd *cobra.Command, args []string) error {
	deviceID := args[0]

	var body []byte
	switch {
	case mqttRaw != "":
		body = []byte(mqttRaw)
	case mqttFile != "":
		data, err := os.ReadFile(mqttFile)
		if err != nil {
			return fmt.Errorf("reading file: %w", err)
		}
		body = data
	case len(args) >= 3:
		payload := opengate.IoTPayload{
			Version: "1.0.0",
			Device:  deviceID,
			Datastreams: []opengate.IoTDatastream{
				{ID: args[1], Datapoints: []opengate.IoTDatapoint{{Value: parseValue(args[2])}}},
			},
		}
		b, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = b
	default:
		return fmt.Errorf("provide a body: <datastream-id> <value>, -f <file>, or --raw <string>")
	}

	topic := mqttTopic
	if topic == "" {
		topic = opengate.MQTTDataTopic(deviceID)
	}

	client, err := mqttClientFor(deviceID)
	if err != nil {
		return err
	}
	defer client.Disconnect()

	n := max(mqttRepeat, 1)
	for i := range n {
		if err := client.Publish(topic, body, byte(mqttQoS)); err != nil {
			return err
		}
		fmt.Printf("Published to %s (%d bytes)\n", topic, len(body))
		if i < n-1 && mqttInterval > 0 {
			time.Sleep(mqttInterval)
		}
	}
	return nil
}

func runIoTSubscribe(cmd *cobra.Command, args []string) error {
	deviceID := args[0]
	topic := mqttTopic
	if topic == "" {
		topic = opengate.MQTTRequestTopic(deviceID)
	}

	client, err := mqttClientFor(deviceID)
	if err != nil {
		return err
	}
	defer client.Disconnect()

	done := make(chan struct{})
	received := 0
	if err := client.Subscribe(topic, byte(mqttQoS), func(t string, payload []byte) {
		received++
		fmt.Printf("[%s] %s\n%s\n", time.Now().Format("15:04:05"), t, prettyOrRaw(payload))
		if mqttCount > 0 && received >= mqttCount {
			select {
			case <-done:
			default:
				close(done)
			}
		}
	}); err != nil {
		return err
	}

	fmt.Printf("Subscribed to %s — waiting for messages (Ctrl-C to stop)\n", topic)
	waitForSignalOr(done)
	return nil
}

func runIoTDevice(cmd *cobra.Command, args []string) error {
	deviceID := args[0]
	reqTopic := mqttReqTopic
	if reqTopic == "" {
		reqTopic = opengate.MQTTRequestTopic(deviceID)
	}
	respTopic := mqttRespTopic
	if respTopic == "" {
		respTopic = opengate.MQTTResponseTopic(deviceID)
	}
	result := mqttResult
	if result == "" {
		result = "SUCCESSFUL"
	}

	var refreshBody []byte
	if mqttRefreshData != "" {
		data, err := os.ReadFile(mqttRefreshData)
		if err != nil {
			return fmt.Errorf("reading --refresh-data file: %w", err)
		}
		refreshBody = data
	}

	client, err := mqttClientFor(deviceID)
	if err != nil {
		return err
	}
	defer client.Disconnect()

	if err := client.Subscribe(reqTopic, byte(mqttQoS), func(_ string, payload []byte) {
		req, err := opengate.ParseOperationRequest(payload)
		if err != nil {
			fmt.Printf("  ! ignoring non-operation message: %v\n", err)
			return
		}
		op := req.Operation.Request
		fmt.Printf("[%s] operation %s id=%s params=%v\n", time.Now().Format("15:04:05"), op.Name, op.ID, op.Parameters)

		resp := opengate.BuildOperationResponse(deviceID, op.Name, op.ID, result, mqttDescription)
		if err := client.Publish(respTopic, resp, byte(mqttQoS)); err != nil {
			fmt.Printf("  ! failed to publish response: %v\n", err)
			return
		}
		fmt.Printf("  → responded %s on %s\n", result, respTopic)

		if op.Name == "REFRESH_INFO" && refreshBody != nil {
			dataTopic := opengate.MQTTDataTopic(deviceID)
			if err := client.Publish(dataTopic, refreshBody, byte(mqttQoS)); err != nil {
				fmt.Printf("  ! failed to publish refresh data: %v\n", err)
			} else {
				fmt.Printf("  → published refresh data on %s\n", dataTopic)
			}
		}
	}); err != nil {
		return err
	}

	fmt.Printf("Virtual device %s online — listening on %s, answering on %s (Ctrl-C to stop)\n", deviceID, reqTopic, respTopic)
	waitForSignalOr(nil)
	return nil
}

// waitForSignalOr blocks until SIGINT/SIGTERM, or until done is closed (if non-nil).
func waitForSignalOr(done <-chan struct{}) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	if done == nil {
		<-sig
		return
	}
	select {
	case <-sig:
	case <-done:
	}
}

// prettyOrRaw indents JSON payloads, falling back to the raw string.
func prettyOrRaw(payload []byte) string {
	var v any
	if err := json.Unmarshal(payload, &v); err == nil {
		if b, err := json.MarshalIndent(v, "", "  "); err == nil {
			return string(b)
		}
	}
	return string(payload)
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

	// Shared MQTT connection flags.
	for _, c := range []*cobra.Command{iotPublishCmd, iotSubscribeCmd, iotDeviceCmd} {
		c.Flags().StringVar(&mqttHost, "mqtt-host", "", "MQTT broker host (default: derived from profile host)")
		c.Flags().IntVar(&mqttPort, "port", opengate.DefaultMQTTPort, "MQTT broker port")
		c.Flags().BoolVar(&mqttTLS, "tls", true, "use TLS (ssl://, port 8883) — the OpenGate broker requires it")
		c.Flags().BoolVar(&mqttInsecure, "insecure", true, "skip TLS certificate verification (OpenGate broker uses a private CA)")
		c.Flags().IntVar(&mqttQoS, "qos", 0, "MQTT QoS (0, 1 or 2)")
	}

	iotPublishCmd.Flags().StringVar(&mqttTopic, "topic", "", "topic to publish to (default: odm/iot/<device-id>)")
	iotPublishCmd.Flags().StringVarP(&mqttFile, "file", "f", "", "path to JSON file with the IoT payload")
	iotPublishCmd.Flags().StringVar(&mqttRaw, "raw", "", "publish this literal string body (for custom CF routes)")
	iotPublishCmd.Flags().IntVar(&mqttRepeat, "repeat", 1, "publish the message this many times")
	iotPublishCmd.Flags().DurationVar(&mqttInterval, "interval", 0, "delay between repeats (e.g. 2s)")

	iotSubscribeCmd.Flags().StringVar(&mqttTopic, "topic", "", "topic to subscribe to (default: odm/request/<device-id>)")
	iotSubscribeCmd.Flags().IntVar(&mqttCount, "count", 0, "stop after receiving N messages (0 = run until Ctrl-C)")

	iotDeviceCmd.Flags().StringVar(&mqttReqTopic, "request-topic", "", "incoming operations topic (default: odm/request/<device-id>)")
	iotDeviceCmd.Flags().StringVar(&mqttRespTopic, "response-topic", "", "operation response topic (default: odm/response/<device-id>)")
	iotDeviceCmd.Flags().StringVar(&mqttResult, "result", "SUCCESSFUL", "resultCode to answer with (SUCCESSFUL or ERROR)")
	iotDeviceCmd.Flags().StringVar(&mqttDescription, "description", "", "step description in the response")
	iotDeviceCmd.Flags().StringVar(&mqttRefreshData, "refresh-data", "", "JSON payload published to odm/iot on REFRESH_INFO")

	iotCollectRawCmd.Flags().StringVar(&rawRoute, "route", "", "connector function's HTTP south path (e.g. ogcli-demo)")
	iotCollectRawCmd.Flags().StringVar(&rawBody, "body", "", "raw body to POST")
	iotCollectRawCmd.Flags().StringVarP(&rawFile, "file", "f", "", "read the raw body from this file")
	iotCollectRawCmd.Flags().StringVar(&rawContentType, "content-type", "application/json", "Content-Type header")
	iotCollectRawCmd.MarkFlagRequired("route")

	iotCmd.AddCommand(iotCollectCmd)
	iotCmd.AddCommand(iotCollectFileCmd)
	iotCmd.AddCommand(iotCollectRawCmd)
	iotCmd.AddCommand(iotPublishCmd)
	iotCmd.AddCommand(iotSubscribeCmd)
	iotCmd.AddCommand(iotDeviceCmd)
	rootCmd.AddCommand(iotCmd)
}
