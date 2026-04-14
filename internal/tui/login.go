package tui

import (
	"fmt"
	"strings"

	"github.com/carlosprados/og-cli/internal/client"
	"github.com/carlosprados/og-cli/internal/config"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type loginModel struct {
	inputs  []textinput.Model
	focused int
	loading bool
}

type loginResultMsg struct {
	result *client.LoginResult
	err    error
}

func newLoginModel() loginModel {
	email := textinput.New()
	email.Placeholder = "user@example.com"
	email.CharLimit = 100
	email.Width = 40
	email.Focus()

	password := textinput.New()
	password.Placeholder = "password"
	password.EchoMode = textinput.EchoPassword
	password.CharLimit = 100
	password.Width = 40

	return loginModel{
		inputs: []textinput.Model{email, password},
	}
}

func (m model) updateLogin(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case loginResultMsg:
		m.login.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		// Update client and profile with new credentials
		m.profile.Token = msg.result.JWT
		m.profile.APIKey = msg.result.APIKey
		if m.profile.Organization == "" {
			m.profile.Organization = msg.result.Domain
		}
		m.client = client.New(m.profile.Host, msg.result.JWT)

		profileName := m.cfg.DefaultProfile
		_ = config.SaveCredentials(profileName, config.Credentials{
			Token:        msg.result.JWT,
			APIKey:       msg.result.APIKey,
			Organization: msg.result.Domain,
		}, m.cfgPath)

		m.message = "Login successful"
		m.view = viewMenu
		return m, nil

	case tea.KeyMsg:
		if m.login.loading {
			return m, nil
		}
		switch msg.String() {
		case "tab", "down":
			m.login.focused = (m.login.focused + 1) % len(m.login.inputs)
			for i := range m.login.inputs {
				if i == m.login.focused {
					m.login.inputs[i].Focus()
				} else {
					m.login.inputs[i].Blur()
				}
			}
			return m, nil
		case "shift+tab", "up":
			m.login.focused = (m.login.focused - 1 + len(m.login.inputs)) % len(m.login.inputs)
			for i := range m.login.inputs {
				if i == m.login.focused {
					m.login.inputs[i].Focus()
				} else {
					m.login.inputs[i].Blur()
				}
			}
			return m, nil
		case "enter":
			email := m.login.inputs[0].Value()
			password := m.login.inputs[1].Value()
			if email == "" || password == "" {
				m.err = fmt.Errorf("email and password are required")
				return m, nil
			}
			m.login.loading = true
			m.err = nil
			return m, m.doLogin(email, password)
		}
	}

	// Update the focused input
	var cmd tea.Cmd
	m.login.inputs[m.login.focused], cmd = m.login.inputs[m.login.focused].Update(msg)
	return m, cmd
}

func (m model) doLogin(email, password string) tea.Cmd {
	return func() tea.Msg {
		c := client.New(m.profile.Host, "")
		result, err := c.Login(email, password)
		return loginResultMsg{result: result, err: err}
	}
}

func (m model) viewLoginScreen() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("  Login"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("  Host: %s", m.profile.Host)))
	b.WriteString("\n\n")

	labels := []string{"  Email:    ", "  Password: "}
	for i, input := range m.login.inputs {
		b.WriteString(normalStyle.Render(labels[i]))
		b.WriteString(input.View())
		b.WriteString("\n")
	}

	if m.login.loading {
		b.WriteString("\n" + dimStyle.Render("  Authenticating..."))
	}

	if m.err != nil {
		b.WriteString("\n" + errorStyle.Render(fmt.Sprintf("  Error: %v", m.err)))
	}

	if m.message != "" {
		b.WriteString("\n" + successStyle.Render(fmt.Sprintf("  %s", m.message)))
	}

	b.WriteString(helpStyle.Render("\n  tab switch field • enter submit • esc back"))

	return b.String()
}
