package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/carlosprados/og-cli/internal/views"
	"github.com/carlosprados/og-cli/pkg/opengate"
	"github.com/carlosprados/og-cli/pkg/query"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// viewPickerModel manages the view picker overlay on the devices screen.
type viewPickerModel struct {
	active  bool
	cursor  int
	options []viewOption
}

type viewOption struct {
	name        string // "" = default columns
	description string
}

const defaultViewLabel = "(default columns)"

func loadViewOptions() ([]viewOption, error) {
	reg, err := views.Load()
	if err != nil {
		return nil, err
	}
	options := []viewOption{{name: "", description: "Identifier, Name, Organization, State"}}
	for _, d := range reg.All() {
		options = append(options, viewOption{name: d.Name, description: d.Description})
	}
	return options, nil
}

// resolveViewClauses expands a view name into select clauses, always
// projecting the device identifier first so row selection keeps working.
func resolveViewClauses(name string) ([]query.SelectClause, error) {
	reg, err := views.Load()
	if err != nil {
		return nil, err
	}
	identifier := query.SelectFromFields([]string{"provision.device.identifier"}, false)
	return reg.ResolveSelect([]string{name}, identifier)
}

// updateDevicesWithViews wraps updateDevicesWithOps adding the 'v' view picker.
func (m model) updateDevicesWithViews(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.viewPicker.active {
		if kmsg, ok := msg.(tea.KeyMsg); ok {
			switch kmsg.String() {
			case "up", "k":
				if m.viewPicker.cursor > 0 {
					m.viewPicker.cursor--
				}
				return m, nil
			case "down", "j":
				if m.viewPicker.cursor < len(m.viewPicker.options)-1 {
					m.viewPicker.cursor++
				}
				return m, nil
			case "enter":
				opt := m.viewPicker.options[m.viewPicker.cursor]
				m.viewPicker.active = false
				m.devices.activeView = opt.name
				m.devices.viewClauses = nil
				if opt.name != "" {
					clauses, err := resolveViewClauses(opt.name)
					if err != nil {
						m.err = err
						return m, nil
					}
					m.devices.viewClauses = clauses
				}
				m.devices.loading = true
				return m, m.fetchDevices()
			case "esc":
				m.viewPicker.active = false
				return m, nil
			}
		}
		return m, nil
	}

	if kmsg, ok := msg.(tea.KeyMsg); ok && kmsg.String() == "v" {
		options, err := loadViewOptions()
		if err != nil {
			m.err = err
			return m, nil
		}
		cursor := 0
		for i, opt := range options {
			if opt.name == m.devices.activeView {
				cursor = i
				break
			}
		}
		m.viewPicker = viewPickerModel{active: true, cursor: cursor, options: options}
		return m, nil
	}

	return m.updateDevicesWithOps(msg)
}

// viewDevicesScreenWithViews wraps the devices screen rendering the picker
// overlay and the active view indicator.
func (m model) viewDevicesScreenWithViews() string {
	if m.viewPicker.active {
		return m.viewViewPicker()
	}

	s := m.viewDevicesScreenWithOps()
	if m.devices.activeView != "" {
		s = strings.Replace(s,
			fmt.Sprintf("%d devices", len(m.devices.items)),
			fmt.Sprintf("%d devices · view: %s", len(m.devices.items), m.devices.activeView),
			1)
	}
	return strings.Replace(s, "r refresh", "v view • r refresh", 1)
}

func (m model) viewViewPicker() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("  Select view"))
	b.WriteString("\n\n")

	for i, opt := range m.viewPicker.options {
		cursor := "  "
		style := normalStyle
		if i == m.viewPicker.cursor {
			cursor = "▸ "
			style = selectedStyle
		}
		label := opt.name
		if label == "" {
			label = defaultViewLabel
		}
		b.WriteString(style.Render(fmt.Sprintf("%s%-14s", cursor, label)))
		b.WriteString(dimStyle.Render(" " + truncate(opt.description, 70)))
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("\n  ↑↓/jk navigate • enter apply • esc cancel"))
	return b.String()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// buildDevicesViewTable renders devices with one column per selected sub-field.
func buildDevicesViewTable(items []json.RawMessage, clauses []query.SelectClause, width int) table.Model {
	type column struct {
		header string
		name   string
		sub    string
	}
	var cols []column
	for _, c := range clauses {
		for _, f := range c.Fields {
			header := f.Alias
			if header == "" {
				header = query.FieldAlias(c.Name)
			}
			cols = append(cols, column{header: header, name: c.Name, sub: f.Field})
		}
	}

	colWidth := 20
	if width > 0 && len(cols) > 0 {
		if w := (width - 10) / len(cols); w > 10 {
			colWidth = w
		}
	}

	columns := make([]table.Column, len(cols))
	for i, c := range cols {
		columns[i] = table.Column{Title: c.header, Width: colWidth}
	}

	rows := make([]table.Row, len(items))
	for i, raw := range items {
		row := make(table.Row, len(cols))
		for j, c := range cols {
			row[j] = opengate.ExtractFlatSub(raw, c.name, c.sub)
		}
		rows[i] = row
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
