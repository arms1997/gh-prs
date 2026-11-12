package main

// Entry point: load the config and run the TUI.

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
	if len(os.Args) > 1 && os.Args[1] == "check" { // validates config, prompts and organize.yml without the TUI
		if _, err := loadOrganize(); err != nil {
			fmt.Fprintln(os.Stderr, "gh-prs:", err)
			os.Exit(1)
		}
		fmt.Printf("ok: %s (%d sections, %d commands)\n", configPath(), len(cfg.Sections), len(cfg.Commands))
		return
	}
	if _, err := tea.NewProgram(newModel(cfg), tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "gh-prs:", err)
		os.Exit(1)
	}
}
