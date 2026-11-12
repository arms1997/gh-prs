package main

// Colors and badge styles.

import (
	"github.com/charmbracelet/lipgloss"

	"gh-prs/core"
)

var (
	plain        = lipgloss.NewStyle()
	dim          = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	bold         = lipgloss.NewStyle().Bold(true)
	green        = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	red          = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	yellow       = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	cyan         = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	sectionStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5"))
	selBg        = lipgloss.Color("238")
	stripeBg     = lipgloss.Color("235")

	// Solid badges mark the reasons a PR needs action.
	badgeFail     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("1"))
	badgeChanges  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("0")).Background(lipgloss.Color("208"))
	badgeConflict = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("5"))

	statusStyles = map[string]lipgloss.Style{
		core.NeedsAction.String():   bold.Foreground(lipgloss.Color("1")),
		core.WaitingReview.String(): bold.Foreground(lipgloss.Color("3")),
		core.ReadyToMerge.String():  bold.Foreground(lipgloss.Color("2")),
		core.Draft.String():         bold.Foreground(lipgloss.Color("8")),
	}
)
