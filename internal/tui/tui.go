// Package tui implements the interactive Bubble Tea TUI for og.
package tui

import (
	"github.com/carlosprados/og-cli/internal/client"
	"github.com/carlosprados/og-cli/internal/config"
	tea "github.com/charmbracelet/bubbletea"
)

// view represents the current screen.
type view int

const (
	viewMenu view = iota
	viewLogin
	viewDatamodels
	viewDatamodelDetail
	viewDevices
	viewDeviceDetail
	viewAlarms
	viewTimeSeries
	viewTimeSeriesData
	viewDatasets
	viewDatasetData
)

// model is the top-level Bubble Tea model.
type model struct {
	view     view
	prevView view
	width    int
	height   int

	// config
	cfg     *config.Config
	profile *config.Profile
	cfgPath string

	// client
	client *client.Client

	// sub-models
	menu            menuModel
	login           loginModel
	datamodels      datamodelsModel
	datamodelDetail datamodelDetailModel
	devices         devicesModel
	deviceDetail    deviceDetailModel
	alarms          alarmsModel
	timeseries      timeseriesModel
	tsData          timeseriesDataModel
	datasets        datasetsModel
	dsData          datasetsDataModel

	// status
	err     error
	message string
}

// Run starts the interactive TUI.
func Run(cfg *config.Config, profile *config.Profile, cfgPath string) error {
	c := client.New(profile.Host, profile.Token)

	m := model{
		view:    viewMenu,
		cfg:     cfg,
		profile: profile,
		cfgPath: cfgPath,
		client:  c,
		menu:    newMenuModel(),
		login:   newLoginModel(),
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.view != viewMenu {
				m.err = nil
				m.message = ""
				return m.goBack(), nil
			}
			return m, tea.Quit
		}
	}

	switch m.view {
	case viewMenu:
		return m.updateMenu(msg)
	case viewLogin:
		return m.updateLogin(msg)
	case viewDatamodels:
		return m.updateDatamodels(msg)
	case viewDatamodelDetail:
		return m.updateDatamodelDetail(msg)
	case viewDevices:
		return m.updateDevices(msg)
	case viewDeviceDetail:
		return m.updateDeviceDetail(msg)
	case viewAlarms:
		return m.updateAlarms(msg)
	case viewTimeSeries:
		return m.updateTimeSeries(msg)
	case viewTimeSeriesData:
		return m.updateTimeSeriesData(msg)
	case viewDatasets:
		return m.updateDatasets(msg)
	case viewDatasetData:
		return m.updateDatasetData(msg)
	}

	return m, nil
}

func (m model) View() string {
	switch m.view {
	case viewMenu:
		return m.viewMenuScreen()
	case viewLogin:
		return m.viewLoginScreen()
	case viewDatamodels:
		return m.viewDatamodelsScreen()
	case viewDatamodelDetail:
		return m.viewDatamodelDetailScreen()
	case viewDevices:
		return m.viewDevicesScreen()
	case viewDeviceDetail:
		return m.viewDeviceDetailScreen()
	case viewAlarms:
		return m.viewAlarmsScreen()
	case viewTimeSeries:
		return m.viewTimeSeriesScreen()
	case viewTimeSeriesData:
		return m.viewTimeSeriesDataScreen()
	case viewDatasets:
		return m.viewDatasetsScreen()
	case viewDatasetData:
		return m.viewDatasetDataScreen()
	}
	return ""
}

func (m model) goBack() model {
	switch m.view {
	case viewDatamodelDetail:
		m.view = viewDatamodels
	case viewDeviceDetail:
		m.view = viewDevices
	case viewTimeSeriesData:
		m.view = viewTimeSeries
	case viewDatasetData:
		m.view = viewDatasets
	default:
		m.view = viewMenu
	}
	m.message = ""
	return m
}

func (m *model) navigate(v view) {
	m.prevView = m.view
	m.view = v
	m.err = nil
	m.message = ""
}
