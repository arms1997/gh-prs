package main

// gh-prs as a desktop app: the same config, cache, jobs and commands as the terminal UI.

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"gh-prs/core"
)

//go:embed all:frontend/dist
var assets embed.FS

// useShellPath gives the app your login shell's PATH. Apps opened from Finder get only the
// system PATH, which misses gh, git, claude and the rest in /opt/homebrew/bin and similar.
func useShellPath() {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/zsh"
	}
	out, err := exec.Command(shell, "-lic", `printf %s "$PATH"`).Output()
	if p := strings.TrimSpace(string(out)); err == nil && p != "" {
		os.Setenv("PATH", p)
	}
}

func main() {
	useShellPath()
	cfg, err := core.LoadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "gh-prs:", err)
		os.Exit(1)
	}
	app := NewApp(cfg)
	bg := &options.RGBA{R: 255, G: 255, B: 255, A: 1} // shown until the page loads; matches style.css --bg
	if out, _ := exec.Command("defaults", "read", "-g", "AppleInterfaceStyle").Output(); strings.TrimSpace(string(out)) == "Dark" {
		bg = &options.RGBA{R: 22, G: 24, B: 29, A: 1}
	}
	err = wails.Run(&options.App{
		Title:            "gh-prs",
		Width:            1440,
		Height:           900,
		MinWidth:         900,
		MinHeight:        500,
		AssetServer:      &assetserver.Options{Assets: assets},
		BackgroundColour: bg,
		OnStartup:        app.startup,
		Bind:             []interface{}{app},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "gh-prs:", err)
		os.Exit(1)
	}
}
