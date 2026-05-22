package main

// model.go contains the Bubble Tea model for the BS3 dev hub. The model is a
// simple state machine with the following states:
//
//	stateMenu        → the main list of actions (always the starting point)
//	stateInput       → a text prompt for actions that require user input
//	stateReleaseForm → a small form collecting a release version + channel
//	stateRunning     → a spinner shown while a background goroutine is running
//	stateOutput      → captured command output shown in a scrollable modal
//	stateDone        → a one-line success/error message; any key returns to menu

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ─── States ───────────────────────────────────────────────────────────────────

type appState int

const (
	stateMenu        appState = iota // main action list
	stateInput                       // text input prompt
	stateReleaseForm                 // release version + channel form
	stateRunning                     // background goroutine + spinner
	stateOutput                      // captured output, scrollable modal
	stateDone                        // one-line result message
)

// ─── Messages ─────────────────────────────────────────────────────────────────

// funcDoneMsg is sent by a background goroutine when it finishes. It carries
// the combined stdout+stderr of the command and any error.
type funcDoneMsg struct {
	output string
	err    error
}

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
	accentStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED")).Bold(true)

	// modalBorderStyle frames the captured-output modal as a rounded popup.
	modalBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#7C3AED")).
				Padding(0, 1)
	// modalHeaderStyle renders the command name as a solid pill in the modal
	// header so it is clear which menu option produced the output.
	modalHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#F5F3FF")).
				Background(lipgloss.Color("#7C3AED")).
				Padding(0, 1)
)

// ─── Model ────────────────────────────────────────────────────────────────────

