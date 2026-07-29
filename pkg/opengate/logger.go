package opengate

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync/atomic"

	"github.com/gorilla/websocket"
)

// Functions-logger kinds.
const (
	LoggerConnectorFunctions = "connectorFunctions"
	LoggerRules              = "rules"
)

const functionsLoggerPath = "/north/functions-logger/%s/organizations/%s/channels/%s/%s"

// LogMessage is a single trace emitted by the functions-logger WebSocket.
type LogMessage struct {
	Message   string `json:"message"`
	Level     string `json:"level"`
	Timestamp int64  `json:"timestamp"` // UTC milliseconds
}

// StreamFunctionLogs connects to the OpenGate functions-logger WebSocket and
// invokes onMessage for each trace until the connection closes or stop is
// signalled (close the stop channel to disconnect).
//
// kind is LoggerConnectorFunctions or LoggerRules. Authentication uses the
// X-ApiKey URL parameter (the device/server API key, not the JWT). level is one
// of ERROR, WARN, INFO, DEBUG, TRACE; empty defaults to the server's INFO.
func (c *Client) StreamFunctionLogs(ctx context.Context, apiKey, kind, org, channel, id, level string, onMessage func(LogMessage), stop <-chan struct{}) error {
	scheme := "wss"
	host := c.BaseURL
	switch {
	case strings.HasPrefix(host, "https://"):
		host = strings.TrimPrefix(host, "https://")
	case strings.HasPrefix(host, "http://"):
		scheme, host = "ws", strings.TrimPrefix(host, "http://")
	}

	q := url.Values{}
	q.Set("X-ApiKey", apiKey)
	if level != "" {
		q.Set("level", level)
	}
	endpoint := fmt.Sprintf("%s://%s"+functionsLoggerPath+"?%s",
		scheme, host, kind, org, channel, id, q.Encode())

	conn, resp, err := websocket.DefaultDialer.DialContext(ctx, endpoint, nil)
	if err != nil {
		if resp != nil {
			return fmt.Errorf("functions-logger dial failed (HTTP %d): %w", resp.StatusCode, err)
		}
		return fmt.Errorf("functions-logger dial failed: %w", err)
	}
	defer conn.Close()

	var stopped atomic.Bool
	go func() {
		<-stop
		stopped.Store(true)
		conn.Close()
	}()

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			if stopped.Load() {
				return nil
			}
			return fmt.Errorf("functions-logger read: %w", err)
		}
		var m LogMessage
		if json.Unmarshal(data, &m) != nil {
			m = LogMessage{Message: string(data), Level: "INFO"}
		}
		onMessage(m)
	}
}

// CollectFunctionLogs streams logs and accumulates them until stop is signalled
// or max messages are collected (max <= 0 means unbounded until stop). Returns
// the messages gathered. Convenience for non-interactive callers (MCP tools).
func (c *Client) CollectFunctionLogs(ctx context.Context, apiKey, kind, org, channel, id, level string, max int, stop <-chan struct{}) ([]LogMessage, error) {
	var msgs []LogMessage
	done := make(chan struct{})
	var once int32
	closeDone := func() {
		if atomic.CompareAndSwapInt32(&once, 0, 1) {
			close(done)
		}
	}

	// Bridge: stop OR max reached closes the stream.
	relay := make(chan struct{})
	go func() {
		select {
		case <-stop:
		case <-done:
		}
		close(relay)
	}()

	err := c.StreamFunctionLogs(ctx, apiKey, kind, org, channel, id, level, func(m LogMessage) {
		msgs = append(msgs, m)
		if max > 0 && len(msgs) >= max {
			closeDone()
		}
	}, relay)
	return msgs, err
}
