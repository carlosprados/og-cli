package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/carlosprados/og-cli/internal/client"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type devicesModel struct {
	table   table.Model
	items   []json.RawMessage
	loaded  bool
	loading bool
}

type deviceDetailModel struct {
	data    json.RawMessage
	summary client.DeviceSummary
	content string
}

type devicesFetchedMsg struct {
	items []json.RawMessage
	err   error
}

type deviceDetailFetchedMsg struct {
	data json.RawMessage
	err  error
}

func (m model) fetchDevices() tea.Cmd {
	return func() tea.Msg {
		resp, err := m.client.SearchDevices(nil)
		if err != nil {
			return devicesFetchedMsg{err: err}
		}
		return devicesFetchedMsg{items: resp.Devices}
	}
}

func (m model) fetchDeviceDetail(id string) tea.Cmd {
	return func() tea.Msg {
		orgName := ""
		if m.profile != nil {
			orgName = m.profile.Organization
		}
		if orgName == "" {
			// Try to extract from the device summary
			for _, raw := range m.devices.items {
				s := client.ParseDeviceSummary(raw)
				if s.Identifier == id {
					orgName = s.Org
					break
				}
			}
		}
		if orgName == "" {
			return deviceDetailFetchedMsg{err: fmt.Errorf("organization required (set in profile or --org)")}
		}
		data, err := m.client.GetDevice(orgName, id)
		return deviceDetailFetchedMsg{data: data, err: err}
	}
}

func (m model) updateDevices(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case devicesFetchedMsg:
		m.devices.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.devices.items = msg.items
		m.devices.loaded = true
		m.devices.table = buildDevicesTable(msg.items, m.width)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if len(m.devices.items) > 0 {
				sel := m.devices.table.Cursor()
				if sel < len(m.devices.items) {
					s := client.ParseDeviceSummary(m.devices.items[sel])
					m.navigate(viewDeviceDetail)
					m.devices.loading = true
					return m, m.fetchDeviceDetail(s.Identifier)
				}
			}
		case "r":
			m.devices.loading = true
			return m, m.fetchDevices()
		}
	}

	var cmd tea.Cmd
	m.devices.table, cmd = m.devices.table.Update(msg)
	return m, cmd
}

func (m model) updateDeviceDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case deviceDetailFetchedMsg:
		m.devices.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.deviceDetail.data = msg.data
		m.deviceDetail.summary = client.ParseDeviceSummary(msg.data)
		// Pretty-print JSON for display
		var pretty json.RawMessage
		if json.Unmarshal(msg.data, &pretty) == nil {
			formatted, err := json.MarshalIndent(pretty, "", "  ")
			if err == nil {
				m.deviceDetail.content = string(formatted)
			}
		}
		return m, nil
	}

	return m, nil
}

func buildDevicesTable(items []json.RawMessage, width int) table.Model {
	columns := []table.Column{
		{Title: "Identifier", Width: 25},
		{Title: "Name", Width: 20},
		{Title: "Organization", Width: 20},
		{Title: "State", Width: 15},
	}
	if width > 0 {
		available := width - 10
		columns[0].Width = available * 30 / 100
		columns[1].Width = available * 25 / 100
		columns[2].Width = available * 25 / 100
		columns[3].Width = available * 20 / 100
	}

	rows := make([]table.Row, len(items))
	for i, raw := range items {
		s := client.ParseDeviceSummary(raw)
		rows[i] = table.Row{s.Identifier, s.Name, s.Org, s.Status}
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(min(len(rows)+1, 20)),
	)
	s := table.DefaultStyles()
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderBottom(true).Bold(true).Foreground(accent)
	s.Selected = s.Selected.Foreground(highlight).Bold(true)
	t.SetStyles(s)
	return t
}

func (m model) viewDevicesScreen() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("  Devices"))
	b.WriteString("\n")

	if m.devices.loading {
		b.WriteString(dimStyle.Render("  Loading..."))
		return b.String()
	}

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %v", m.err)))
		b.WriteString(helpStyle.Render("\n  r retry • esc back"))
		return b.String()
	}

	if m.devices.loaded {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %d devices", len(m.devices.items))))
		b.WriteString("\n\n")
		b.WriteString(m.devices.table.View())
	}

	b.WriteString(helpStyle.Render("\n  ↑↓/jk navigate • enter detail • r refresh • esc back"))

	return b.String()
}

func (m model) viewDeviceDetailScreen() string {
	var b strings.Builder

	if m.devices.loading {
		b.WriteString(dimStyle.Render("  Loading..."))
		return b.String()
	}

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %v", m.err)))
		b.WriteString(helpStyle.Render("\n  esc back"))
		return b.String()
	}

	s := m.deviceDetail.summary
	b.WriteString(titleStyle.Render(fmt.Sprintf("  %s", s.Identifier)))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("  %s • %s", s.Org, s.Status)))
	b.WriteString("\n\n")

	// Show truncated JSON (fit to screen)
	content := m.deviceDetail.content
	lines := strings.Split(content, "\n")
	maxLines := m.height - 8
	if maxLines < 5 {
		maxLines = 5
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		lines = append(lines, dimStyle.Render(fmt.Sprintf("  ... (%d more lines)", len(strings.Split(content, "\n"))-maxLines)))
	}
	b.WriteString(strings.Join(lines, "\n"))

	b.WriteString(helpStyle.Render("\n  esc back"))

	return b.String()
}
