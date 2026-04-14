package tui

import (
	"fmt"
	"strings"

	"github.com/carlosprados/og-cli/internal/client"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type datasetsModel struct {
	table   table.Model
	items   []client.Dataset
	loaded  bool
	loading bool
}

type datasetsDataModel struct {
	ds      *client.Dataset
	data    *client.DatasetDataResponse
	table   table.Model
	loading bool
}

type datasetsFetchedMsg struct {
	items []client.Dataset
	err   error
}

type datasetsDataFetchedMsg struct {
	ds   *client.Dataset
	data *client.DatasetDataResponse
	err  error
}

func (m model) fetchDatasets() tea.Cmd {
	return func() tea.Msg {
		orgName := ""
		if m.profile != nil {
			orgName = m.profile.Organization
		}
		if orgName == "" {
			return datasetsFetchedMsg{err: fmt.Errorf("organization required")}
		}
		resp, err := m.client.ListDatasets(orgName)
		if err != nil {
			return datasetsFetchedMsg{err: err}
		}
		return datasetsFetchedMsg{items: resp.Datasets}
	}
}

func (m model) fetchDatasetData(ds *client.Dataset) tea.Cmd {
	return func() tea.Msg {
		orgName := ""
		if m.profile != nil {
			orgName = m.profile.Organization
		}
		if orgName == "" {
			return datasetsDataFetchedMsg{err: fmt.Errorf("organization required")}
		}
		data, err := m.client.QueryDatasetData(orgName, ds.Identifier, nil)
		return datasetsDataFetchedMsg{ds: ds, data: data, err: err}
	}
}

// --- list view ---

func (m model) updateDatasets(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case datasetsFetchedMsg:
		m.datasets.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.datasets.items = msg.items
		m.datasets.loaded = true
		m.datasets.table = buildDatasetsTable(msg.items, m.width)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if len(m.datasets.items) > 0 {
				sel := m.datasets.table.Cursor()
				if sel < len(m.datasets.items) {
					ds := m.datasets.items[sel]
					m.navigate(viewDatasetData)
					m.dsData.loading = true
					return m, m.fetchDatasetData(&ds)
				}
			}
		case "r":
			m.datasets.loading = true
			return m, m.fetchDatasets()
		}
	}

	var cmd tea.Cmd
	m.datasets.table, cmd = m.datasets.table.Update(msg)
	return m, cmd
}

// --- data view ---

func (m model) updateDatasetData(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case datasetsDataFetchedMsg:
		m.dsData.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.dsData.ds = msg.ds
		m.dsData.data = msg.data
		m.dsData.table = buildDatasetDataTable(msg.data, m.width)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			if m.dsData.ds != nil {
				m.dsData.loading = true
				return m, m.fetchDatasetData(m.dsData.ds)
			}
		}
	}

	var cmd tea.Cmd
	m.dsData.table, cmd = m.dsData.table.Update(msg)
	return m, cmd
}

// --- table builders ---

func buildDatasetsTable(items []client.Dataset, width int) table.Model {
	columns := []table.Column{
		{Title: "Identifier", Width: 28},
		{Title: "Name", Width: 30},
		{Title: "Description", Width: 30},
		{Title: "Columns", Width: 8},
	}
	if width > 0 {
		available := width - 10
		columns[0].Width = available * 28 / 100
		columns[1].Width = available * 28 / 100
		columns[2].Width = available * 34 / 100
		columns[3].Width = available * 10 / 100
	}

	rows := make([]table.Row, len(items))
	for i, ds := range items {
		rows[i] = table.Row{ds.Identifier, ds.Name, ds.Description, fmt.Sprintf("%d", len(ds.Columns))}
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

func buildDatasetDataTable(data *client.DatasetDataResponse, width int) table.Model {
	if data == nil || len(data.Columns) == 0 {
		return table.New()
	}

	colCount := len(data.Columns)
	colWidth := 20
	if width > 0 && colCount > 0 {
		colWidth = (width - 10) / colCount
		if colWidth < 10 {
			colWidth = 10
		}
	}

	columns := make([]table.Column, colCount)
	for i, name := range data.Columns {
		columns[i] = table.Column{Title: name, Width: colWidth}
	}

	rows := make([]table.Row, len(data.Data))
	for i, row := range data.Data {
		cells := make([]string, len(row))
		for j, val := range row {
			cells[j] = fmt.Sprintf("%v", val)
		}
		rows[i] = cells
	}

	height := len(rows) + 1
	if height > 20 {
		height = 20
	}
	if height < 2 {
		height = 2
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(height),
	)
	s := table.DefaultStyles()
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderBottom(true).Bold(true).Foreground(accent)
	s.Selected = s.Selected.Foreground(highlight).Bold(true)
	t.SetStyles(s)
	return t
}

// --- views ---

func (m model) viewDatasetsScreen() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("  Datasets"))
	b.WriteString("\n")

	if m.datasets.loading {
		b.WriteString(dimStyle.Render("  Loading..."))
		return b.String()
	}

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %v", m.err)))
		b.WriteString(helpStyle.Render("\n  r retry • esc back"))
		return b.String()
	}

	if m.datasets.loaded {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %d datasets", len(m.datasets.items))))
		b.WriteString("\n\n")
		b.WriteString(m.datasets.table.View())
	}

	b.WriteString(helpStyle.Render("\n  ↑↓/jk navigate • enter view data • r refresh • esc back"))

	return b.String()
}

func (m model) viewDatasetDataScreen() string {
	var b strings.Builder

	if m.dsData.loading {
		b.WriteString(dimStyle.Render("  Loading data..."))
		return b.String()
	}

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %v", m.err)))
		b.WriteString(helpStyle.Render("\n  esc back"))
		return b.String()
	}

	if m.dsData.ds != nil {
		ds := m.dsData.ds
		b.WriteString(titleStyle.Render(fmt.Sprintf("  %s", ds.Name)))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %s • %s", ds.Identifier, ds.Description)))
		b.WriteString("\n\n")
	}

	if m.dsData.data != nil && len(m.dsData.data.Data) > 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %d rows", len(m.dsData.data.Data))))
		b.WriteString("\n\n")
		b.WriteString(m.dsData.table.View())
	} else {
		b.WriteString(dimStyle.Render("  No data."))
	}

	b.WriteString(helpStyle.Render("\n  ↑↓/jk navigate • r refresh • esc back"))

	return b.String()
}
