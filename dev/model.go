package main

// model.go contains the Bubble Tea model for the BS3 dev hub. The model is a
// simple state machine with four states:
//
//	stateMenu    → the main list of actions (always the starting point)
//	stateInput   → a text prompt for actions that require user input
//	stateRunning → a spinner shown while a background goroutine is running
//	stateDone    → a success/error message; any key returns to stateMenu

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ─── States ───────────────────────────────────────────────────────────────────

type appState int

const (
	stateMenu    appState = iota // main action list
	stateInput                   // text input prompt
	stateRunning                 // background goroutine + spinner
	stateDone                    // result message
)

// ─── Messages ─────────────────────────────────────────────────────────────────

// funcDoneMsg is sent by a background goroutine when it finishes.
type funcDoneMsg struct{ err error }

// execDoneMsg is sent when a tea.ExecProcess command exits.
type execDoneMsg struct{ err error }

// ─── List Item ────────────────────────────────────────────────────────────────

// menuItem wraps an *action so it satisfies the bubbles list.Item interface.
type menuItem struct{ act *action }

func (m menuItem) Title() string       { return m.act.title }
func (m menuItem) Description() string { return m.act.description }
func (m menuItem) FilterValue() string { return m.act.title }

// ─── Styles ───────────────────────────────────────────────────────────────────

var (
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F87")).Bold(true)
	subtleStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
	boldStyle    = lipgloss.NewStyle().Bold(true)
)

// ─── Model ────────────────────────────────────────────────────────────────────

type model struct {
	repoRoot string

	state         appState
	pendingAction *action // action currently awaiting input or running

	list    list.Model
	input   textinput.Model
	spinner spinner.Model

	resultMsg    string // populated in stateDone
	resultIsErr  bool
	width, height int
}

// newModel constructs the initial model with all actions loaded into the list.
func newModel(repoRoot string) model {
	// ── List ──────────────────────────────────────────────────────────────────
	items := make([]list.Item, len(allActions))
	for i, a := range allActions {
		items[i] = menuItem{a}
	}

	delegate := list.NewDefaultDelegate()
	// Tighten vertical spacing between items.
	delegate.SetSpacing(0)

	l := list.New(items, delegate, 0, 0)
	l.Title = "BS3 Dev Hub"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	// Disable the default quit binding so we can handle q ourselves.
	l.KeyMap.Quit.Unbind()

	// ── Text Input ────────────────────────────────────────────────────────────
	ti := textinput.New()
	ti.CharLimit = 200

	// ── Spinner ───────────────────────────────────────────────────────────────
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))

	return model{
		repoRoot: repoRoot,
		state:    stateMenu,
		list:     l,
		input:    ti,
		spinner:  sp,
	}
}

// ─── Bubble Tea Interface ─────────────────────────────────────────────────────

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// ── Window resize ─────────────────────────────────────────────────────────
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.list.SetSize(msg.Width, msg.Height)
		return m, nil

	// ── Background goroutine finished ─────────────────────────────────────────
	case funcDoneMsg:
		if msg.err != nil {
			m.resultMsg = errorStyle.Render("✗  " + msg.err.Error())
			m.resultIsErr = true
		} else {
			m.resultMsg = successStyle.Render("✓  " + m.pendingAction.title + " completed successfully")
			m.resultIsErr = false
		}
		m.state = stateDone
		return m, nil

	// ── ExecProcess finished (error only matters if we want to surface it) ────
	case execDoneMsg:
		// The subprocess already printed its own output. Return to menu silently
		// unless it errored, in which case we surface the exit status.
		if msg.err != nil {
			m.resultMsg = errorStyle.Render(fmt.Sprintf("✗  process exited with error: %v", msg.err))
			m.resultIsErr = true
			m.state = stateDone
		}
		return m, nil

	// ── Spinner tick (only consumed while running) ────────────────────────────
	case spinner.TickMsg:
		if m.state == stateRunning {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	// ── Key events ────────────────────────────────────────────────────────────
	case tea.KeyMsg:
		switch m.state {

		case stateMenu:
			return m.updateMenu(msg)

		case stateInput:
			return m.updateInput(msg)

		case stateRunning:
			// Ignore all keys while a goroutine is running.
			return m, nil

		case stateDone:
			// Any key returns to the menu.
			m.state = stateMenu
			m.pendingAction = nil
			m.resultMsg = ""
			return m, nil
		}
	}

	// ── Propagate remaining messages to the active sub-component ─────────────
	switch m.state {
	case stateMenu:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	case stateInput:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	return m, nil
}

