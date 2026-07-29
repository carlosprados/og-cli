package tui

import (
	"fmt"
	"strings"

	"github.com/carlosprados/og-cli/internal/config"
	"github.com/carlosprados/og-cli/pkg/opengate"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type loginModel struct {
	inputs  []textinput.Model
	focused int
	loading bool
	// twoFA becomes true once the server reports a 2FA challenge, revealing the
	// code field. We retry the login carrying the code typed into it.
	twoFA bool
}

const (
	loginFieldEmail = iota
	loginFieldPassword
	loginFieldTwoFA
)

type loginResultMsg struct {
	result *opengate.LoginResult
	// challenge is true when the error is a 2FA challenge (code required or
	// wrong/expired); the view then reveals the code field instead of failing.
	challenge bool
	err       error
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

	twoFA := textinput.New()
	twoFA.Placeholder = "123456"
	twoFA.CharLimit = 6
	twoFA.Width = 40

	return loginModel{
		inputs: []textinput.Model{email, password, twoFA},
	}
}

func (m model) updateLogin(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case loginResultMsg:
		m.login.loading = false
		if msg.err != nil {
			if msg.challenge {
				// 2FA required (or the code was wrong/expired): reveal the code
				// field and let the user type a fresh code, then retry.
				m.login.twoFA = true
				m.err = fmt.Errorf("2FA code required — enter the 6-digit code")
				m.login.focusField(loginFieldTwoFA)
				return m, nil
			}
			m.err = msg.err
			return m, nil
		}
		// Update client and profile with new credentials
		m.profile.Token = msg.result.JWT
		m.profile.APIKey = msg.result.APIKey
		if m.profile.Organization == "" {
			m.profile.Organization = msg.result.Domain
		}
		m.client = opengate.New(m.profile.Host, msg.result.JWT, m.profile.ClientOptions()...)

		// Preserve Email and the stored TOTP secret so token refresh and
		// non-interactive re-login keep working (parity with `og login`).
		_ = config.SaveCredentials(m.profileName, config.Credentials{
			Token:        msg.result.JWT,
			APIKey:       msg.result.APIKey,
			Organization: msg.result.Domain,
			Email:        m.login.inputs[loginFieldEmail].Value(),
			TOTPSecret:   m.profile.TOTPSecret,
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
			m.login.focusField((m.login.focused + 1) % m.login.fieldCount())
			return m, nil
		case "shift+tab", "up":
			m.login.focusField((m.login.focused - 1 + m.login.fieldCount()) % m.login.fieldCount())
			return m, nil
		case "enter":
			email := m.login.inputs[loginFieldEmail].Value()
			password := m.login.inputs[loginFieldPassword].Value()
			if email == "" || password == "" {
				m.err = fmt.Errorf("email and password are required")
				return m, nil
			}
			twoFaCode := ""
			if m.login.twoFA {
				twoFaCode = strings.TrimSpace(m.login.inputs[loginFieldTwoFA].Value())
				if twoFaCode == "" {
					m.err = fmt.Errorf("enter the 6-digit 2FA code")
					return m, nil
				}
			}
			m.login.loading = true
			m.err = nil
			return m, m.doLogin(email, password, twoFaCode)
		}
	}

	// Update the focused input
	var cmd tea.Cmd
	m.login.inputs[m.login.focused], cmd = m.login.inputs[m.login.focused].Update(msg)
	return m, cmd
}

// fieldCount returns the number of navigable fields: email + password, plus the
// 2FA code field once a challenge has revealed it.
func (lm *loginModel) fieldCount() int {
	if lm.twoFA {
		return 3
	}
	return 2
}

// focusField moves focus to field i and blurs the rest.
func (lm *loginModel) focusField(i int) {
	lm.focused = i
	for j := range lm.inputs {
		if j == i {
			lm.inputs[j].Focus()
		} else {
			lm.inputs[j].Blur()
		}
	}
}

func (m model) doLogin(email, password, twoFaCode string) tea.Cmd {
	totpSecret := m.profile.TOTPSecret
	return func() tea.Msg {
		// Hands-off 2FA: if no explicit code was typed but the profile holds a
		// TOTP secret, derive the code ourselves (parity with `og login` and
		// the MCP login tool).
		if twoFaCode == "" && totpSecret != "" {
			if code, gerr := opengate.GenerateTOTPCode(totpSecret); gerr == nil {
				twoFaCode = code
			}
		}
		c := opengate.New(m.profile.Host, "", m.profile.ClientOptions()...)
		result, err := c.Login(m.ctx, email, password, twoFaCode)
		if err != nil && opengate.Is2FAChallenge(err) {
			return loginResultMsg{err: err, challenge: true}
		}
		return loginResultMsg{result: result, err: err}
	}
}

func (m model) viewLoginScreen() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("  Login"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("  Host: %s", m.profile.Host)))
	b.WriteString("\n\n")

	labels := []string{"  Email:    ", "  Password: ", "  2FA code: "}
	for i := 0; i < m.login.fieldCount(); i++ {
		b.WriteString(normalStyle.Render(labels[i]))
		b.WriteString(m.login.inputs[i].View())
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
