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

type datamodelsModel struct {
	table   table.Model
	items   []client.Datamodel
	loaded  bool
	loading bool
}

type datamodelDetailModel struct {
	dm    *client.Datamodel
	table table.Model
}

type datamodelsFetchedMsg struct {
	items []client.Datamodel
	err   error
}

type datamodelDetailFetchedMsg struct {
	dm  *client.Datamodel
	err error
}

func (m model) fetchDatamodels() tea.Cmd {
	return func() tea.Msg {
		resp, err := m.client.SearchDatamodels(nil)
		if err != nil {
			return datamodelsFetchedMsg{err: err}
		}
		return datamodelsFetchedMsg{items: resp.Datamodels}
	}
}

func (m model) fetchDatamodelDetail(orgName, id string) tea.Cmd {
	return func() tea.Msg {
		dm, err := m.client.GetDatamodel(orgName, id)
		return datamodelDetailFetchedMsg{dm: dm, err: err}
	}
}

func (m model) updateDatamodels(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case datamodelsFetchedMsg:
		m.datamodels.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.datamodels.items = msg.items
		m.datamodels.loaded = true
		m.datamodels.table = buildDatamodelsTable(msg.items, m.width)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if len(m.datamodels.items) > 0 {
				sel := m.datamodels.table.Cursor()
				if sel < len(m.datamodels.items) {
					dm := m.datamodels.items[sel]
					m.navigate(viewDatamodelDetail)
					m.datamodels.loading = true
					return m, m.fetchDatamodelDetail(dm.OrganizationName, dm.Identifier)
				}
			}
		case "r":
			m.datamodels.loading = true
			return m, m.fetchDatamodels()
		}
	}

	var cmd tea.Cmd
	m.datamodels.table, cmd = m.datamodels.table.Update(msg)
	return m, cmd
}

func (m model) updateDatamodelDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case datamodelDetailFetchedMsg:
		m.datamodels.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.datamodelDetail.dm = msg.dm
		m.datamodelDetail.table = buildDatamodelDetailTable(msg.dm, m.width)
		return m, nil
	}

	var cmd tea.Cmd
	m.datamodelDetail.table, cmd = m.datamodelDetail.table.Update(msg)
	return m, cmd
}

func buildDatamodelsTable(items []client.Datamodel, width int) table.Model {
	columns := []table.Column{
		{Title: "Identifier", Width: 30},
		{Title: "Organization", Width: 20},
		{Title: "Name", Width: 30},
		{Title: "Version", Width: 12},
	}
	if width > 0 {
		available := width - 10
		columns[0].Width = available * 30 / 100
		columns[1].Width = available * 20 / 100
		columns[2].Width = available * 35 / 100
		columns[3].Width = available * 15 / 100
	}

	rows := make([]table.Row, len(items))
	for i, dm := range items {
		rows[i] = table.Row{dm.Identifier, dm.OrganizationName, dm.Name, dm.Version}
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

func buildDatamodelDetailTable(dm *client.Datamodel, width int) table.Model {
	columns := []table.Column{
		{Title: "Category", Width: 20},
		{Title: "Datastream", Width: 35},
		{Title: "Name", Width: 25},
		{Title: "Period", Width: 10},
		{Title: "Schema", Width: 15},
		{Title: "Access", Width: 10},
	}
	if width > 0 {
		available := width - 10
		columns[0].Width = available * 15 / 100
		columns[1].Width = available * 30 / 100
		columns[2].Width = available * 20 / 100
		columns[3].Width = available * 10 / 100
		columns[4].Width = available * 15 / 100
		columns[5].Width = available * 10 / 100
	}

	var rows []table.Row
	for _, cat := range dm.Categories {
		for _, ds := range cat.Datastreams {
			schemaType := ""
			if ds.Schema != nil {
				var s struct{ Type string }
				if json.Unmarshal(ds.Schema, &s) == nil && s.Type != "" {
					schemaType = s.Type
				}
			}
			rows = append(rows, table.Row{cat.Identifier, ds.Identifier, ds.Name, ds.Period, schemaType, ds.Access})
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

func (m model) viewDatamodelsScreen() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("  Datamodels"))
	b.WriteString("\n")

	if m.datamodels.loading {
		b.WriteString(dimStyle.Render("  Loading..."))
		return b.String()
	}

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %v", m.err)))
		b.WriteString(helpStyle.Render("\n  r retry • esc back"))
		return b.String()
	}

	if m.datamodels.loaded {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %d datamodels", len(m.datamodels.items))))
		b.WriteString("\n\n")
		b.WriteString(m.datamodels.table.View())
	}

	b.WriteString(helpStyle.Render("\n  ↑↓/jk navigate • enter detail • r refresh • esc back"))

	return b.String()
}

func (m model) viewDatamodelDetailScreen() string {
	var b strings.Builder

	if m.datamodels.loading {
		b.WriteString(dimStyle.Render("  Loading..."))
		return b.String()
	}

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %v", m.err)))
		b.WriteString(helpStyle.Render("\n  esc back"))
		return b.String()
	}

	if m.datamodelDetail.dm != nil {
		dm := m.datamodelDetail.dm
		b.WriteString(titleStyle.Render(fmt.Sprintf("  %s", dm.Identifier)))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %s • %s • v%s", dm.OrganizationName, dm.Name, dm.Version)))
		b.WriteString("\n\n")
		b.WriteString(m.datamodelDetail.table.View())
	}

	b.WriteString(helpStyle.Render("\n  ↑↓/jk navigate • esc back"))

	return b.String()
}