// updateMenu handles key events while in stateMenu.
func (m model) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "q":
		// Only quit when not filtering — 'q' is a valid filter character too.
		if m.list.FilterState() != list.Filtering {
			return m, tea.Quit
		}
	case "enter":
		if m.list.FilterState() == list.Filtering {
			break // let the list confirm the filter
		}
		selected, ok := m.list.SelectedItem().(menuItem)
		if !ok {
			break
		}
		return m.dispatchAction(selected.act)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// updateInput handles key events while in stateInput.
func (m model) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		// Cancel: discard input and return to menu.
		m.state = stateMenu
		m.pendingAction = nil
		return m, nil
	case "enter":
		inputVal := strings.TrimSpace(m.input.Value())
		if inputVal == "" {
			return m, nil
		}
		act := m.pendingAction
		m.pendingAction = nil
		m.state = stateMenu

		if act.makeInputCmd != nil {
			// Run the resulting command interactively.
			cmd := act.makeInputCmd(m.repoRoot, inputVal)
			return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
				return execDoneMsg{err: err}
			})
		}
		if act.runWithInput != nil {
			// Wrap the captured input value in a goroutine action.
			resolved := &action{
				title: act.title,
				run: func(repoRoot string) error {
					return act.runWithInput(repoRoot, inputVal)
				},
			}
			m.pendingAction = resolved
			m.state = stateRunning
			return m, tea.Batch(m.spinner.Tick, runFunc(m.repoRoot, resolved))
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// dispatchAction starts the appropriate execution path for the selected action.
func (m model) dispatchAction(act *action) (tea.Model, tea.Cmd) {
	switch {
	case act.makeCmd != nil:
		// Suspend TUI and hand the terminal to the command.
		return m, tea.ExecProcess(act.makeCmd(m.repoRoot), func(err error) tea.Msg {
			return execDoneMsg{err: err}
		})

	case act.run != nil:
		// Launch goroutine and show spinner.
		m.pendingAction = act
		m.state = stateRunning
		return m, tea.Batch(m.spinner.Tick, runFunc(m.repoRoot, act))

	case act.inputPrompt != "":
		// Collect user input before running.
		m.pendingAction = act
		m.input.Reset()
		m.input.Placeholder = ""
		m.input.Focus()
		m.state = stateInput
		return m, textinput.Blink
	}

	return m, nil
}

// runFunc returns a tea.Cmd that executes act.run in a goroutine and sends
// a funcDoneMsg when it completes.
func runFunc(repoRoot string, act *action) tea.Cmd {
	return func() tea.Msg {
		return funcDoneMsg{err: act.run(repoRoot)}
	}
}

// ─── View ─────────────────────────────────────────────────────────────────────

func (m model) View() string {
	switch m.state {

	case stateMenu:
		return m.list.View()

	case stateInput:
		content := fmt.Sprintf(
			"%s\n\n%s\n%s\n\n%s",
			boldStyle.Render(m.pendingAction.title),
			m.pendingAction.inputPrompt,
			m.input.View(),
			subtleStyle.Render("enter to confirm • esc to cancel"),
		)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)

	case stateRunning:
		content := fmt.Sprintf(
			"%s  %s",
			m.spinner.View(),
			boldStyle.Render(m.pendingAction.title),
		)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)

	case stateDone:
		content := fmt.Sprintf(
			"%s\n\n%s",
			m.resultMsg,
			subtleStyle.Render("press any key to return to menu"),
		)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}

	return ""
}
