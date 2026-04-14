package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type menuItem struct {
	label string
	view  view
}

type menuModel struct {
	items  []menuItem
	cursor int
}

func newMenuModel() menuModel {
	return menuModel{
		items: []menuItem{
			{"Login", viewLogin},
			{"Datamodels", viewDatamodels},
			{"Devices", viewDevices},
			{"Alarms", viewAlarms},
			{"Time Series", viewTimeSeries},
			{"Datasets", viewDatasets},
			{"Jobs", viewJobs},
		},
	}
}

func (m model) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.menu.cursor > 0 {
				m.menu.cursor--
			}
		case "down", "j":
			if m.menu.cursor < len(m.menu.items)-1 {
				m.menu.cursor++
			}
		case "enter":
			selected := m.menu.items[m.menu.cursor]
			m.navigate(selected.view)
			switch selected.view {
			case viewDatamodels:
				return m, m.fetchDatamodels()
			case viewDevices:
				return m, m.fetchDevices()
			case viewAlarms:
				return m, m.fetchAlarms()
			case viewTimeSeries:
				return m, m.fetchTimeSeries()
			case viewDatasets:
				return m, m.fetchDatasets()
			case viewJobs:
				return m, m.fetchJobs()
			}
			return m, nil
		case "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) viewMenuScreen() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("  og — OpenGate CLI"))
	b.WriteString("\n")

	if m.profile != nil && m.profile.Token != "" {
		b.WriteString(successStyle.Render(fmt.Sprintf("  Connected to %s", m.profile.Host)))
	} else {
		b.WriteString(dimStyle.Render("  Not logged in"))
	}
	b.WriteString("\n\n")

	for i, item := range m.menu.items {
		cursor := "  "
		style := normalStyle
		if i == m.menu.cursor {
			cursor = "▸ "
			style = selectedStyle
		}
		b.WriteString(style.Render(cursor + item.label))
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("\n  ↑↓/jk navigate • enter select • q quit"))

	return b.String()
}
