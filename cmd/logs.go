package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/carlosprados/og-cli/pkg/opengate"
	"github.com/charmbracelet/lipgloss"
)

// logsLevel is the shared --level flag for the connectors/rules logs commands.
var logsLevel string

// level → colour. Severity ordering follows OpenGate's logger (ERROR highest).
var logLevelStyles = map[string]lipgloss.Style{
	"ERROR": lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true),  // bright red
	"WARN":  lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true), // bright yellow
	"INFO":  lipgloss.NewStyle().Foreground(lipgloss.Color("10")),            // bright green
	"DEBUG": lipgloss.NewStyle().Foreground(lipgloss.Color("12")),            // bright blue
	"TRACE": lipgloss.NewStyle().Foreground(lipgloss.Color("8")),             // grey
}

var logTimestampStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

func printLogLine(m opengate.LogMessage) {
	level := strings.ToUpper(m.Level)
	style, ok := logLevelStyles[level]
	if !ok {
		style = lipgloss.NewStyle()
	}
	ts := ""
	if m.Timestamp > 0 {
		ts = time.UnixMilli(m.Timestamp).Format("15:04:05.000")
	}
	fmt.Printf("%s %s %s\n",
		logTimestampStyle.Render(ts),
		style.Render(fmt.Sprintf("%-5s", level)),
		m.Message,
	)
}

// streamFunctionLogs runs the live logger for a connector function or rule,
// printing colourised traces until the user interrupts with Ctrl-C.
func streamFunctionLogs(ctx context.Context, kind, channel, id string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	if p.APIKey == "" {
		return fmt.Errorf("no API key in profile — run 'og login' first to obtain one")
	}
	orgName, err := resolveOrg(p)
	if err != nil {
		return err
	}
	c := opengate.New(p.Host, p.Token)

	stop := make(chan struct{})
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sig
		close(stop)
	}()

	level := strings.ToUpper(logsLevel)
	fmt.Fprintf(os.Stderr, "Streaming %s logs for %s (channel %s, level %s) — Ctrl-C to stop\n",
		kind, id, channel, level)
	return c.StreamFunctionLogs(ctx, p.APIKey, kind, orgName, channel, id, level, printLogLine, stop)
}
