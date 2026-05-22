package bs3logger

import (
	"fmt"
	"os"
	"time"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
	charm "github.com/charmbracelet/log"
)

// Global logger instance
var Logger *charm.Logger
var ErrBars = lipgloss.NewStyle().Foreground(lipgloss.Color("#ED6E88"))
var Logo_Purple = compat.AdaptiveColor{
	Light: lipgloss.Color("#614f97ff"),
	Dark:  lipgloss.Color("#846ccc"),
}
var Logo_LightBlue = compat.AdaptiveColor{
	Light: lipgloss.Color("#8fb2cb"),
	Dark:  lipgloss.Color("#89c5f0"),
}
var BS3_BG = lipgloss.Color("#05085c")

func init() {
	Logger = charm.NewWithOptions(os.Stderr, charm.Options{
		ReportTimestamp: true,
		TimeFormat:      time.Kitchen,
	})
}

func LogError(logFunc func(interface{}, ...interface{}), msg string, keyvals ...interface{}) {
	if len(keyvals) > 0 {
		if s, ok := keyvals[0].(string); ok {
			keyvals[0] = "\u21B3 " + s
		}
	}

	bar := ErrBars.Render("---")

	// Write to stderr so these lines interleave in order with the logger,
	// which also writes to stderr; mixing in stdout reorders under Docker.
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, bar)
	logFunc(msg+"\n", keyvals...)
	fmt.Fprintln(os.Stderr, bar)
	fmt.Fprintln(os.Stderr)
}

func LogAddInfo(logFunc func(interface{}, ...interface{}), msg string, keyvals ...interface{}) {
	if len(keyvals) > 0 {
		if s, ok := keyvals[0].(string); ok {
			keyvals[0] = "\u21B3 " + s
		}
	}

	logFunc(msg+"\n", keyvals...)
}

// titleBox styles "BS3" like a Bubble Tea list header: a solid pill with
// padded, bold text on the brand purple background.
var titleBox = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("#f5f3ff")).
	Background(Logo_Purple).
	Padding(0, 1)

// PrintBS3 and PrintLogSeperator write to os.Stderr — the same stream the
// logger uses — so the pills appear in order with the log output. Writing to
// stdout instead causes Docker to flush them after all stderr log lines.

func PrintBS3() {
	fmt.Fprintln(os.Stderr, titleBox.Render("Welcome to BS3."))
}

func PrintLogSeperator() {
	fmt.Fprintln(os.Stderr, titleBox.Render("Logs"))
}
