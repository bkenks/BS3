package secrets

import (
	"errors"
	"fmt"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bkenks/bs3-cli/internal/apiclient"
	"github.com/bkenks/bs3-cli/internal/tui/ui/events"
	"github.com/bkenks/bs3-cli/internal/tui/ui/shared"
)

type EditDialog struct {
	name      string
	folder    string
	input     textinput.Model
	statusMsg string
	client    apiclient.Client
}

func NewEditDialog(name, folder string, client apiclient.Client) *EditDialog {
	val := textinput.New()
	val.Placeholder = "new value"
	val.EchoMode = textinput.EchoPassword
	val.EchoCharacter = '•'
	val.CharLimit = 512

	return &EditDialog{name: name, folder: folder, input: val, client: client}
}

func (m *EditDialog) Init() tea.Cmd { return m.input.Focus() }

func (m *EditDialog) SetStatusMsg(msg string) { m.statusMsg = msg }

func (m *EditDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, isKey := msg.(tea.KeyPressMsg)
	if !isKey {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	switch keyMsg.String() {
	case "esc":
		return m, events.CmdSetState(shared.StateSecretsList)
	case "enter":
		value := m.input.Value()
		if value == "" {
			m.statusMsg = "value is required"
			return m, nil
		}
		return m, m.editCmd(value)
	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
}

func (m *EditDialog) editCmd(value string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.EditSecret(m.name, m.folder, value); err != nil {
			if errors.Is(err, apiclient.ErrSecretNotFound) {
				return events.APIError{Err: fmt.Errorf(
					"secret %q no longer exists", m.name)}
			}
			return events.APIError{Err: err}
		}
		return events.SecretEdited{}
	}
}

func (m *EditDialog) View() tea.View {
	folder := m.folder
	if folder == "" {
		folder = "(ungrouped)"
	}
	body := fmt.Sprintf("Name:   %s\nFolder: %s\n\nNew Value\n%s",
		m.name, folder, m.input.View())
	if m.statusMsg != "" {
		body += "\n\n" + m.statusMsg
	}
	body += "\n\nenter · save   esc · cancel"
	dialog := shared.DialogStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, shared.DialogTitleStyle.Render("Edit Secret"), body),
	)
	return tea.NewView(lipgloss.Place(shared.WindowSize.Width, shared.WindowSize.Height, lipgloss.Center, lipgloss.Center, dialog))
}
