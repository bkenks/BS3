package secrets

import (
	"errors"
	"fmt"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bkenks/bs3/internal/apiclient"
	"github.com/bkenks/bs3/internal/tui/ui/events"
	"github.com/bkenks/bs3/internal/tui/ui/shared"
)

// CreateFolderDialog prompts for a folder name and creates an empty folder.
type CreateFolderDialog struct {
	input     textinput.Model
	statusMsg string
	client    apiclient.Client
}

func NewCreateFolderDialog(client apiclient.Client) *CreateFolderDialog {
	folder := textinput.New()
	folder.Placeholder = "folder"
	folder.CharLimit = 128

	return &CreateFolderDialog{input: folder, client: client}
}

func (m *CreateFolderDialog) Init() tea.Cmd { return m.input.Focus() }

func (m *CreateFolderDialog) SetStatusMsg(msg string) { m.statusMsg = msg }

func (m *CreateFolderDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		name := m.input.Value()
		if name == "" {
			m.statusMsg = "folder name is required"
			return m, nil
		}
		return m, m.createCmd(name)
	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
}

func (m *CreateFolderDialog) createCmd(name string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.CreateFolder(name); err != nil {
			if errors.Is(err, apiclient.ErrFolderExists) {
				return events.APIError{Err: fmt.Errorf("a folder named %q already exists", name)}
			}
			return events.APIError{Err: err}
		}
		return events.FolderCreated{}
	}
}

func (m *CreateFolderDialog) View() tea.View {
	body := fmt.Sprintf("Folder name\n%s", m.input.View())
	if m.statusMsg != "" {
		body += "\n\n" + m.statusMsg
	}
	body += "\n\nenter · create   esc · cancel"
	dialog := shared.DialogStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, shared.DialogTitleStyle.Render("New Folder"), body),
	)
	return tea.NewView(lipgloss.Place(shared.WindowSize.Width, shared.WindowSize.Height, lipgloss.Center, lipgloss.Center, dialog))
}
