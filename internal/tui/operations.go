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

type jobsModel struct {
	table   table.Model
	items   []json.RawMessage
	loaded  bool
	loading bool
}

type jobDetailModel struct {
	jobID      string
	jobData    json.RawMessage
	operations []json.RawMessage
	table      table.Model
	loading    bool
}

type tasksModel struct {
	table   table.Model
	items   []json.RawMessage
	loaded  bool
	loading bool
}

type jobsFetchedMsg struct {
	items []json.RawMessage
	err   error
}

type jobDetailFetchedMsg struct {
	jobID      string
	jobData    json.RawMessage
	operations []json.RawMessage
	err        error
}

type jobCreatedMsg struct {
	data json.RawMessage
	err  error
}

type tasksFetchedMsg struct {
	items []json.RawMessage
	err   error
}

func (m model) fetchJobs() tea.Cmd {
	return func() tea.Msg {
		resp, err := m.client.SearchJobs(nil)
		if err != nil {
			return jobsFetchedMsg{err: err}
		}
		return jobsFetchedMsg{items: resp.Jobs}
	}
}

func (m model) fetchJobDetail(jobID string) tea.Cmd {
	return func() tea.Msg {
		jobData, err := m.client.GetJob(jobID)
		if err != nil {
			return jobDetailFetchedMsg{jobID: jobID, err: err}
		}
		opsResp, err := m.client.GetJobOperations(jobID)
		var ops []json.RawMessage
		if err == nil && opsResp != nil {
			ops = opsResp.Operations
		}
		return jobDetailFetchedMsg{jobID: jobID, jobData: jobData, operations: ops}
	}
}

func (m model) createQuickJob(operationName string, deviceID string) tea.Cmd {
	return func() tea.Msg {
		job := map[string]any{
			"job": map[string]any{
				"request": map[string]any{
					"name":       operationName,
					"parameters": map[string]any{},
					"active":     true,
					"schedule": map[string]any{
						"stop": map[string]any{"delayed": 90000},
					},
					"operationParameters": map[string]any{
						"timeout": 85000,
						"retries": 0,
					},
					"target": map[string]any{
						"append": map[string]any{
							"entities": []string{deviceID},
						},
					},
				},
			},
		}
		body, _ := json.Marshal(job)
		data, err := m.client.CreateJob(body)
		return jobCreatedMsg{data: data, err: err}
	}
}

func (m model) fetchTasks() tea.Cmd {
	return func() tea.Msg {
		resp, err := m.client.SearchTasks(nil)
		if err != nil {
			return tasksFetchedMsg{err: err}
		}
		return tasksFetchedMsg{items: resp.Tasks}
	}
}

// --- jobs list ---

func (m model) updateJobs(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case jobsFetchedMsg:
		m.jobs.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.jobs.items = msg.items
		m.jobs.loaded = true
		m.jobs.table = buildJobsTable(msg.items, m.width)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if len(m.jobs.items) > 0 {
				sel := m.jobs.table.Cursor()
				if sel < len(m.jobs.items) {
					jobID := extractJobField(m.jobs.items[sel], "id")
					m.navigate(viewJobDetail)
					m.jobDetail.loading = true
					return m, m.fetchJobDetail(jobID)
				}
			}
		case "r":
			m.jobs.loading = true
			return m, m.fetchJobs()
		}
	}

	var cmd tea.Cmd
	m.jobs.table, cmd = m.jobs.table.Update(msg)
	return m, cmd
}

// --- job detail ---

func (m model) updateJobDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case jobDetailFetchedMsg:
		m.jobDetail.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.jobDetail.jobID = msg.jobID
		m.jobDetail.jobData = msg.jobData
		m.jobDetail.operations = msg.operations
		m.jobDetail.table = buildOperationsTable(msg.operations, m.width)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			m.jobDetail.loading = true
			return m, m.fetchJobDetail(m.jobDetail.jobID)
		}
	}

	var cmd tea.Cmd
	m.jobDetail.table, cmd = m.jobDetail.table.Update(msg)
	return m, cmd
}

// --- tasks ---

func (m model) updateTasks(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tasksFetchedMsg:
		m.tasks.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.tasks.items = msg.items
		m.tasks.loaded = true
		m.tasks.table = buildTasksTable(msg.items, m.width)
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "r" {
			m.tasks.loading = true
			return m, m.fetchTasks()
		}
	}

	var cmd tea.Cmd
	m.tasks.table, cmd = m.tasks.table.Update(msg)
	return m, cmd
}

