package secrets

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bkenks/bs3/internal/apiclient"
	"github.com/bkenks/bs3/internal/tui/ui/events"
	"github.com/bkenks/bs3/internal/tui/ui/shared"
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
	keyMsg, isKey := msg.(tea.KeyMsg)
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

func (m *MoveDialog) View() string {
	body := fmt.Sprintf("Move %q to folder\n%s", m.name, m.input.View())
	if m.statusMsg != "" {
		body += "\n\n" + m.statusMsg
	}
	body += "\n\nenter · move   esc · cancel"
	dialog := shared.DialogStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, shared.DialogTitleStyle.Render("Move Secret"), body),
	)
	return lipgloss.Place(shared.WindowSize.Width, shared.WindowSize.Height, lipgloss.Center, lipgloss.Center, dialog)
}
