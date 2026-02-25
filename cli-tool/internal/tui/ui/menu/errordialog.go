package menu

import (
	"encoding/json"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bkenks/bs3-cli/internal/tui/ui/events"
	"github.com/bkenks/bs3-cli/internal/tui/ui/shared"
)

// ErrorDialog displays an API error as pretty JSON and lets the user
// press any key to return to the main menu.
type ErrorDialog struct {
	body string
}

// NewErrorDialog accepts a raw error string (e.g. "error opening vault: {...}"),
// extracts the JSON portion if present, and pretty-prints it.
func NewErrorDialog(raw string) *ErrorDialog {
	return &ErrorDialog{body: extractPrettyJSON(raw)}
}

func (m *ErrorDialog) Init() tea.Cmd { return nil }

func (m *ErrorDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "ctrl+c":
			return m, tea.Quit
		default:
			return m, events.CmdSetState(shared.StateMainMenu)
		}
	}
	return m, nil
}

func (m *ErrorDialog) View() string {
	body := m.body + "\n\nany key · back to menu"
	dialog := shared.DialogStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, shared.DialogTitleStyle.Render("Error"), body),
	)
	return lipgloss.Place(shared.WindowSize.Width, shared.WindowSize.Height, lipgloss.Center, lipgloss.Center, dialog)
}

// extractPrettyJSON finds the first '{' in raw, attempts to parse everything
// from there as JSON, and returns a pretty-printed string. Falls back to the
// raw string if no valid JSON is found.
func extractPrettyJSON(raw string) string {
	start := strings.Index(raw, "{")
	if start >= 0 {
		var obj interface{}
		if err := json.Unmarshal([]byte(raw[start:]), &obj); err == nil {
			if pretty, err := json.MarshalIndent(obj, "", "  "); err == nil {
				return string(pretty)
			}
		}
	}
	return raw
}