// --- devices: launch operation ---

// operationMenuModel manages the operation picker overlay on the devices screen.
type operationMenuModel struct {
	active   bool
	cursor   int
	deviceID string
	options  []string
}

var defaultOperations = []string{
	"REBOOT_EQUIPMENT",
	"EQUIPMENT_DIAGNOSTIC",
}

func (m model) updateDevicesWithOps(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle job creation result
	if msg, ok := msg.(jobCreatedMsg); ok {
		m.opMenu.active = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.message = "Job created successfully"
		}
		return m, nil
	}

	// Operation menu is active
	if m.opMenu.active {
		if kmsg, ok := msg.(tea.KeyMsg); ok {
			switch kmsg.String() {
			case "up", "k":
				if m.opMenu.cursor > 0 {
					m.opMenu.cursor--
				}
				return m, nil
			case "down", "j":
				if m.opMenu.cursor < len(m.opMenu.options)-1 {
					m.opMenu.cursor++
				}
				return m, nil
			case "enter":
				op := m.opMenu.options[m.opMenu.cursor]
				m.opMenu.active = false
				return m, m.createQuickJob(op, m.opMenu.deviceID)
			case "esc":
				m.opMenu.active = false
				return m, nil
			}
		}
		return m, nil
	}

	// Normal devices update, but intercept 'o' key for operations
	if kmsg, ok := msg.(tea.KeyMsg); ok && kmsg.String() == "o" {
		if len(m.devices.items) > 0 {
			sel := m.devices.table.Cursor()
			if sel < len(m.devices.items) {
				s := client.ParseDeviceSummary(m.devices.items[sel])
				m.opMenu = operationMenuModel{
					active:   true,
					cursor:   0,
					deviceID: s.Identifier,
					options:  defaultOperations,
				}
				return m, nil
			}
		}
	}

	return m.updateDevices(msg)
}

// --- table builders ---

func extractJobField(raw json.RawMessage, path ...string) string {
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return ""
	}
	var current any = m
	for _, key := range path {
		cm, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = cm[key]
	}
	if s, ok := current.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", current)
}

func buildJobsTable(items []json.RawMessage, width int) table.Model {
	columns := []table.Column{
		{Title: "ID", Width: 38},
		{Title: "Operation", Width: 25},
		{Title: "Status", Width: 15},
	}
	if width > 0 {
		a := width - 10
		columns[0].Width = a * 45 / 100
		columns[1].Width = a * 30 / 100
		columns[2].Width = a * 25 / 100
	}

	rows := make([]table.Row, len(items))
	for i, raw := range items {
		rows[i] = table.Row{
			extractJobField(raw, "id"),
			extractJobField(raw, "request", "name"),
			extractJobField(raw, "report", "summary", "status"),
		}
	}

	t := table.New(table.WithColumns(columns), table.WithRows(rows), table.WithFocused(true), table.WithHeight(min(len(rows)+1, 20)))
	s := table.DefaultStyles()
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderBottom(true).Bold(true).Foreground(accent)
	s.Selected = s.Selected.Foreground(highlight).Bold(true)
	t.SetStyles(s)
	return t
}

func buildOperationsTable(items []json.RawMessage, width int) table.Model {
	columns := []table.Column{
		{Title: "Entity", Width: 20},
		{Title: "Operation", Width: 22},
		{Title: "Status", Width: 24},
		{Title: "Date", Width: 22},
	}
	if width > 0 {
		a := width - 10
		columns[0].Width = a * 22 / 100
		columns[1].Width = a * 24 / 100
		columns[2].Width = a * 28 / 100
		columns[3].Width = a * 26 / 100
	}

	rows := make([]table.Row, len(items))
	for i, raw := range items {
		rows[i] = table.Row{
			extractJobField(raw, "entityId"),
			extractJobField(raw, "name"),
			extractJobField(raw, "status"),
			extractJobField(raw, "date"),
		}
	}

	t := table.New(table.WithColumns(columns), table.WithRows(rows), table.WithFocused(true), table.WithHeight(min(len(rows)+1, 20)))
	s := table.DefaultStyles()
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderBottom(true).Bold(true).Foreground(accent)
	s.Selected = s.Selected.Foreground(highlight).Bold(true)
	t.SetStyles(s)
	return t
}

