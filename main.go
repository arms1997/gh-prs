package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "gh-prs:", err)
		os.Exit(1)
	}
	if _, err := tea.NewProgram(newModel(cfg), tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "gh-prs:", err)
		os.Exit(1)
	}
}
