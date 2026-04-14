package tui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/carlosprados/og-cli/internal/client"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type devicesModel struct {
	table   table.Model
	items   []json.RawMessage
	loaded  bool
	loading bool
}

// deviceField is a parsed key-value from the flattened device JSON.
type deviceField struct {
	Key   string
	Value string
	Date  string
}

// deviceDetailModel holds parsed device data for the tabbed detail view.
type deviceDetailModel struct {
	data    json.RawMessage
	summary client.DeviceSummary
	tab     int // 0=overview, 1=datastreams, 2=json

	// Parsed sections for overview tab
	identity []deviceField
	state    []deviceField
	admin    []deviceField
	comms    []deviceField
	location string

	// Parsed datastreams for datastreams tab
	datastreams    []deviceField
	datastreamsTbl table.Model

	// Pretty JSON for json tab
	jsonContent string
	jsonScroll  int
}

type devicesFetchedMsg struct {
	items []json.RawMessage
	err   error
}

type deviceDetailFetchedMsg struct {
	data json.RawMessage
	err  error
}

func (m model) fetchDevices() tea.Cmd {
	return func() tea.Msg {
		resp, err := m.client.SearchDevices(nil)
		if err != nil {
			return devicesFetchedMsg{err: err}
		}
		return devicesFetchedMsg{items: resp.Devices}
	}
}

func (m model) fetchDeviceDetail(id string) tea.Cmd {
	return func() tea.Msg {
		orgName := ""
		if m.profile != nil {
			orgName = m.profile.Organization
		}
		if orgName == "" {
			for _, raw := range m.devices.items {
				s := client.ParseDeviceSummary(raw)
				if s.Identifier == id {
					orgName = s.Org
					break
				}
			}
		}
		if orgName == "" {
			return deviceDetailFetchedMsg{err: fmt.Errorf("organization required")}
		}
		data, err := m.client.GetDevice(orgName, id)
		return deviceDetailFetchedMsg{data: data, err: err}
	}
}

func (m model) updateDevices(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case devicesFetchedMsg:
		m.devices.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.devices.items = msg.items
		m.devices.loaded = true
		m.devices.table = buildDevicesTable(msg.items, m.width)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if len(m.devices.items) > 0 {
				sel := m.devices.table.Cursor()
				if sel < len(m.devices.items) {
					s := client.ParseDeviceSummary(m.devices.items[sel])
					m.navigate(viewDeviceDetail)
					m.devices.loading = true
					return m, m.fetchDeviceDetail(s.Identifier)
				}
			}
		case "r":
			m.devices.loading = true
			return m, m.fetchDevices()
		}
	}

	var cmd tea.Cmd
	m.devices.table, cmd = m.devices.table.Update(msg)
	return m, cmd
}

