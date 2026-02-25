package bs3logger

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/lipgloss"
	charm "github.com/charmbracelet/log"
)

// Global logger instance
var Logger *charm.Logger
var ErrBars = lipgloss.NewStyle().Foreground(lipgloss.Color("#ED6E88"))
var Logo_Purple = lipgloss.AdaptiveColor{
	Light: "#614f97ff",
	Dark:  "#846ccc",
}
var Logo_LightBlue = lipgloss.AdaptiveColor{
	Light: "#8fb2cb",
	Dark:  "#89c5f0",
}
var BS3_BG = lipgloss.Color("#05085c")

var logoBox = lipgloss.NewStyle().
	Padding(0, 10, 1, 11).
	BorderForeground(Logo_Purple).
	Border(lipgloss.ThickBorder(), true, false)
var logsBox = lipgloss.NewStyle().
	Bold(true).
	Padding(0, 10).
	BorderForeground(Logo_Purple).
	Border(lipgloss.ThickBorder(), true, false)

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

	fmt.Println()
	fmt.Println(bar)
	logFunc(msg+"\n", keyvals...)
	fmt.Println(bar)
	fmt.Println()
}

func LogAddInfo(logFunc func(interface{}, ...interface{}), msg string, keyvals ...interface{}) {
	if len(keyvals) > 0 {
		if s, ok := keyvals[0].(string); ok {
			keyvals[0] = "\u21B3 " + s
		}
	}

	logFunc(msg+"\n", keyvals...)
}

func PrintBS3() {
	b := lipgloss.NewStyle().Foreground(Logo_Purple).Render(B)
	s := lipgloss.NewStyle().Foreground(Logo_LightBlue).Render(S)
	three := lipgloss.NewStyle().Foreground(Logo_LightBlue).Render(Three)

	bs3 := lipgloss.JoinHorizontal(lipgloss.Left, b, s, three)

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		bs3,
	)

	boxedContent := logoBox.Render(content)

	fmt.Print(boxedContent)
	fmt.Println()
}

func PrintLogSeperator() {
	logsText := lipgloss.NewStyle().Foreground(Logo_Purple).Render("Logs")
	fmt.Print(logsBox.Render(logsText))
	fmt.Println()
}