func buildTasksTable(items []json.RawMessage, width int) table.Model {
	columns := []table.Column{
		{Title: "ID", Width: 38},
		{Title: "Name", Width: 30},
		{Title: "State", Width: 12},
	}
	if width > 0 {
		a := width - 10
		columns[0].Width = a * 45 / 100
		columns[1].Width = a * 35 / 100
		columns[2].Width = a * 20 / 100
	}

	rows := make([]table.Row, len(items))
	for i, raw := range items {
		rows[i] = table.Row{
			extractJobField(raw, "id"),
			extractJobField(raw, "name"),
			extractJobField(raw, "state"),
		}
	}

	t := table.New(table.WithColumns(columns), table.WithRows(rows), table.WithFocused(true), table.WithHeight(min(len(rows)+1, 20)))
	s := table.DefaultStyles()
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderBottom(true).Bold(true).Foreground(accent)
	s.Selected = s.Selected.Foreground(highlight).Bold(true)
	t.SetStyles(s)
	return t
}

// --- views ---

func (m model) viewJobsScreen() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("  Jobs"))
	b.WriteString("\n")
	if m.jobs.loading {
		b.WriteString(dimStyle.Render("  Loading..."))
		return b.String()
	}
	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %v", m.err)))
		b.WriteString(helpStyle.Render("\n  r retry • esc back"))
		return b.String()
	}
	if m.jobs.loaded {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %d jobs", len(m.jobs.items))))
		b.WriteString("\n\n")
		b.WriteString(m.jobs.table.View())
	}
	b.WriteString(helpStyle.Render("\n  ↑↓/jk navigate • enter detail • r refresh • esc back"))
	return b.String()
}

func (m model) viewJobDetailScreen() string {
	var b strings.Builder

	if m.jobDetail.loading {
		b.WriteString(dimStyle.Render("  Loading..."))
		return b.String()
	}

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %v", m.err)))
		b.WriteString(helpStyle.Render("\n  esc back"))
		return b.String()
	}

	// Job header
	opName := extractJobField(m.jobDetail.jobData, "request", "name")
	status := extractJobField(m.jobDetail.jobData, "report", "summary", "status")
	total := extractJobField(m.jobDetail.jobData, "report", "summary", "total")

	b.WriteString(titleStyle.Render(fmt.Sprintf("  %s", opName)))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("  ID: %s • Status: %s • Total: %s", m.jobDetail.jobID, status, total)))
	b.WriteString("\n\n")

	if len(m.jobDetail.operations) > 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %d operations", len(m.jobDetail.operations))))
		b.WriteString("\n\n")
		b.WriteString(m.jobDetail.table.View())
	} else {
		b.WriteString(dimStyle.Render("  No operations."))
	}

	b.WriteString(helpStyle.Render("\n  ↑↓/jk navigate • r refresh • esc back"))
	return b.String()
}

func (m model) viewDevicesScreenWithOps() string {
	if m.opMenu.active {
		return m.viewOperationMenu()
	}
	// Reuse the base view but change the help text
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

	if m.message != "" {
		b.WriteString(successStyle.Render(fmt.Sprintf("  %s", m.message)))
		b.WriteString("\n")
	}

	if m.devices.loaded {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %d devices", len(m.devices.items))))
		b.WriteString("\n\n")
		b.WriteString(m.devices.table.View())
	}

	b.WriteString(helpStyle.Render("\n  ↑↓/jk navigate • enter detail • o launch operation • r refresh • esc back"))

	return b.String()
}

func (m model) viewOperationMenu() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(fmt.Sprintf("  Launch operation on %s", m.opMenu.deviceID)))
	b.WriteString("\n\n")

	for i, op := range m.opMenu.options {
		cursor := "  "
		style := normalStyle
		if i == m.opMenu.cursor {
			cursor = "▸ "
			style = selectedStyle
		}
		b.WriteString(style.Render(cursor + op))
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("\n  ↑↓/jk navigate • enter launch • esc cancel"))
	return b.String()
}

func (m model) viewTasksScreen() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("  Tasks"))
	b.WriteString("\n")
	if m.tasks.loading {
		b.WriteString(dimStyle.Render("  Loading..."))
		return b.String()
	}
	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %v", m.err)))
		b.WriteString(helpStyle.Render("\n  r retry • esc back"))
		return b.String()
	}
	if m.tasks.loaded {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %d tasks", len(m.tasks.items))))
		b.WriteString("\n\n")
		b.WriteString(m.tasks.table.View())
	}
	b.WriteString(helpStyle.Render("\n  ↑↓/jk navigate • r refresh • esc back"))
	return b.String()
}
