package tokens

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bkenks/bs3-cli/internal/apiclient"
	"github.com/bkenks/bs3-cli/internal/tui/ui/events"
	"github.com/bkenks/bs3-cli/internal/tui/ui/shared"
)

type GenerateDialog struct {
	inputs    []textinput.Model
	focusIdx  int
	statusMsg string
	client    apiclient.Client
}

func NewGenerateDialog(client apiclient.Client) *GenerateDialog {
	name := textinput.New()
	name.Placeholder = "name"
	name.CharLimit = 128

	ttl := textinput.New()
	ttl.Placeholder = "TTL in seconds (empty = no expiry)"
	ttl.CharLimit = 20

	return &GenerateDialog{inputs: []textinput.Model{name, ttl}, client: client}
}

func (m *GenerateDialog) Init() tea.Cmd           { return m.inputs[0].Focus() }
func (m *GenerateDialog) SetStatusMsg(msg string) { m.statusMsg = msg }

func (m *GenerateDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		return m, events.CmdSetState(shared.StateTokensList)
	case "tab":
		m.focusIdx = (m.focusIdx + 1) % len(m.inputs)
		return m, m.cycleFocus()
	case "enter":
		if m.focusIdx < len(m.inputs)-1 {
			m.focusIdx++
			return m, m.cycleFocus()
		}
		name := m.inputs[0].Value()
		if name == "" {
			m.statusMsg = "name is required"
			return m, nil
		}
		var ttl int64
		if raw := m.inputs[1].Value(); raw != "" {
			parsed, err := strconv.ParseInt(raw, 10, 64)
			if err != nil || parsed < 0 {
				m.statusMsg = "TTL must be a non-negative integer"
				return m, nil
			}
			ttl = parsed
		}
		return m, m.generateCmd(name, ttl)
	default:
		var cmd tea.Cmd
		m.inputs[m.focusIdx], cmd = m.inputs[m.focusIdx].Update(msg)
		return m, cmd
	}
}

func (m *GenerateDialog) cycleFocus() tea.Cmd {
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

func (m *GenerateDialog) generateCmd(name string, ttl int64) tea.Cmd {
	return func() tea.Msg {
		result, err := m.client.GenerateToken(name, ttl)
		if err != nil {
			return events.APIError{Err: err}
		}
		return events.TokenGenerated{Name: result.Name, Token: result.Token, ExpiresIn: result.ExpiresIn}
	}
}

func (m *GenerateDialog) View() string {
	body := fmt.Sprintf("Name\n%s\n\nTTL (seconds)\n%s", m.inputs[0].View(), m.inputs[1].View())
	if m.statusMsg != "" {
		body += "\n\n" + m.statusMsg
	}
	body += "\n\nenter · next/submit   tab · cycle   esc · cancel"
	dialog := shared.DialogStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, shared.DialogTitleStyle.Render("Generate Token"), body),
	)
	return lipgloss.Place(shared.WindowSize.Width, shared.WindowSize.Height, lipgloss.Center, lipgloss.Center, dialog)
}
