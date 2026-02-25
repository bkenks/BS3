package users

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bkenks/bs3-cli/internal/apiclient"
	"github.com/bkenks/bs3-cli/internal/tui/ui/events"
	"github.com/bkenks/bs3-cli/internal/tui/ui/shared"
)

type AddDialog struct {
	inputs    []textinput.Model
	focusIdx  int
	statusMsg string
	client    apiclient.Client
}

func NewAddDialog(client apiclient.Client) *AddDialog {
	username := textinput.New()
	username.Placeholder = "username"
	username.CharLimit = 128

	password := textinput.New()
	password.Placeholder = "password (min 8 characters)"
	password.EchoMode = textinput.EchoPassword
	password.EchoCharacter = '•'
	password.CharLimit = 256

	return &AddDialog{inputs: []textinput.Model{username, password}, client: client}
}

func (m *AddDialog) Init() tea.Cmd           { return m.inputs[0].Focus() }
func (m *AddDialog) SetStatusMsg(msg string) { m.statusMsg = msg }

func (m *AddDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, isKey := msg.(tea.KeyMsg)
	if !isKey {
		var cmds []tea.Cmd
		for i := range m.inputs {
			var cmd tea.Cmd
			m.inputs[i], cmd = m.inputs[i].Update(msg)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)
	}
	switch keyMsg.String() {
	case "esc":
		return m, events.CmdSetState(shared.StateUsersList)
	case "tab":
		m.focusIdx = (m.focusIdx + 1) % len(m.inputs)
		return m, m.cycleFocus()
	case "enter":
		if m.focusIdx < len(m.inputs)-1 {
			m.focusIdx++
			return m, m.cycleFocus()
		}
		username := m.inputs[0].Value()
		password := m.inputs[1].Value()
		if username == "" || password == "" {
			m.statusMsg = "username and password are required"
			return m, nil
		}
		if len(password) < 8 {
			m.statusMsg = "password must be at least 8 characters"
			return m, nil
		}
		return m, m.addCmd(username, password)
	default:
		var cmd tea.Cmd
		m.inputs[m.focusIdx], cmd = m.inputs[m.focusIdx].Update(msg)
		return m, cmd
	}
}

func (m *AddDialog) cycleFocus() tea.Cmd {
	var blinkCmd tea.Cmd
	for i := range m.inputs {
		if i == m.focusIdx {
			blinkCmd = m.inputs[i].Focus()
		} else {
			m.inputs[i].Blur()
		}
	}
	return blinkCmd
}

func (m *AddDialog) addCmd(username, password string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.AddUser(username, password); err != nil {
			return events.APIError{Err: err}
		}
		return events.UserAdded{}
	}
}

func (m *AddDialog) View() string {
	body := fmt.Sprintf("Username\n%s\n\nPassword\n%s", m.inputs[0].View(), m.inputs[1].View())
	if m.statusMsg != "" {
		body += "\n\n" + m.statusMsg
	}
	body += "\n\nenter · next/submit   tab · cycle   esc · cancel"
	dialog := shared.DialogStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, shared.DialogTitleStyle.Render("Add User"), body),
	)
	return lipgloss.Place(shared.WindowSize.Width, shared.WindowSize.Height, lipgloss.Center, lipgloss.Center, dialog)
}
