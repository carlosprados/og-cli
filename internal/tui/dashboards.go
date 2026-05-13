package tui

import (
	"fmt"
	"strings"

	"github.com/carlosprados/og-cli/internal/client"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type dashboardDetailModel struct {
	dash    *client.Dashboard
	table   table.Model
	loading bool
}

type dashboardDetailFetchedMsg struct {
	dash *client.Dashboard
	err  error
}

func (m model) fetchDashboardDetail(id string) tea.Cmd {
	return func() tea.Msg {
		d, err := m.client.GetDashboard(id)
		return dashboardDetailFetchedMsg{dash: d, err: err}
	}
}

func (m model) updateDashboardDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case dashboardDetailFetchedMsg:
		m.dashboardDetail.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.dashboardDetail.dash = msg.dash
		m.dashboardDetail.table = buildDashboardDetailTable(msg.dash, m.width)
		return m, nil
	}

	var cmd tea.Cmd
	m.dashboardDetail.table, cmd = m.dashboardDetail.table.Update(msg)
	return m, cmd
}

func buildDashboardDetailTable(d *client.Dashboard, width int) table.Model {
	columns := []table.Column{
		{Title: "Widget Type", Width: 25},
		{Title: "WID", Width: 20},
		{Title: "Position", Width: 12},
		{Title: "Size", Width: 12},
	}
	if width > 0 {
		available := width - 10
		columns[0].Width = available * 35 / 100
		columns[1].Width = available * 30 / 100
		columns[2].Width = available * 15 / 100
		columns[3].Width = available * 15 / 100
	}

	rows := make([]table.Row, 0, len(d.Grid))
	for _, g := range d.Grid {
		wtype := ""
		wid := ""
		if g.Definition != nil {
			wtype = g.Definition.Type
			wid = g.Definition.Wid
		}
		pos := fmt.Sprintf("%d,%d", g.X, g.Y)
		size := fmt.Sprintf("%dx%d", g.W, g.H)
		rows = append(rows, table.Row{wtype, wid, pos, size})
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

func (m model) viewDashboardDetailScreen() string {
	var b strings.Builder

	if m.dashboardDetail.loading {
		b.WriteString(dimStyle.Render("  Loading..."))
		return b.String()
	}

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %v", m.err)))
		b.WriteString(helpStyle.Render("\n  esc back"))
		return b.String()
	}

	if m.dashboardDetail.dash != nil {
		d := m.dashboardDetail.dash
		b.WriteString(titleStyle.Render(fmt.Sprintf("  %s", d.Title)))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %s • workspace: %s • %d widgets", d.ID, d.Workspaces, len(d.Grid))))
		b.WriteString("\n\n")
		b.WriteString(m.dashboardDetail.table.View())
	}

	b.WriteString(helpStyle.Render("\n  ↑↓/jk navigate • esc back"))

	return b.String()
}