func (m model) updateDeviceDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case deviceDetailFetchedMsg:
		m.devices.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.deviceDetail = parseDeviceDetail(msg.data, m.width)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "1":
			m.deviceDetail.tab = 0
			return m, nil
		case "2":
			m.deviceDetail.tab = 1
			return m, nil
		case "3":
			m.deviceDetail.tab = 2
			m.deviceDetail.jsonScroll = 0
			return m, nil
		case "tab":
			m.deviceDetail.tab = (m.deviceDetail.tab + 1) % 3
			return m, nil
		}

		// Scroll JSON in tab 3
		if m.deviceDetail.tab == 2 {
			switch msg.String() {
			case "down", "j":
				m.deviceDetail.jsonScroll++
				return m, nil
			case "up", "k":
				if m.deviceDetail.jsonScroll > 0 {
					m.deviceDetail.jsonScroll--
				}
				return m, nil
			}
		}

		// Table navigation in tab 2
		if m.deviceDetail.tab == 1 {
			var cmd tea.Cmd
			m.deviceDetail.datastreamsTbl, cmd = m.deviceDetail.datastreamsTbl.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

// --- device detail parser ---

func parseDeviceDetail(data json.RawMessage, width int) deviceDetailModel {
	d := deviceDetailModel{
		data:    data,
		summary: client.ParseDeviceSummary(data),
	}

	// Pretty JSON
	formatted, err := json.MarshalIndent(json.RawMessage(data), "", "  ")
	if err == nil {
		d.jsonContent = string(formatted)
	}

	// Parse flattened fields
	var flat map[string]json.RawMessage
	if json.Unmarshal(data, &flat) != nil {
		return d
	}

	for key, raw := range flat {
		val, date := extractValueAndDate(raw)
		if val == "" {
			continue
		}

		f := deviceField{Key: shortKey(key), Value: val, Date: date}

		switch {
		case strings.HasPrefix(key, "provision.device.identifier") ||
			strings.HasPrefix(key, "provision.device.name") ||
			strings.HasPrefix(key, "provision.device.description") ||
			strings.HasPrefix(key, "provision.device.serialNumber") ||
			strings.HasPrefix(key, "provision.device.specificType") ||
			strings.HasPrefix(key, "provision.device.model"):
			d.identity = append(d.identity, f)

		case strings.HasPrefix(key, "provision.device.administrativeState") ||
			strings.HasPrefix(key, "provision.device.operationalStatus"):
			d.state = append(d.state, f)

		case strings.HasPrefix(key, "provision.administration."):
			d.admin = append(d.admin, f)

		case strings.HasPrefix(key, "provision.device.location"):
			d.location = val

		case strings.HasPrefix(key, "provision.device.communicationModules"):
			d.comms = append(d.comms, f)

		case key == "resourceType":
			// skip

		default:
			// Everything else is a collected datastream
			if !strings.HasPrefix(key, "provision.") {
				d.datastreams = append(d.datastreams, f)
			}
		}
	}

	sort.Slice(d.identity, func(i, j int) bool { return d.identity[i].Key < d.identity[j].Key })
	sort.Slice(d.state, func(i, j int) bool { return d.state[i].Key < d.state[j].Key })
	sort.Slice(d.admin, func(i, j int) bool { return d.admin[i].Key < d.admin[j].Key })
	sort.Slice(d.datastreams, func(i, j int) bool { return d.datastreams[i].Key < d.datastreams[j].Key })

	d.datastreamsTbl = buildDatastreamsTable(d.datastreams, width)

	return d
}

func extractValueAndDate(raw json.RawMessage) (string, string) {
	var wrapper struct {
		Value struct {
			Current struct {
				Value json.RawMessage `json:"value"`
				Date  string          `json:"date"`
			} `json:"_current"`
		} `json:"_value"`
	}
	if json.Unmarshal(raw, &wrapper) == nil && len(wrapper.Value.Current.Value) > 0 {
		return rawValueToString(wrapper.Value.Current.Value), wrapper.Value.Current.Date
	}
	return "", ""
}

func rawValueToString(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	// Try number
	var n float64
	if json.Unmarshal(raw, &n) == nil {
		if n == float64(int64(n)) {
			return fmt.Sprintf("%d", int64(n))
		}
		return fmt.Sprintf("%g", n)
	}
	// Try bool
	var b bool
	if json.Unmarshal(raw, &b) == nil {
		return fmt.Sprintf("%v", b)
	}
	// Object/array: compact JSON
	compact := strings.TrimSpace(string(raw))
	if len(compact) > 60 {
		compact = compact[:57] + "..."
	}
	return compact
}

func shortKey(key string) string {
	// Remove common prefixes for cleaner display
	key = strings.TrimPrefix(key, "provision.device.")
	key = strings.TrimPrefix(key, "provision.administration.")
	key = strings.TrimPrefix(key, "device.")
	return key
}

func buildDatastreamsTable(fields []deviceField, width int) table.Model {
	columns := []table.Column{
		{Title: "Datastream", Width: 35},
		{Title: "Value", Width: 20},
		{Title: "Date", Width: 26},
	}
	if width > 0 {
		a := width - 10
		columns[0].Width = a * 40 / 100
		columns[1].Width = a * 25 / 100
		columns[2].Width = a * 35 / 100
	}

	rows := make([]table.Row, len(fields))
	for i, f := range fields {
		rows[i] = table.Row{f.Key, f.Value, f.Date}
	}

	height := len(rows) + 1
	if height > 18 {
		height = 18
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

// --- table builders ---

func buildDevicesTable(items []json.RawMessage, width int) table.Model {
	columns := []table.Column{
		{Title: "Identifier", Width: 25},
		{Title: "Name", Width: 20},
		{Title: "Organization", Width: 20},
		{Title: "State", Width: 15},
	}
	if width > 0 {
		available := width - 10
		columns[0].Width = available * 30 / 100
		columns[1].Width = available * 25 / 100
		columns[2].Width = available * 25 / 100
		columns[3].Width = available * 20 / 100
	}

	rows := make([]table.Row, len(items))
	for i, raw := range items {
		s := client.ParseDeviceSummary(raw)
		rows[i] = table.Row{s.Identifier, s.Name, s.Org, s.Status}
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

// --- views ---

var (
	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(subtle).
			Padding(0, 1).
			MarginBottom(1)

	cardTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accent)

	fieldKeyStyle = lipgloss.NewStyle().
			Foreground(subtle).
			Width(22)

	fieldValStyle = lipgloss.NewStyle().
			Foreground(text)

	tabActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accent).
			Underline(true)

	tabInactiveStyle = lipgloss.NewStyle().
				Foreground(subtle)
)

func (m model) viewDevicesScreen() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("  Devices"))
	b.WriteString("\n")

	if m.devices.loading {
		b.WriteString(dimStyle.Render("  Loading..."))
		return b.String()
	}

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %v", m.err)))
		b.WriteString(helpStyle.Render("\n  r retry • esc back"))
		return b.String()
	}

	if m.devices.loaded {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %d devices", len(m.devices.items))))
		b.WriteString("\n\n")
		b.WriteString(m.devices.table.View())
	}

	b.WriteString(helpStyle.Render("\n  ↑↓/jk navigate • enter detail • r refresh • esc back"))

	return b.String()
}

