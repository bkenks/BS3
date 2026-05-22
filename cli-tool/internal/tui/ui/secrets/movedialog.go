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

type MoveDialog struct {
	name          string
	currentFolder string
	input         textinput.Model
	statusMsg     string
	client        apiclient.Client
}

func NewMoveDialog(name, currentFolder string, client apiclient.Client) *MoveDialog {
	folder := textinput.New()
	folder.Placeholder = "folder"
	folder.CharLimit = 128
	folder.SetValue(currentFolder)

	return &MoveDialog{name: name, currentFolder: currentFolder, input: folder, client: client}
}

func (m *MoveDialog) Init() tea.Cmd { return m.input.Focus() }

func (m *MoveDialog) SetStatusMsg(msg string) { m.statusMsg = msg }

func (m *MoveDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		return m, m.moveCmd(m.input.Value())
	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
}

func (m *MoveDialog) moveCmd(toFolder string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.MoveSecret(m.name, m.currentFolder, toFolder); err != nil {
			if errors.Is(err, apiclient.ErrSecretExists) {
				return events.APIError{Err: fmt.Errorf(
					"a secret named %q already exists in folder %q", m.name, toFolder)}
			}
			return events.APIError{Err: err}
		}
		return events.SecretMoved{}
	}
}

func (m *MoveDialog) View() tea.View {
	body := fmt.Sprintf("Move %q to folder\n%s", m.name, m.input.View())
	if m.statusMsg != "" {
		body += "\n\n" + m.statusMsg
	}
	body += "\n\nenter · move   esc · cancel"
	dialog := shared.DialogStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, shared.DialogTitleStyle.Render("Move Secret"), body),
	)
	return tea.NewView(lipgloss.Place(shared.WindowSize.Width, shared.WindowSize.Height, lipgloss.Center, lipgloss.Center, dialog))
}
