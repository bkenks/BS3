package secrets

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bkenks/bs3-cli/internal/apiclient"
	"github.com/bkenks/bs3-cli/internal/tui/ui/events"
	"github.com/bkenks/bs3-cli/internal/tui/ui/shared"
)

type ViewDialog struct {
	name   string
	folder string
	value  string
	client apiclient.Client
}

func NewViewDialog(name, folder string, client apiclient.Client) *ViewDialog {
	return &ViewDialog{name: name, folder: folder, client: client}
}

func (m *ViewDialog) Init() tea.Cmd {
	return func() tea.Msg {
		result, err := m.client.GetSecret(m.name, m.folder)
		if err != nil {
			return events.APIError{Err: err}
		}
		return events.SecretFetched{Value: result["secret"]}
	}
}

func (m *ViewDialog) SetValue(v string)       { m.value = v }
func (m *ViewDialog) SetStatusMsg(msg string) { m.value = "error: " + msg }

func (m *ViewDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok && keyMsg.String() == "esc" {
		return m, events.CmdSetState(shared.StateSecretsList)
	}
	return m, nil
}

func (m *ViewDialog) View() tea.View {
	value := m.value
	if value == "" {
		value = "loading..."
	}
	body := fmt.Sprintf("Name:  %s\nValue: %s\n\nesc · close", m.name, value)
	dialog := shared.DialogStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, shared.DialogTitleStyle.Render("View Secret"), body),
	)
	return tea.NewView(lipgloss.Place(shared.WindowSize.Width, shared.WindowSize.Height, lipgloss.Center, lipgloss.Center, dialog))
}
