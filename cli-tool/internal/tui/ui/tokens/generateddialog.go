package tokens

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bkenks/bs3/internal/tui/ui/events"
	"github.com/bkenks/bs3/internal/tui/ui/shared"
)

type GeneratedDialog struct {
	name      string
	token     string
	expiresIn int64
}

func NewGeneratedDialog(name, token string, expiresIn int64) *GeneratedDialog {
	return &GeneratedDialog{name: name, token: token, expiresIn: expiresIn}
}

func (m *GeneratedDialog) Init() tea.Cmd { return nil }

func (m *GeneratedDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "esc" {
		return m, events.CmdSetState(shared.StateTokensList)
	}
	return m, nil
}

func (m *GeneratedDialog) View() string {
	exp := "never expires"
	if m.expiresIn > 0 {
		exp = fmt.Sprintf("expires in %ds", m.expiresIn)
	}
	body := fmt.Sprintf("Name:  %s\nToken: %s\n%s\n\nCopy this token now — it will not be shown again.\n\nesc · close",
		m.name, m.token, exp)
	dialog := shared.DialogStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, shared.DialogTitleStyle.Render("Token Generated"), body),
	)
	return lipgloss.Place(shared.WindowSize.Width, shared.WindowSize.Height, lipgloss.Center, lipgloss.Center, dialog)
}