type model struct {
	repoRoot string

	state         appState
	pendingAction *action // action currently awaiting input or running

	list     list.Model
	input    textinput.Model
	spinner  spinner.Model
	viewport viewport.Model

	// Release form fields.
	releaseInput  textinput.Model
	releaseStable bool

	resultMsg     string // populated in stateDone
	resultIsErr   bool
	outputTitle   string // header title for stateOutput
	outputIsErr   bool   // whether the captured action failed
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

	// ── Release Version Input ─────────────────────────────────────────────────
	ri := textinput.New()
	ri.CharLimit = 64
	ri.Placeholder = "1.2.3"

	// ── Spinner ───────────────────────────────────────────────────────────────
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))

	return model{
		repoRoot:      repoRoot,
		state:         stateMenu,
		list:          l,
		input:         ti,
		spinner:       sp,
		viewport:      viewport.New(),
		releaseInput:  ri,
		releaseStable: true,
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
		vw, vh := m.outputViewportSize()
		m.viewport.SetWidth(vw)
		m.viewport.SetHeight(vh)
		return m, nil

	// ── Background goroutine finished ─────────────────────────────────────────
	case funcDoneMsg:
		m.outputTitle = m.pendingAction.title
		m.outputIsErr = msg.err != nil

		body := msg.output
		if msg.err != nil {
			// Surface the error in the body so it is visible even when the
			// command produced no output of its own.
			if strings.TrimSpace(body) == "" {
				body = msg.err.Error()
			} else {
				body = body + "\n\n" + msg.err.Error()
			}
		}
		if strings.TrimSpace(body) == "" {
			body = "(no output)"
		}

		vw, vh := m.outputViewportSize()
		m.viewport.SetWidth(vw)
		m.viewport.SetHeight(vh)
		m.viewport.SetContent(body)
		m.viewport.GotoTop()
		m.state = stateOutput
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
	case tea.KeyPressMsg:
		switch m.state {

		case stateMenu:
			return m.updateMenu(msg)

		case stateInput:
			return m.updateInput(msg)

		case stateReleaseForm:
			return m.updateReleaseForm(msg)

		case stateRunning:
			// Ignore all keys while a goroutine is running.
			return m, nil

		case stateOutput:
			return m.updateOutput(msg)

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
	case stateReleaseForm:
		var cmd tea.Cmd
		m.releaseInput, cmd = m.releaseInput.Update(msg)
		return m, cmd
	case stateOutput:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	return m, nil
}

// updateMenu handles key events while in stateMenu.
func (m model) updateMenu(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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
func (m model) updateInput(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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
				run: func(repoRoot string) (string, error) {
					return "", act.runWithInput(repoRoot, inputVal)
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

// updateReleaseForm handles key events while in stateReleaseForm.
func (m model) updateReleaseForm(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		// Cancel: discard the form and return to menu.
		m.state = stateMenu
		m.pendingAction = nil
		return m, nil
	case "tab", "space", "left", "right":
		// Toggle the Stable / Pre-release channel.
		m.releaseStable = !m.releaseStable
		return m, nil
	case "enter":
		version := strings.TrimSpace(m.releaseInput.Value())
		if version == "" {
			// Require a non-empty version; the script validates the format.
			return m, nil
		}
		act := m.pendingAction
		stable := m.releaseStable
		script := act.releaseScript

		// Run the release script through the same captured-output path as
		// regular run actions so its output lands in the modal.
		resolved := &action{
			title: act.title,
			run: func(repoRoot string) (string, error) {
				return runReleaseScript(repoRoot, script, version, stable)
			},
		}
		m.pendingAction = resolved
		m.state = stateRunning
		return m, tea.Batch(m.spinner.Tick, runFunc(m.repoRoot, resolved))
	}

	var cmd tea.Cmd
	m.releaseInput, cmd = m.releaseInput.Update(msg)
	return m, cmd
}

// updateOutput handles key events while in stateOutput.
func (m model) updateOutput(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc", "q":
		// Close the modal and return to the menu.
		m.state = stateMenu
		m.pendingAction = nil
		return m, nil
	}

	// Let the viewport handle arrows / pgup / pgdn for scrolling.
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
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

	case act.releaseScript != "":
		// Collect a release version + channel before running the script.
		m.pendingAction = act
		m.releaseInput.Reset()
		m.releaseInput.Focus()
		m.releaseStable = true
		m.state = stateReleaseForm
		return m, textinput.Blink

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
// a funcDoneMsg carrying the captured output and any error.
func runFunc(repoRoot string, act *action) tea.Cmd {
	return func() tea.Msg {
		out, err := act.run(repoRoot)
		return funcDoneMsg{output: out, err: err}
	}
}

// outputModalDims returns the total width/height (including border) of the
// captured-output modal popup, leaving a margin so it floats over the screen.
func (m model) outputModalDims() (int, int) {
	w := m.width - 8
	if w < 24 {
		w = 24
	}
	h := m.height - 4
	if h < 8 {
		h = 8
	}
	return w, h
}

// outputViewportSize returns the width/height for the stateOutput viewport,
// i.e. the modal's interior minus its border, padding, header, and footer.
func (m model) outputViewportSize() (int, int) {
	w, h := m.outputModalDims()
	// Subtract 2 border cols + 2 padding cols.
	vw := w - 4
	// Subtract 2 border rows + header (1) + footer (1) + 2 blank spacer rows.
	vh := h - 6
	if vw < 1 {
		vw = 1
	}
	if vh < 1 {
		vh = 1
	}
	return vw, vh
}

// ─── View ─────────────────────────────────────────────────────────────────────

func (m model) View() tea.View {
	var content string
	switch m.state {

	case stateMenu:
		content = m.list.View()

	case stateInput:
		body := fmt.Sprintf(
			"%s\n\n%s\n%s\n\n%s",
			boldStyle.Render(m.pendingAction.title),
			m.pendingAction.inputPrompt,
			m.input.View(),
			subtleStyle.Render("enter to confirm • esc to cancel"),
		)
		content = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, body)

	case stateReleaseForm:
		content = m.viewReleaseForm()

	case stateRunning:
		body := fmt.Sprintf(
			"%s  %s",
			m.spinner.View(),
			boldStyle.Render(m.pendingAction.title),
		)
		content = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, body)

	case stateOutput:
		content = m.viewOutput()

	case stateDone:
		body := fmt.Sprintf(
			"%s\n\n%s",
			m.resultMsg,
			subtleStyle.Render("press any key to return to menu"),
		)
		content = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, body)
	}

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

// viewReleaseForm renders the release version + channel form.
func (m model) viewReleaseForm() string {
	stableLabel := "Stable"
	prereleaseLabel := "Pre-release"
	if m.releaseStable {
		stableLabel = accentStyle.Render("[ Stable ]")
		prereleaseLabel = subtleStyle.Render("  Pre-release  ")
	} else {
		stableLabel = subtleStyle.Render("  Stable  ")
		prereleaseLabel = accentStyle.Render("[ Pre-release ]")
	}

	content := fmt.Sprintf(
		"%s\n\n%s\n%s\n\n%s\n%s  %s\n\n%s",
		boldStyle.Render(m.pendingAction.title),
		"Version (bare semver, e.g. 1.2.3):",
		m.releaseInput.View(),
		"Channel:",
		stableLabel,
		prereleaseLabel,
		subtleStyle.Render("tab/space to toggle channel • enter to release • esc to cancel"),
	)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

// viewOutput renders the captured-output modal: a rounded bordered popup with
// a header pill naming the menu option, the scrollable viewport, and a footer
// hint. The popup is centered on screen so it clearly reads as a modal.
func (m model) viewOutput() string {
	boxW, boxH := m.outputModalDims()

	// Header: command-name pill + success/failure status.
	titlePill := modalHeaderStyle.Render(m.outputTitle)
	var status string
	if m.outputIsErr {
		status = errorStyle.Render("✗ failed")
	} else {
		status = successStyle.Render("✓ success")
	}
	header := lipgloss.JoinHorizontal(lipgloss.Center, titlePill, "  ", status)

	footer := subtleStyle.Render("↑/↓ pgup/pgdn to scroll • esc or q to close")

	inner := fmt.Sprintf("%s\n\n%s\n\n%s", header, m.viewport.View(), footer)

	// Width/Height set the interior; the border adds 2 cols/rows around it.
	modal := modalBorderStyle.
		Width(boxW - 2).
		Height(boxH - 2).
		Render(inner)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}
