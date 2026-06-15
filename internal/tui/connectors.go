package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/carlosprados/og-cli/pkg/opengate"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- connector functions ---

type connectorsModel struct {
	table   table.Model
	items   []json.RawMessage
	loaded  bool
	loading bool
}

type connectorsFetchedMsg struct {
	items []json.RawMessage
	err   error
}

type connectorStatusMsg struct {
	id     string
	status string
	err    error
}

// nextConnectorStatus cycles DISABLED → TEST → PRODUCTION → DISABLED.
func nextConnectorStatus(current string) string {
	switch strings.ToUpper(current) {
	case "DISABLED":
		return "TEST"
	case "TEST":
		return "PRODUCTION"
	default:
		return "DISABLED"
	}
}

func (m model) fetchConnectors() tea.Cmd {
	org := ""
	if m.profile != nil {
		org = m.profile.Organization
	}
	return func() tea.Msg {
		if org == "" {
			return connectorsFetchedMsg{err: fmt.Errorf("organization required (set it in the profile)")}
		}
		resp, err := m.client.ListConnectorFunctions(org, tuiDefaultChannel)
		if err != nil {
			return connectorsFetchedMsg{err: err}
		}
		return connectorsFetchedMsg{items: resp.ConnectorFunctions}
	}
}

func (m model) cycleConnectorStatus(raw json.RawMessage) tea.Cmd {
	s := opengate.ParseConnectorFunctionSummary(raw)
	org := ""
	if m.profile != nil {
		org = m.profile.Organization
	}
	return func() tea.Msg {
		if org == "" {
			return connectorStatusMsg{err: fmt.Errorf("organization required (set it in the profile)")}
		}
		target := nextConnectorStatus(s.OperationalStatus)
		err := m.client.SetConnectorFunctionStatus(org, tuiDefaultChannel, s.Identifier, target)
		return connectorStatusMsg{id: s.Identifier, status: target, err: err}
	}
}

func (m model) updateConnectors(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case connectorsFetchedMsg:
		m.connectors.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.connectors.items = msg.items
		m.connectors.loaded = true
		m.connectors.table = buildConnectorsTable(msg.items, m.width)
		return m, nil

	case connectorStatusMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.message = fmt.Sprintf("Connector function %s set to %s", msg.id, msg.status)
		m.connectors.loading = true
		return m, m.fetchConnectors()

	case tea.KeyMsg:
		switch msg.String() {
		case "s":
			if len(m.connectors.items) > 0 {
				sel := m.connectors.table.Cursor()
				if sel < len(m.connectors.items) {
					return m, m.cycleConnectorStatus(m.connectors.items[sel])
				}
			}
		case "r":
			m.connectors.loading = true
			return m, m.fetchConnectors()
		}
	}

	var cmd tea.Cmd
	m.connectors.table, cmd = m.connectors.table.Update(msg)
	return m, cmd
}

func buildConnectorsTable(items []json.RawMessage, width int) table.Model {
	columns := []table.Column{
		{Title: "Name", Width: 28},
		{Title: "Type", Width: 12},
		{Title: "Status", Width: 12},
		{Title: "Operation", Width: 16},
		{Title: "Identifier", Width: 38},
	}
	if width > 0 {
		a := width - 10
		columns[0].Width = a * 26 / 100
		columns[1].Width = a * 12 / 100
		columns[2].Width = a * 12 / 100
		columns[3].Width = a * 16 / 100
		columns[4].Width = a * 34 / 100
	}

	rows := make([]table.Row, len(items))
	for i, raw := range items {
		s := opengate.ParseConnectorFunctionSummary(raw)
		rows[i] = table.Row{s.DisplayName(), s.Type, s.OperationalStatus, s.OperationName, s.Identifier}
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(min(len(rows)+1, 20)),
	)
	st := table.DefaultStyles()
	st.Header = st.Header.BorderStyle(lipgloss.NormalBorder()).BorderBottom(true).Bold(true).Foreground(accent)
	st.Selected = st.Selected.Foreground(highlight).Bold(true)
	t.SetStyles(st)
	return t
}

func (m model) viewConnectorsScreen() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("  Connector Functions"))
	b.WriteString("\n")

	if m.connectors.loading {
		b.WriteString(dimStyle.Render("  Loading..."))
		return b.String()
	}
	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %v", m.err)))
		b.WriteString(helpStyle.Render("\n  r retry • esc back"))
		return b.String()
	}
	if m.message != "" {
		b.WriteString(successStyle.Render(fmt.Sprintf("  %s", m.message)))
		b.WriteString("\n")
	}
	if m.connectors.loaded {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %d connector functions", len(m.connectors.items))))
		b.WriteString("\n\n")
		b.WriteString(m.connectors.table.View())
	}

	b.WriteString(helpStyle.Render("\n  ↑↓/jk navigate • s cycle status • r refresh • esc back"))
	return b.String()
}
