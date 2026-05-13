package tui

import (
	"fmt"
	"strings"

	"github.com/carlosprados/og-cli/internal/client"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type workspacesModel struct {
	table   table.Model
	items   []client.Workspace
	loaded  bool
	loading bool
}

type workspaceDetailModel struct {
	ws    *client.Workspace
	table table.Model
}

type workspacesFetchedMsg struct {
	items []client.Workspace
	err   error
}

type workspaceDetailFetchedMsg struct {
	ws  *client.Workspace
	err error
}

func (m model) fetchWorkspaces() tea.Cmd {
	return func() tea.Msg {
		wss, err := m.client.ListWorkspaces(true)
		if err != nil {
			return workspacesFetchedMsg{err: err}
		}
		return workspacesFetchedMsg{items: wss}
	}
}

func (m model) fetchWorkspaceDetail(id string) tea.Cmd {
	return func() tea.Msg {
		w, err := m.client.GetWorkspace(id, true)
		return workspaceDetailFetchedMsg{ws: w, err: err}
	}
}

func (m model) updateWorkspaces(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case workspacesFetchedMsg:
		m.workspaces.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.workspaces.items = msg.items
		m.workspaces.loaded = true
		m.workspaces.table = buildWorkspacesTable(msg.items, m.width)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if len(m.workspaces.items) > 0 {
				sel := m.workspaces.table.Cursor()
				if sel < len(m.workspaces.items) {
					w := m.workspaces.items[sel]
					m.navigate(viewWorkspaceDetail)
					m.workspaces.loading = true
					return m, m.fetchWorkspaceDetail(w.ID)
				}
			}
		case "r":
			m.workspaces.loading = true
			return m, m.fetchWorkspaces()
		}
	}

	var cmd tea.Cmd
	m.workspaces.table, cmd = m.workspaces.table.Update(msg)
	return m, cmd
}

func (m model) updateWorkspaceDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case workspaceDetailFetchedMsg:
		m.workspaces.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.workspaceDetail.ws = msg.ws
		m.workspaceDetail.table = buildWorkspaceDetailTable(msg.ws, m.width)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.workspaceDetail.ws != nil && len(m.workspaceDetail.ws.Dashboards) > 0 {
				sel := m.workspaceDetail.table.Cursor()
				if sel < len(m.workspaceDetail.ws.Dashboards) {
					wd := m.workspaceDetail.ws.Dashboards[sel]
					id := wd.ID
					if wd.Dashboard != nil && wd.Dashboard.ID != "" {
						id = wd.Dashboard.ID
					}
					if id == "" {
						return m, nil
					}
					m.navigate(viewDashboardDetail)
					m.dashboardDetail.loading = true
					return m, m.fetchDashboardDetail(id)
				}
			}
		}
	}

	var cmd tea.Cmd
	m.workspaceDetail.table, cmd = m.workspaceDetail.table.Update(msg)
	return m, cmd
}

func buildWorkspacesTable(items []client.Workspace, width int) table.Model {
	columns := []table.Column{
		{Title: "ID", Width: 30},
		{Title: "Name", Width: 25},
		{Title: "Owner", Width: 25},
		{Title: "Dashboards", Width: 12},
		{Title: "Domains", Width: 10},
	}
	if width > 0 {
		available := width - 10
		columns[0].Width = available * 30 / 100
		columns[1].Width = available * 25 / 100
		columns[2].Width = available * 25 / 100
		columns[3].Width = available * 10 / 100
		columns[4].Width = available * 10 / 100
	}

	rows := make([]table.Row, len(items))
	for i, w := range items {
		rows[i] = table.Row{
			w.ID,
			w.Name,
			w.Owner,
			fmt.Sprintf("%d", len(w.Dashboards)),
			fmt.Sprintf("%d", len(w.Domains)),
		}
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

func buildWorkspaceDetailTable(w *client.Workspace, width int) table.Model {
	columns := []table.Column{
		{Title: "Dashboard ID", Width: 35},
		{Title: "Title", Width: 35},
		{Title: "Owner", Width: 25},
		{Title: "Position", Width: 10},
	}
	if width > 0 {
		available := width - 10
		columns[0].Width = available * 35 / 100
		columns[1].Width = available * 30 / 100
		columns[2].Width = available * 25 / 100
		columns[3].Width = available * 10 / 100
	}

	rows := make([]table.Row, 0, len(w.Dashboards))
	for _, wd := range w.Dashboards {
		id := wd.ID
		title := ""
		owner := ""
		if wd.Dashboard != nil {
			if wd.Dashboard.ID != "" {
				id = wd.Dashboard.ID
			}
			title = wd.Dashboard.Title
			owner = wd.Dashboard.Owner
		}
		pos := fmt.Sprintf("%d,%d", wd.X, wd.Y)
		rows = append(rows, table.Row{id, title, owner, pos})
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

func (m model) viewWorkspacesScreen() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("  Workspaces"))
	b.WriteString("\n")

	if m.workspaces.loading {
		b.WriteString(dimStyle.Render("  Loading..."))
		return b.String()
	}

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %v", m.err)))
		b.WriteString(helpStyle.Render("\n  r retry • esc back"))
		return b.String()
	}

	if m.workspaces.loaded {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %d workspaces", len(m.workspaces.items))))
		b.WriteString("\n\n")
		b.WriteString(m.workspaces.table.View())
	}

	b.WriteString(helpStyle.Render("\n  ↑↓/jk navigate • enter dashboards • r refresh • esc back"))

	return b.String()
}

func (m model) viewWorkspaceDetailScreen() string {
	var b strings.Builder

	if m.workspaces.loading {
		b.WriteString(dimStyle.Render("  Loading..."))
		return b.String()
	}

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %v", m.err)))
		b.WriteString(helpStyle.Render("\n  esc back"))
		return b.String()
	}

	if m.workspaceDetail.ws != nil {
		w := m.workspaceDetail.ws
		b.WriteString(titleStyle.Render(fmt.Sprintf("  %s", w.Name)))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %s • owner: %s • %d dashboards", w.ID, w.Owner, len(w.Dashboards))))
		b.WriteString("\n\n")
		b.WriteString(m.workspaceDetail.table.View())
	}

	b.WriteString(helpStyle.Render("\n  ↑↓/jk navigate • enter open dashboard • esc back"))

	return b.String()
}
