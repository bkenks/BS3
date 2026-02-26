package menu

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bkenks/bs3/internal/apiclient"
	"github.com/bkenks/bs3/internal/constants"
	"github.com/bkenks/bs3/internal/tui/ui/events"
	"github.com/bkenks/bs3/internal/tui/ui/shared"
)

// vaultResultMsg carries the raw response body from a successful /openvault call.
type vaultResultMsg struct{ body string }

type OpenVaultDialog struct {
	inputs    []textinput.Model
	focusIdx  int
	statusMsg string
	result    string
	client    apiclient.Client
}

func NewOpenVaultDialog(client apiclient.Client) *OpenVaultDialog {
	username := textinput.New()
	username.Placeholder = "username"
	username.CharLimit = 128
	username.SetValue(os.Getenv(constants.ENV_VAR_BS3_USERNAME))

	password := textinput.New()
	password.Placeholder = "password"
	password.EchoMode = textinput.EchoPassword
	password.EchoCharacter = '•'
	password.CharLimit = 256
	password.SetValue(os.Getenv(constants.ENV_VAR_BS3_PASSWORD))

	passphrase := textinput.New()
	passphrase.Placeholder = "master passphrase"
	passphrase.EchoMode = textinput.EchoPassword
	passphrase.EchoCharacter = '•'
	passphrase.CharLimit = 512

	return &OpenVaultDialog{
		inputs: []textinput.Model{username, password, passphrase},
		client: client,
	}
}

func (m *OpenVaultDialog) Init() tea.Cmd { return m.inputs[0].Focus() }

func (m *OpenVaultDialog) SetStatusMsg(msg string) { m.statusMsg = msg }

func (m *OpenVaultDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle the async vault result.
	if v, ok := msg.(vaultResultMsg); ok {
		m.result = v.body
		return m, nil
	}

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

	// In result view, only ESC is meaningful.
	if m.result != "" {
		if keyMsg.String() == "esc" {
			return m, events.CmdSetState(shared.StateMainMenu)
		}
		return m, nil
	}

	switch keyMsg.String() {
	case "esc":
		return m, events.CmdSetState(shared.StateMainMenu)
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
		passphrase := m.inputs[2].Value()
		if username == "" || password == "" || passphrase == "" {
			m.statusMsg = "all fields are required"
			return m, nil
		}
		return m, m.openVaultCmd(username, password, passphrase)
	default:
		var cmd tea.Cmd
		m.inputs[m.focusIdx], cmd = m.inputs[m.focusIdx].Update(msg)
		return m, cmd
	}
}

func (m *OpenVaultDialog) cycleFocus() tea.Cmd {
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

func (m *OpenVaultDialog) openVaultCmd(username, password, passphrase string) tea.Cmd {
	return func() tea.Msg {
		body, err := m.client.OpenVault(username, password, passphrase)
		if err != nil {
			return events.APIError{Err: err}
		}
		return vaultResultMsg{body: string(body)}
	}
}

func (m *OpenVaultDialog) View() string {
	var content string

	if m.result != "" {
		content = fmt.Sprintf("Response\n\n%s\n\nesc · back", m.result)
	} else {
		body := fmt.Sprintf(
			"Username\n%s\n\nPassword\n%s\n\nMaster Passphrase\n%s",
			m.inputs[0].View(), m.inputs[1].View(), m.inputs[2].View(),
		)
		if m.statusMsg != "" {
			body += "\n\n" + m.statusMsg
		}
		body += "\n\nenter · next/submit   tab · cycle   esc · cancel"
		content = body
	}

	dialog := shared.DialogStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, shared.DialogTitleStyle.Render("Open Vault"), content),
	)
	return lipgloss.Place(shared.WindowSize.Width, shared.WindowSize.Height, lipgloss.Center, lipgloss.Center, dialog)
}
