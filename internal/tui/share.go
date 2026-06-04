package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// sharePromptModel is the email-input overlay shown when sharing a workspace
// from the Workspaces screen ('s' key).
type sharePromptModel struct {
	active      bool
	workspaceID string
	name        string
	input       textinput.Model
}

type workspaceSharedMsg struct {
	id    string
	users []string
	err   error
}

func newSharePrompt(workspaceID, name string) sharePromptModel {
	ti := textinput.New()
	ti.Placeholder = "user1@example.com, user2@example.com (empty = unshare)"
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 60
	return sharePromptModel{active: true, workspaceID: workspaceID, name: name, input: ti}
}

func (m model) shareWorkspace(id string, users []string) tea.Cmd {
	return func() tea.Msg {
		_, err := m.client.ShareWorkspace(id, users, nil)
		return workspaceSharedMsg{id: id, users: users, err: err}
	}
}

// updateWorkspacesWithShare wraps updateWorkspaces adding the 's' share prompt.
func (m model) updateWorkspacesWithShare(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(workspaceSharedMsg); ok {
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		if len(msg.users) == 0 {
			m.message = fmt.Sprintf("Workspace %s unshared", msg.id)
		} else {
			m.message = fmt.Sprintf("Workspace %s shared with %s", msg.id, strings.Join(msg.users, ", "))
		}
		return m, nil
	}

	if m.sharePrompt.active {
		if kmsg, ok := msg.(tea.KeyMsg); ok {
			switch kmsg.String() {
			case "enter":
				var users []string
				for _, u := range strings.Split(m.sharePrompt.input.Value(), ",") {
					if u = strings.TrimSpace(u); u != "" {
						users = append(users, u)
					}
				}
				id := m.sharePrompt.workspaceID
				m.sharePrompt.active = false
				return m, m.shareWorkspace(id, users)
			case "esc":
				m.sharePrompt.active = false
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.sharePrompt.input, cmd = m.sharePrompt.input.Update(msg)
		return m, cmd
	}

	if kmsg, ok := msg.(tea.KeyMsg); ok && kmsg.String() == "s" {
		if len(m.workspaces.items) > 0 {
			sel := m.workspaces.table.Cursor()
			if sel < len(m.workspaces.items) {
				w := m.workspaces.items[sel]
				m.sharePrompt = newSharePrompt(w.ID, w.Name)
				return m, textinput.Blink
			}
		}
	}

	return m.updateWorkspaces(msg)
}

// viewWorkspacesScreenWithShare renders the share prompt over the base screen.
func (m model) viewWorkspacesScreenWithShare() string {
	if m.sharePrompt.active {
		var b strings.Builder
		b.WriteString(titleStyle.Render(fmt.Sprintf("  Share workspace %s", m.sharePrompt.name)))
		b.WriteString("\n\n  Users (comma-separated emails). The list REPLACES current sharing;\n  leave empty and press enter to unshare.\n\n  ")
		b.WriteString(m.sharePrompt.input.View())
		b.WriteString(helpStyle.Render("\n\n  enter apply • esc cancel"))
		return b.String()
	}

	s := m.viewWorkspacesScreen()
	return strings.Replace(s, "r refresh", "s share • r refresh", 1)
}
