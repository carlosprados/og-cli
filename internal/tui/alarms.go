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

type alarmsModel struct {
	table   table.Model
	items   []client.Alarm
	loaded  bool
	loading bool
}

type alarmsFetchedMsg struct {
	items []client.Alarm
	err   error
}

type alarmActionMsg struct {
	action string
	resp   *client.AlarmActionResponse
	err    error
}

func (m model) fetchAlarms() tea.Cmd {
	return func() tea.Msg {
		resp, err := m.client.SearchAlarms(nil)
		if err != nil {
			return alarmsFetchedMsg{err: err}
		}
		return alarmsFetchedMsg{items: resp.Alarms}
	}
}

func (m model) doAlarmAction(action string, id string) tea.Cmd {
	return func() tea.Msg {
		var resp *client.AlarmActionResponse
		var err error
		if action == "attend" {
			resp, err = m.client.AttendAlarms([]string{id}, "")
		} else {
			resp, err = m.client.CloseAlarms([]string{id}, "")
		}
		return alarmActionMsg{action: action, resp: resp, err: err}
	}
}

func (m model) updateAlarms(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case alarmsFetchedMsg:
		m.alarms.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.alarms.items = msg.items
		m.alarms.loaded = true
		m.alarms.table = buildAlarmsTable(msg.items, m.width)
		return m, nil

	case alarmActionMsg:
		m.alarms.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.message = fmt.Sprintf("%s: %d ok, %d errors", msg.action, msg.resp.Result.Successful, msg.resp.Result.Error.Count)
		// Refresh list
		m.alarms.loading = true
		return m, m.fetchAlarms()

	case tea.KeyMsg:
		switch msg.String() {
		case "a":
			if sel := selectedAlarm(m); sel != nil {
				m.alarms.loading = true
				return m, m.doAlarmAction("attend", sel.Identifier)
			}
		case "c":
			if sel := selectedAlarm(m); sel != nil {
				m.alarms.loading = true
				return m, m.doAlarmAction("close", sel.Identifier)
			}
		case "r":
			m.alarms.loading = true
			return m, m.fetchAlarms()
		}
	}

	var cmd tea.Cmd
	m.alarms.table, cmd = m.alarms.table.Update(msg)
	return m, cmd
}

func selectedAlarm(m model) *client.Alarm {
	if len(m.alarms.items) == 0 {
		return nil
	}
	idx := m.alarms.table.Cursor()
	if idx < len(m.alarms.items) {
		return &m.alarms.items[idx]
	}
	return nil
}

func buildAlarmsTable(items []client.Alarm, width int) table.Model {
	columns := []table.Column{
		{Title: "Severity", Width: 12},
		{Title: "Status", Width: 10},
		{Title: "Name", Width: 25},
		{Title: "Entity", Width: 20},
		{Title: "Rule", Width: 20},
		{Title: "Opened", Width: 22},
	}
	if width > 0 {
		available := width - 10
		columns[0].Width = available * 10 / 100
		columns[1].Width = available * 10 / 100
		columns[2].Width = available * 22 / 100
		columns[3].Width = available * 20 / 100
		columns[4].Width = available * 18 / 100
		columns[5].Width = available * 20 / 100
	}

	rows := make([]table.Row, len(items))
	for i, a := range items {
		rows[i] = table.Row{a.Severity, a.Status, a.Name, a.EntityIdentifier, a.Rule, a.OpeningDate}
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

func (m model) viewAlarmsScreen() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("  Alarms"))
	b.WriteString("\n")

	if m.alarms.loading {
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

	if m.alarms.loaded {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %d alarms", len(m.alarms.items))))
		b.WriteString("\n\n")
		if len(m.alarms.items) > 0 {
			b.WriteString(m.alarms.table.View())
			// Show detail of selected alarm
			if sel := selectedAlarm(m); sel != nil {
				b.WriteString("\n")
				detail := formatAlarmDetail(sel)
				b.WriteString(detail)
			}
		}
	}

	b.WriteString(helpStyle.Render("\n  ↑↓/jk navigate • a attend • c close • r refresh • esc back"))

	return b.String()
}

func formatAlarmDetail(a *client.Alarm) string {
	var b strings.Builder
	b.WriteString(dimStyle.Render(fmt.Sprintf("  ID: %s", a.Identifier)))
	if a.Description != "" {
		b.WriteString(dimStyle.Render(fmt.Sprintf(" • %s", a.Description)))
	}

	detail, _ := json.MarshalIndent(a, "  ", "  ")
	_ = detail // available for future expansion
	return b.String()
}
