package tui

import (
	"fmt"
	"strings"

	"github.com/carlosprados/og-cli/internal/client"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type timeseriesModel struct {
	table   table.Model
	items   []client.TimeSeries
	loaded  bool
	loading bool
}

type timeseriesDataModel struct {
	ts      *client.TimeSeries
	data    *client.TimeSeriesDataResponse
	table   table.Model
	loading bool
}

type timeseriesFetchedMsg struct {
	items []client.TimeSeries
	err   error
}

type timeseriesDataFetchedMsg struct {
	ts   *client.TimeSeries
	data *client.TimeSeriesDataResponse
	err  error
}

func (m model) fetchTimeSeries() tea.Cmd {
	return func() tea.Msg {
		orgName := ""
		if m.profile != nil {
			orgName = m.profile.Organization
		}
		if orgName == "" {
			return timeseriesFetchedMsg{err: fmt.Errorf("organization required")}
		}
		resp, err := m.client.ListTimeSeries(orgName)
		if err != nil {
			return timeseriesFetchedMsg{err: err}
		}
		return timeseriesFetchedMsg{items: resp.Timeseries}
	}
}

func (m model) fetchTimeSeriesData(ts *client.TimeSeries) tea.Cmd {
	return func() tea.Msg {
		orgName := ""
		if m.profile != nil {
			orgName = m.profile.Organization
		}
		if orgName == "" {
			return timeseriesDataFetchedMsg{err: fmt.Errorf("organization required")}
		}
		data, err := m.client.QueryTimeSeriesData(orgName, ts.Identifier, nil)
		return timeseriesDataFetchedMsg{ts: ts, data: data, err: err}
	}
}

// --- list view ---

func (m model) updateTimeSeries(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case timeseriesFetchedMsg:
		m.timeseries.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.timeseries.items = msg.items
		m.timeseries.loaded = true
		m.timeseries.table = buildTimeSeriesTable(msg.items, m.width)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if len(m.timeseries.items) > 0 {
				sel := m.timeseries.table.Cursor()
				if sel < len(m.timeseries.items) {
					ts := m.timeseries.items[sel]
					m.navigate(viewTimeSeriesData)
					m.tsData.loading = true
					return m, m.fetchTimeSeriesData(&ts)
				}
			}
		case "r":
			m.timeseries.loading = true
			return m, m.fetchTimeSeries()
		}
	}

	var cmd tea.Cmd
	m.timeseries.table, cmd = m.timeseries.table.Update(msg)
	return m, cmd
}

// --- data view ---

func (m model) updateTimeSeriesData(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case timeseriesDataFetchedMsg:
		m.tsData.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.tsData.ts = msg.ts
		m.tsData.data = msg.data
		m.tsData.table = buildTimeSeriesDataTable(msg.data, m.width)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			if m.tsData.ts != nil {
				m.tsData.loading = true
				return m, m.fetchTimeSeriesData(m.tsData.ts)
			}
		}
	}

	var cmd tea.Cmd
	m.tsData.table, cmd = m.tsData.table.Update(msg)
	return m, cmd
}

// --- table builders ---

func buildTimeSeriesTable(items []client.TimeSeries, width int) table.Model {
	columns := []table.Column{
		{Title: "Identifier", Width: 28},
		{Title: "Name", Width: 30},
		{Title: "Bucket(s)", Width: 10},
		{Title: "Retention(s)", Width: 12},
		{Title: "Columns", Width: 8},
	}
	if width > 0 {
		available := width - 10
		columns[0].Width = available * 28 / 100
		columns[1].Width = available * 32 / 100
		columns[2].Width = available * 13 / 100
		columns[3].Width = available * 15 / 100
		columns[4].Width = available * 12 / 100
	}

	rows := make([]table.Row, len(items))
	for i, ts := range items {
		rows[i] = table.Row{
			ts.Identifier,
			ts.Name,
			fmt.Sprintf("%d", ts.TimeBucket),
			fmt.Sprintf("%d", ts.Retention),
			fmt.Sprintf("%d", len(ts.Columns)),
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

func buildTimeSeriesDataTable(data *client.TimeSeriesDataResponse, width int) table.Model {
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

func (m model) viewTimeSeriesScreen() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("  Time Series"))
	b.WriteString("\n")

	if m.timeseries.loading {
		b.WriteString(dimStyle.Render("  Loading..."))
		return b.String()
	}

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %v", m.err)))
		b.WriteString(helpStyle.Render("\n  r retry • esc back"))
		return b.String()
	}

	if m.timeseries.loaded {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %d time series", len(m.timeseries.items))))
		b.WriteString("\n\n")
		b.WriteString(m.timeseries.table.View())
	}

	b.WriteString(helpStyle.Render("\n  ↑↓/jk navigate • enter view data • r refresh • esc back"))

	return b.String()
}

func (m model) viewTimeSeriesDataScreen() string {
	var b strings.Builder

	if m.tsData.loading {
		b.WriteString(dimStyle.Render("  Loading data..."))
		return b.String()
	}

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %v", m.err)))
		b.WriteString(helpStyle.Render("\n  esc back"))
		return b.String()
	}

	if m.tsData.ts != nil {
		ts := m.tsData.ts
		b.WriteString(titleStyle.Render(fmt.Sprintf("  %s", ts.Name)))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %s • bucket %ds", ts.Identifier, ts.TimeBucket)))
		b.WriteString("\n")

		// Show column definitions
		var colInfo []string
		for _, col := range ts.Columns {
			colInfo = append(colInfo, fmt.Sprintf("%s(%s)", col.AggregationFunction, col.Name))
		}
		if len(colInfo) > 0 {
			b.WriteString(dimStyle.Render(fmt.Sprintf("  Columns: %s", strings.Join(colInfo, ", "))))
		}
		b.WriteString("\n\n")
	}

	if m.tsData.data != nil && len(m.tsData.data.Data) > 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %d rows", len(m.tsData.data.Data))))
		b.WriteString("\n\n")
		b.WriteString(m.tsData.table.View())
	} else {
		b.WriteString(dimStyle.Render("  No data."))
	}

	b.WriteString(helpStyle.Render("\n  ↑↓/jk navigate • r refresh • esc back"))

	return b.String()
}
