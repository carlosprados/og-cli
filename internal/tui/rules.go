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

const tuiDefaultChannel = "default_channel"

// --- rules ---

type rulesModel struct {
	table   table.Model
	items   []json.RawMessage
	loaded  bool
	loading bool
}

type rulesFetchedMsg struct {
	items []json.RawMessage
	err   error
}

type ruleToggledMsg struct {
	id     string
	active bool
	err    error
}

func (m model) fetchRules() tea.Cmd {
	return func() tea.Msg {
		resp, err := m.client.SearchRules(m.ctx, nil)
		if err != nil {
			return rulesFetchedMsg{err: err}
		}
		return rulesFetchedMsg{items: resp.Rules}
	}
}

func (m model) toggleRule(raw json.RawMessage) tea.Cmd {
	s := opengate.ParseRuleSummary(raw)
	org := ""
	if m.profile != nil {
		org = m.profile.Organization
	}
	return func() tea.Msg {
		if org == "" {
			return ruleToggledMsg{err: fmt.Errorf("organization required (set it in the profile)")}
		}
		target := !s.Active
		err := m.client.SetRuleActive(m.ctx, org, tuiDefaultChannel, s.Identifier, target)
		return ruleToggledMsg{id: s.Identifier, active: target, err: err}
	}
}

func (m model) updateRules(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case rulesFetchedMsg:
		m.rules.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.rules.items = msg.items
		m.rules.loaded = true
		m.rules.table = buildRulesTable(msg.items, m.width)
		return m, nil

	case ruleToggledMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		state := "disabled"
		if msg.active {
			state = "enabled"
		}
		m.message = fmt.Sprintf("Rule %s %s", msg.id, state)
		m.rules.loading = true
		return m, m.fetchRules()

	case tea.KeyMsg:
		switch msg.String() {
		case "t":
			if len(m.rules.items) > 0 {
				sel := m.rules.table.Cursor()
				if sel < len(m.rules.items) {
					return m, m.toggleRule(m.rules.items[sel])
				}
			}
		case "r":
			m.rules.loading = true
			return m, m.fetchRules()
		}
	}

	var cmd tea.Cmd
	m.rules.table, cmd = m.rules.table.Update(msg)
	return m, cmd
}

func buildRulesTable(items []json.RawMessage, width int) table.Model {
	columns := []table.Column{
		{Title: "Name", Width: 30},
		{Title: "Mode", Width: 10},
		{Title: "Active", Width: 8},
		{Title: "Trigger", Width: 12},
		{Title: "Identifier", Width: 38},
	}
	if width > 0 {
		a := width - 10
		columns[0].Width = a * 30 / 100
		columns[1].Width = a * 10 / 100
		columns[2].Width = a * 8 / 100
		columns[3].Width = a * 14 / 100
		columns[4].Width = a * 38 / 100
	}

	rows := make([]table.Row, len(items))
	for i, raw := range items {
		s := opengate.ParseRuleSummary(raw)
		rows[i] = table.Row{s.Name, s.Mode, fmt.Sprintf("%v", s.Active), s.RuleTriggerName(), s.Identifier}
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

func (m model) viewRulesScreen() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("  Rules"))
	b.WriteString("\n")

	if m.rules.loading {
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
	if m.rules.loaded {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %d rules", len(m.rules.items))))
		b.WriteString("\n\n")
		b.WriteString(m.rules.table.View())
	}

	b.WriteString(helpStyle.Render("\n  ↑↓/jk navigate • t toggle active • r refresh • esc back"))
	return b.String()
}

// --- operation types ---

type optypesModel struct {
	table   table.Model
	items   []json.RawMessage
	loaded  bool
	loading bool
}

type optypesFetchedMsg struct {
	items []json.RawMessage
	err   error
}

func (m model) fetchOpTypes() tea.Cmd {
	return func() tea.Msg {
		data, err := m.client.SearchOpTypes(m.ctx, nil)
		if err != nil {
			return optypesFetchedMsg{err: err}
		}
		var items []json.RawMessage
		if json.Unmarshal(data, &items) != nil {
			items = nil
		}
		return optypesFetchedMsg{items: items}
	}
}

func (m model) updateOpTypes(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case optypesFetchedMsg:
		m.optypes.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.optypes.items = msg.items
		m.optypes.loaded = true
		m.optypes.table = buildOpTypesTable(msg.items, m.width)
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "r" {
			m.optypes.loading = true
			return m, m.fetchOpTypes()
		}
	}

	var cmd tea.Cmd
	m.optypes.table, cmd = m.optypes.table.Update(msg)
	return m, cmd
}

func buildOpTypesTable(items []json.RawMessage, width int) table.Model {
	columns := []table.Column{
		{Title: "Name", Width: 28},
		{Title: "Title", Width: 25},
		{Title: "ApplicableTo", Width: 25},
		{Title: "Description", Width: 30},
	}
	if width > 0 {
		a := width - 10
		columns[0].Width = a * 25 / 100
		columns[1].Width = a * 22 / 100
		columns[2].Width = a * 23 / 100
		columns[3].Width = a * 30 / 100
	}

	rows := make([]table.Row, len(items))
	for i, raw := range items {
		s := opengate.ParseOpTypeSummary(raw)
		rows[i] = table.Row{s.Name, s.Title, strings.Join(s.ApplicableTo, ","), s.Description}
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

func (m model) viewOpTypesScreen() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("  Operation Types"))
	b.WriteString("\n")

	if m.optypes.loading {
		b.WriteString(dimStyle.Render("  Loading..."))
		return b.String()
	}
	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %v", m.err)))
		b.WriteString(helpStyle.Render("\n  r retry • esc back"))
		return b.String()
	}
	if m.optypes.loaded {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %d operation types", len(m.optypes.items))))
		b.WriteString("\n\n")
		b.WriteString(m.optypes.table.View())
	}

	b.WriteString(helpStyle.Render("\n  ↑↓/jk navigate • r refresh • esc back"))
	return b.String()
}
