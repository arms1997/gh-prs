package main

// Config: the user file layered over the embedded defaults/, and prompt file lookup.

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Section struct {
	Title   string   `yaml:"title"`
	Query   string   `yaml:"query"`
	GroupBy []string `yaml:"group_by"`
}

type Config struct {
	ProjectLabelPrefix string    `yaml:"project_label_prefix"`
	RefreshInterval    string    `yaml:"refresh_interval"`
	ReposDir           string    `yaml:"repos_dir"`
	Opener             string    `yaml:"opener"`
	PromptContextFile  string    `yaml:"prompt_context_file"`
	Sections           []Section `yaml:"sections"`
	Commands           []Command `yaml:"commands"`

	refresh       time.Duration // parsed RefreshInterval; 0 disables auto-refresh
	promptContext string        // contents of PromptContextFile, prepended to every prompt
}

//go:embed defaults
var defaultsFS embed.FS

// defaultConfig assembles the built-in defaults: defaults/config.yml plus every file in
// defaults/commands/, in file-name order.
func defaultConfig() (Config, error) {
	var cfg Config
	if err := decodeStrict(mustRead("defaults/config.yml"), &cfg); err != nil {
		return cfg, fmt.Errorf("defaults/config.yml: %w", err)
	}
	files, _ := fs.Glob(defaultsFS, "defaults/commands/*.yml")
	sort.Strings(files)
	for _, f := range files {
		var cmds []Command
		if err := decodeStrict(mustRead(f), &cmds); err != nil {
			return cfg, fmt.Errorf("%s: %w", f, err)
		}
		cfg.Commands = append(cfg.Commands, cmds...)
	}
	return cfg, nil
}

func mustRead(name string) []byte {
	b, err := defaultsFS.ReadFile(name)
	if err != nil {
		panic(err) // embedded at build time
	}
	return b
}

// decodeStrict decodes YAML into v, rejecting unknown keys so typos fail loudly.
func decodeStrict(data []byte, v any) error {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(v); err != nil && err != io.EOF {
		return err
	}
	return nil
}

// readPrompt finds a prompt file: absolute and ~/ paths as given, otherwise a name in
// ~/.config/gh-prs/prompts/ (so you can replace a built-in prompt) and then the built-ins.
func readPrompt(name string) (string, error) {
	if filepath.IsAbs(name) || strings.HasPrefix(name, "~/") {
		b, err := os.ReadFile(expandHome(name))
		return string(b), err
	}
	if b, err := os.ReadFile(filepath.Join(filepath.Dir(configPath()), "prompts", name)); err == nil {
		return string(b), nil
	}
	if b, err := defaultsFS.ReadFile(path.Join("defaults/prompts", name)); err == nil {
		return string(b), nil
	}
	return "", fmt.Errorf("prompt %q not found in %s or the built-in prompts", name, filepath.Join(filepath.Dir(configPath()), "prompts"))
}

func configPath() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "gh-prs", "config.yml")
}

// loadConfig reads the config, writing the default on first run so it is easy to find and edit.
func loadConfig() (Config, error) {
	path := configPath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		data = mustRead("defaults/config.yml")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return Config{}, err
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return Config{}, err
		}
	} else if err != nil {
		return Config{}, err
	}
	cfg, err := parseConfig(data)
	if err != nil {
		return cfg, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

// parseConfig layers the user's file over the defaults: keys they set replace the default wholesale.
func parseConfig(data []byte) (Config, error) {
	cfg, err := defaultConfig()
	if err != nil {
		return cfg, err
	}
	if err := decodeStrict(data, &cfg); err != nil {
		return cfg, err
	}
	if cfg.PromptContextFile != "" {
		if cfg.promptContext, err = readPrompt(cfg.PromptContextFile); err != nil {
			return cfg, err
		}
	}
	cfg.refresh = 2 * time.Minute
	if cfg.RefreshInterval != "" {
		d, err := time.ParseDuration(cfg.RefreshInterval)
		if cfg.RefreshInterval == "0" {
			d, err = 0, nil
		}
		if err != nil || d < 0 || (d > 0 && d < 30*time.Second) {
			return cfg, fmt.Errorf("refresh_interval %q: use a duration of at least 30s, or 0 to disable", cfg.RefreshInterval)
		}
		cfg.refresh = d
	}
	if len(cfg.Sections) == 0 {
		return cfg, fmt.Errorf("no sections defined")
	}
	for _, s := range cfg.Sections {
		if s.Query == "" {
			return cfg, fmt.Errorf("section %q has no query", s.Title)
		}
		for _, g := range s.GroupBy {
			if _, ok := groupers[g]; !ok {
				return cfg, fmt.Errorf("section %q: unknown group_by %q (use project, status, repo)", s.Title, g)
			}
		}
	}
	if err := validateCommands(&cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}
