package secrets

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bkenks/bs3-cli/internal/apiclient"
	"github.com/bkenks/bs3-cli/internal/tui/ui/events"
	"github.com/bkenks/bs3-cli/internal/tui/ui/shared"
)

type ViewDialog struct {
	name   string
	value  string
	client apiclient.Client
}

func NewViewDialog(name string, client apiclient.Client) *ViewDialog {
	return &ViewDialog{name: name, client: client}
}

func (m *ViewDialog) Init() tea.Cmd {
	return func() tea.Msg {
		result, err := m.client.GetSecret(m.name)
		if err != nil {
			return events.APIError{Err: err}
		}
		return events.SecretFetched{Value: result["secret"]}
	}
}

func (m *ViewDialog) SetValue(v string)       { m.value = v }
func (m *ViewDialog) SetStatusMsg(msg string) { m.value = "error: " + msg }

func (m *ViewDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "esc" {
		return m, events.CmdSetState(shared.StateSecretsList)
	}
	return m, nil
}

func (m *ViewDialog) View() string {
	value := m.value
	if value == "" {
		value = "loading..."
	}
	body := fmt.Sprintf("Name:  %s\nValue: %s\n\nesc · close", m.name, value)
	dialog := shared.DialogStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, shared.DialogTitleStyle.Render("View Secret"), body),
	)
	return lipgloss.Place(shared.WindowSize.Width, shared.WindowSize.Height, lipgloss.Center, lipgloss.Center, dialog)
}
