package tokens

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bkenks/bs3/internal/apiclient"
	"github.com/bkenks/bs3/internal/constants"
	"github.com/bkenks/bs3/internal/tui/ui/events"
	"github.com/bkenks/bs3/internal/tui/ui/shared"
)

type DeleteDialog struct {
	name      string
	statusMsg string
	client    apiclient.Client
}

func NewDeleteDialog(name string, client apiclient.Client) *DeleteDialog {
	return &DeleteDialog{name: name, client: client}
}

func (m *DeleteDialog) Init() tea.Cmd           { return nil }
func (m *DeleteDialog) SetStatusMsg(msg string) { m.statusMsg = msg }

func (m *DeleteDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	switch {
	case key.Matches(keyMsg, constants.ConfirmKeyMap.Proceed):
		return m, m.deleteCmd()
	case key.Matches(keyMsg, constants.ConfirmKeyMap.Exit):
		return m, events.CmdSetState(shared.StateTokensList)
	}
	return m, nil
}

func (m *DeleteDialog) deleteCmd() tea.Cmd {
	return func() tea.Msg {
		if err := m.client.DeleteToken(m.name); err != nil {
			return events.APIError{Err: err}
		}
		return events.TokenDeleted{}
	}
}

func (m *DeleteDialog) View() tea.View {
	body := fmt.Sprintf("Delete token %q?", m.name)
	if m.statusMsg != "" {
		body += "\n\n" + m.statusMsg
	}
	body += "\n\nctrl+p · confirm   esc · cancel"
	dialog := shared.DialogStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, shared.DialogTitleStyle.Render("Delete Token"), body),
	)
	return tea.NewView(lipgloss.Place(shared.WindowSize.Width, shared.WindowSize.Height, lipgloss.Center, lipgloss.Center, dialog))
}
