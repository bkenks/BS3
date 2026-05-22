// Package main is the BS3 developer hub — a Bubble Tea TUI that provides a
// central menu for common development tasks: building, running, testing, and
// publishing the server, cli-tool, and logger modules.
//
// Run from the repository root:
//
//	go run ./dev/
package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
)

func main() {
	repoRoot, err := findRepoRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	p := tea.NewProgram(newModel(repoRoot))
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error running dev hub:", err)
		os.Exit(1)
	}
}

// findRepoRoot returns the current working directory after verifying it
// contains the expected BS3 module subdirectories. This ensures the dev hub
// is always run from the correct location.
func findRepoRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	for _, sub := range []string{"server", "cli-tool", "logger"} {
		if _, err := os.Stat(filepath.Join(cwd, sub)); err != nil {
			return "", fmt.Errorf("must be run from the BS3 repository root (missing %s/)", sub)
		}
	}

	return cwd, nil
}
