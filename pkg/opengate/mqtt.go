package opengate

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Default OpenGate MQTT broker settings (South plane).
const (
	DefaultMQTTPort    = 1883
	DefaultMQTTTLSPort = 8883
)

// MQTTClient is a thin wrapper over the paho MQTT client for the OpenGate South
// plane. Authentication uses username = device id, password = API key.
type MQTTClient struct {
	c        mqtt.Client
	deviceID string
}

// MQTTHostFromProfile strips the scheme (and any trailing slash) from a profile
// host so it can be used as an MQTT broker host. "https://api.opengate.es" → "api.opengate.es".
func MQTTHostFromProfile(host string) string {
	host = strings.TrimSuffix(host, "/")
	if i := strings.Index(host, "://"); i != -1 {
		host = host[i+3:]
	}
	return host
}

// NewMQTTClient connects to the OpenGate MQTT broker as the given device.
// useTLS switches to ssl:// (default TLS port when port is 0/default plaintext).
// The broker presents a public Let's Encrypt chain, so TLS verifies against the
// system root store by default. insecure skips verification entirely (escape
// hatch). caFile, when set, supplies an extra CA/chain PEM (e.g. for a site whose
// broker still omits the intermediate) appended to the system pool.
func NewMQTTClient(host string, port int, useTLS, insecure bool, caFile, deviceID, apiKey string) (*MQTTClient, error) {
	scheme := "tcp"
	if useTLS {
		scheme = "ssl"
		if port == 0 || port == DefaultMQTTPort {
			port = DefaultMQTTTLSPort
		}
	}
	if port == 0 {
		port = DefaultMQTTPort
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("%s://%s:%d", scheme, host, port))
	opts.SetClientID(fmt.Sprintf("og-%s-%d", deviceID, os.Getpid()))
	opts.SetUsername(deviceID)
	opts.SetPassword(apiKey)
	opts.SetConnectTimeout(15 * time.Second)
	opts.SetAutoReconnect(true)
	opts.SetCleanSession(true)
	if useTLS {
		cfg, err := mqttTLSConfig(host, insecure, caFile)
		if err != nil {
			return nil, err
		}
		opts.SetTLSConfig(cfg)
	}

	c := mqtt.NewClient(opts)
	if tok := c.Connect(); tok.Wait() && tok.Error() != nil {
		return nil, fmt.Errorf("connecting to MQTT broker %s://%s:%d: %w", scheme, host, port, tok.Error())
	}
	return &MQTTClient{c: c, deviceID: deviceID}, nil
}

// mqttTLSConfig builds the TLS config for the broker. By default it verifies the
// server certificate against the system root store (the broker now serves the full
// Let's Encrypt chain). insecure disables verification; caFile appends an extra
// CA/chain PEM to the system pool for sites that still need it.
func mqttTLSConfig(host string, insecure bool, caFile string) (*tls.Config, error) {
	cfg := &tls.Config{ServerName: host}
	if insecure {
		cfg.InsecureSkipVerify = true
		return cfg, nil
	}
	if caFile != "" {
		pool, err := x509.SystemCertPool()
		if err != nil || pool == nil {
			pool = x509.NewCertPool()
		}
		pem, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("reading --ca-file %s: %w", caFile, err)
		}
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("no valid certificates found in --ca-file %s", caFile)
		}
		cfg.RootCAs = pool
	}
	return cfg, nil
}

// Publish sends a message to a topic and waits for it to leave the client.
func (m *MQTTClient) Publish(topic string, payload []byte, qos byte) error {
	tok := m.c.Publish(topic, qos, false, payload)
	tok.Wait()
	if err := tok.Error(); err != nil {
		return fmt.Errorf("publishing to %s: %w", topic, err)
	}
	return nil
}

// Subscribe registers a handler for a topic. The handler runs on the paho
// callback goroutine; the caller is responsible for blocking (e.g. on a signal).
func (m *MQTTClient) Subscribe(topic string, qos byte, handler func(topic string, payload []byte)) error {
	tok := m.c.Subscribe(topic, qos, func(_ mqtt.Client, msg mqtt.Message) {
		handler(msg.Topic(), msg.Payload())
	})
	tok.Wait()
	if err := tok.Error(); err != nil {
		return fmt.Errorf("subscribing to %s: %w", topic, err)
	}
	return nil
}

// Disconnect cleanly closes the connection.
func (m *MQTTClient) Disconnect() {
	m.c.Disconnect(250)
}

// --- default OpenGate South topics ---

// MQTTDataTopic is the default topic to publish collected data for a device.
func MQTTDataTopic(deviceID string) string { return "odm/iot/" + deviceID }

// MQTTRequestTopic is the default topic a device subscribes to for incoming operations.
func MQTTRequestTopic(deviceID string) string { return "odm/request/" + deviceID }

// MQTTResponseTopic is the default topic a device publishes operation responses to.
func MQTTResponseTopic(deviceID string) string { return "odm/response/" + deviceID }

// --- operations over MQTT ---

// OperationRequest is the message OpenGate publishes to odm/request/{deviceId}.
type OperationRequest struct {
	Operation struct {
		Request struct {
			Timestamp  int64          `json:"timestamp"`
			Name       string         `json:"name"`
			Parameters map[string]any `json:"parameters"`
			ID         string         `json:"id"`
		} `json:"request"`
	} `json:"operation"`
}

// ParseOperationRequest decodes an odm/request payload.
func ParseOperationRequest(payload []byte) (*OperationRequest, error) {
	var r OperationRequest
	if err := json.Unmarshal(payload, &r); err != nil {
		return nil, fmt.Errorf("parsing operation request: %w", err)
	}
	return &r, nil
}

// BuildOperationResponse builds the odm/response payload acknowledging an
// operation. resultCode is typically SUCCESSFUL or ERROR; description is a
// human-readable step description (may be empty).
func BuildOperationResponse(deviceID, name, id, resultCode, description string) []byte {
	now := time.Now().UnixMilli()
	resp := map[string]any{
		"operation": map[string]any{
			"response": map[string]any{
				"name":              name,
				"timestamp":         now,
				"resultDescription": resultDescriptionFor(resultCode),
				"steps": []map[string]any{
					{
						"timestamp":   now,
						"description": description,
						"name":        name,
						"result":      resultCode,
					},
				},
				"deviceId":   deviceID,
				"resultCode": resultCode,
				"id":         id,
			},
		},
		"version": "1.0",
	}
	out, _ := json.Marshal(resp)
	return out
}

func resultDescriptionFor(resultCode string) string {
	if resultCode == "SUCCESSFUL" {
		return "Success"
	}
	return resultCode
}
