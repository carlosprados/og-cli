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

// --- provision functions (provision processors) ---

type provisionModel struct {
	table   table.Model
	items   []json.RawMessage
	loaded  bool
	loading bool
}

type provisionFetchedMsg struct {
	items []json.RawMessage
	err   error
}

func (m model) fetchProvision() tea.Cmd {
	org := ""
	if m.profile != nil {
		org = m.profile.Organization
	}
	return func() tea.Msg {
		if org == "" {
			return provisionFetchedMsg{err: fmt.Errorf("organization required (set it in the profile)")}
		}
		items, err := m.client.ListProvisionProcessors(org)
		if err != nil {
			return provisionFetchedMsg{err: err}
		}
		return provisionFetchedMsg{items: items}
	}
}

func (m model) updateProvision(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case provisionFetchedMsg:
		m.provision.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.provision.items = msg.items
		m.provision.loaded = true
		m.provision.table = buildProvisionTable(msg.items, m.width)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			m.provision.loading = true
			return m, m.fetchProvision()
		}
	}

	var cmd tea.Cmd
	m.provision.table, cmd = m.provision.table.Update(msg)
	return m, cmd
}

func buildProvisionTable(items []json.RawMessage, width int) table.Model {
	columns := []table.Column{
		{Title: "Name", Width: 24},
		{Title: "Sheet", Width: 14},
		{Title: "HeaderRow", Width: 10},
		{Title: "ResultColumn", Width: 16},
		{Title: "Identifier", Width: 38},
	}
	if width > 0 {
		a := width - 10
		columns[0].Width = a * 24 / 100
		columns[1].Width = a * 14 / 100
		columns[2].Width = a * 10 / 100
		columns[3].Width = a * 18 / 100
		columns[4].Width = a * 34 / 100
	}

	rows := make([]table.Row, len(items))
	for i, raw := range items {
		s := opengate.ParseProvisionProcessorSummary(raw)
		rows[i] = table.Row{s.Name, s.SheetName, s.HeaderRow, s.ResultColumnName, s.ProvisionProcessorID}
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

func (m model) viewProvisionScreen() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("  Provision Functions"))
	b.WriteString("\n")

	if m.provision.loading {
		b.WriteString(dimStyle.Render("  Loading..."))
		return b.String()
	}
	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %v", m.err)))
		b.WriteString(helpStyle.Render("\n  r retry • esc back"))
		return b.String()
	}
	if m.provision.loaded {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %d provision functions", len(m.provision.items))))
		b.WriteString("\n\n")
		b.WriteString(m.provision.table.View())
	}

	b.WriteString(helpStyle.Render("\n  ↑↓/jk navigate • r refresh • esc back"))
	b.WriteString(dimStyle.Render("\n  plan/bulk run from the CLI: og provision plan <id> --file data.xlsx"))
	return b.String()
}
