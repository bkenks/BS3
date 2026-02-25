package secrets

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bkenks/bs3-cli/internal/apiclient"
	"github.com/bkenks/bs3-cli/internal/tui/ui/events"
	"github.com/bkenks/bs3-cli/internal/tui/ui/shared"
)

type CreateDialog struct {
	inputs    []textinput.Model
	focusIdx  int
	statusMsg string
	client    apiclient.Client
}

func NewCreateDialog(client apiclient.Client) *CreateDialog {
	name := textinput.New()
	name.Placeholder = "name"
	name.CharLimit = 128

	val := textinput.New()
	val.Placeholder = "value"
	val.EchoMode = textinput.EchoPassword
	val.EchoCharacter = '•'
	val.CharLimit = 512

	return &CreateDialog{inputs: []textinput.Model{name, val}, client: client}
}

func (m *CreateDialog) Init() tea.Cmd { return m.inputs[0].Focus() }

func (m *CreateDialog) SetStatusMsg(msg string) { m.statusMsg = msg }

func (m *CreateDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		return m, events.CmdSetState(shared.StateSecretsList)
	case "tab":
		m.focusIdx = (m.focusIdx + 1) % len(m.inputs)
		return m, m.cycleFocus()
	case "enter":
		if m.focusIdx < len(m.inputs)-1 {
			m.focusIdx++
			return m, m.cycleFocus()
		}
		name, value := m.inputs[0].Value(), m.inputs[1].Value()
		if name == "" || value == "" {
			m.statusMsg = "name and value are required"
			return m, nil
		}
		return m, m.storeCmd(name, value)
	default:
		var cmd tea.Cmd
		m.inputs[m.focusIdx], cmd = m.inputs[m.focusIdx].Update(msg)
		return m, cmd
	}
}

func (m *CreateDialog) cycleFocus() tea.Cmd {
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

func (m *CreateDialog) storeCmd(name, value string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.StoreSecret(name, value); err != nil {
			return events.APIError{Err: err}
		}
		return events.SecretStored{}
	}
}

func (m *CreateDialog) View() string {
	body := fmt.Sprintf("Name\n%s\n\nValue\n%s", m.inputs[0].View(), m.inputs[1].View())
	if m.statusMsg != "" {
		body += "\n\n" + m.statusMsg
	}
	body += "\n\nenter · next/submit   tab · cycle   esc · cancel"
	dialog := shared.DialogStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, shared.DialogTitleStyle.Render("New Secret"), body),
	)
	return lipgloss.Place(shared.WindowSize.Width, shared.WindowSize.Height, lipgloss.Center, lipgloss.Center, dialog)
}