func (m model) viewDeviceDetailScreen() string {
	var b strings.Builder

	if m.devices.loading {
		b.WriteString(dimStyle.Render("  Loading..."))
		return b.String()
	}

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %v", m.err)))
		b.WriteString(helpStyle.Render("\n  esc back"))
		return b.String()
	}

	d := m.deviceDetail

	// Header
	b.WriteString(titleStyle.Render(fmt.Sprintf("  %s", d.summary.Identifier)))
	b.WriteString("\n")

	// Tabs
	tabs := []string{"Overview", "Datastreams", "JSON"}
	b.WriteString("  ")
	for i, t := range tabs {
		style := tabInactiveStyle
		if i == d.tab {
			style = tabActiveStyle
		}
		b.WriteString(style.Render(fmt.Sprintf(" %d:%s ", i+1, t)))
		if i < len(tabs)-1 {
			b.WriteString(dimStyle.Render("│"))
		}
	}
	b.WriteString("\n\n")

	// Tab content
	switch d.tab {
	case 0:
		b.WriteString(m.renderOverviewTab())
	case 1:
		b.WriteString(m.renderDatastreamsTab())
	case 2:
		b.WriteString(m.renderJSONTab())
	}

	b.WriteString(helpStyle.Render("\n  1/2/3 or tab switch • ↑↓/jk scroll • esc back"))

	return b.String()
}

func (m model) renderOverviewTab() string {
	var b strings.Builder
	d := m.deviceDetail
	cardWidth := m.width - 6
	if cardWidth < 40 {
		cardWidth = 40
	}

	style := cardStyle.Width(cardWidth)

	// Identity card
	if len(d.identity) > 0 {
		b.WriteString(renderCard(style, "Identity", d.identity))
	}

	// State card
	if len(d.state) > 0 {
		b.WriteString(renderCard(style, "State", d.state))
	}

	// Administration card
	if len(d.admin) > 0 {
		b.WriteString(renderCard(style, "Administration", d.admin))
	}

	// Location
	if d.location != "" {
		var locFields []deviceField
		locFields = append(locFields, deviceField{Key: "coordinates", Value: d.location})
		b.WriteString(renderCard(style, "Location", locFields))
	}

	// Communications (if not too many)
	if len(d.comms) > 0 && len(d.comms) <= 12 {
		b.WriteString(renderCard(style, "Communications", d.comms))
	}

	return b.String()
}

func renderCard(style lipgloss.Style, title string, fields []deviceField) string {
	var content strings.Builder
	content.WriteString(cardTitleStyle.Render(title))
	content.WriteString("\n")
	for _, f := range fields {
		content.WriteString(fieldKeyStyle.Render(f.Key))
		content.WriteString(fieldValStyle.Render(f.Value))
		content.WriteString("\n")
	}
	return style.Render(content.String()) + "\n"
}

func (m model) renderDatastreamsTab() string {
	d := m.deviceDetail
	if len(d.datastreams) == 0 {
		return dimStyle.Render("  No collected datastreams.")
	}
	var b strings.Builder
	b.WriteString(dimStyle.Render(fmt.Sprintf("  %d datastreams", len(d.datastreams))))
	b.WriteString("\n\n")
	b.WriteString(d.datastreamsTbl.View())
	return b.String()
}

func (m model) renderJSONTab() string {
	d := m.deviceDetail
	lines := strings.Split(d.jsonContent, "\n")

	maxLines := m.height - 10
	if maxLines < 5 {
		maxLines = 5
	}

	start := d.jsonScroll
	if start >= len(lines) {
		start = len(lines) - 1
	}
	if start < 0 {
		start = 0
	}

	end := start + maxLines
	if end > len(lines) {
		end = len(lines)
	}

	visible := lines[start:end]

	var b strings.Builder
	b.WriteString(dimStyle.Render(fmt.Sprintf("  Lines %d-%d of %d", start+1, end, len(lines))))
	b.WriteString("\n\n")
	b.WriteString(strings.Join(visible, "\n"))

	return b.String()
}
