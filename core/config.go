package core

// Config: the user file layered over the embedded defaults/, and prompt file lookup.

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"gh-prs/defaults"
)

type Section struct {
	Title   string   `yaml:"title"`
	Query   string   `yaml:"query"`
	GroupBy []string `yaml:"group_by"`
}

type Config struct {
	ProjectLabelPrefix string         `yaml:"project_label_prefix"`
	RefreshInterval    string         `yaml:"refresh_interval"`
	ReposDir           string         `yaml:"repos_dir"`
	Opener             string         `yaml:"opener"`
	PromptContextFile  string         `yaml:"prompt_context_file"`
	LinearWorkspace    string         `yaml:"linear_workspace"`
	Sections           []Section      `yaml:"sections"`
	Commands           []Command      `yaml:"commands"`
	Columns            map[string]int `yaml:"columns,omitempty"` // desktop list widths in pixels, by ColumnNames

	Refresh       time.Duration `yaml:"-"` // parsed RefreshInterval; 0 disables auto-refresh
	promptContext string        // contents of PromptContextFile, prepended to every prompt
}

// Defaults holds the built-in defaults/ directory (config.yml, commands/, prompts/).
var Defaults fs.FS = defaults.FS

// defaultConfig assembles the built-in defaults: defaults/config.yml plus every file in
// defaults/commands/, in file-name order.
func defaultConfig() (Config, error) {
	var cfg Config
	if err := decodeStrict(mustRead("config.yml"), &cfg); err != nil {
		return cfg, fmt.Errorf("defaults/config.yml: %w", err)
	}
	files, _ := fs.Glob(Defaults, "commands/*.yml")
	sort.Strings(files)
	for _, f := range files {
		var cmds []Command
		if err := decodeStrict(mustRead(f), &cmds); err != nil {
			return cfg, fmt.Errorf("defaults/%s: %w", f, err)
		}
		cfg.Commands = append(cfg.Commands, cmds...)
	}
	return cfg, nil
}

func mustRead(name string) []byte {
	b, err := fs.ReadFile(Defaults, name)
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
	if b, err := os.ReadFile(filepath.Join(filepath.Dir(ConfigPath()), "prompts", name)); err == nil {
		return string(b), nil
	}
	if b, err := fs.ReadFile(Defaults, path.Join("prompts", name)); err == nil {
		return string(b), nil
	}
	return "", fmt.Errorf("prompt %q not found in %s or the built-in prompts", name, filepath.Join(filepath.Dir(ConfigPath()), "prompts"))
}

func ConfigPath() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "gh-prs", "config.yml")
}

// loadConfig reads the config, writing the default on first run so it is easy to find and edit.
func LoadConfig() (Config, error) {
	path := ConfigPath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		data = mustRead("config.yml")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return Config{}, err
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return Config{}, err
		}
	} else if err != nil {
		return Config{}, err
	}
	cfg, err := ParseConfig(data)
	if err != nil {
		return cfg, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

// parseConfig layers the user's file over the defaults: keys they set replace the default wholesale.
func ParseConfig(data []byte) (Config, error) {
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
	cfg.Refresh = 2 * time.Minute
	if cfg.RefreshInterval != "" {
		d, err := time.ParseDuration(cfg.RefreshInterval)
		if cfg.RefreshInterval == "0" {
			d, err = 0, nil
		}
		if err != nil || d < 0 || (d > 0 && d < 30*time.Second) {
			return cfg, fmt.Errorf("refresh_interval %q: use a duration of at least 30s, or 0 to disable", cfg.RefreshInterval)
		}
		cfg.Refresh = d
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
				return cfg, fmt.Errorf("section %q: unknown group_by %q (use project, status, repo, turn)", s.Title, g)
			}
		}
	}
	if err := validateCommands(&cfg); err != nil {
		return cfg, err
	}
	if err := validateColumns(cfg.Columns); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// ColumnNames are the desktop list's resizable columns; the title takes the rest of the width.
var ColumnNames = []string{"repo", "diff", "attention", "next"}

func validateColumns(cols map[string]int) error {
	for k, w := range cols {
		if !slices.Contains(ColumnNames, k) {
			return fmt.Errorf("columns: unknown column %q (use %s)", k, strings.Join(ColumnNames, ", "))
		}
		if w < 40 || w > 800 {
			return fmt.Errorf("columns.%s: %d isn't between 40 and 800", k, w)
		}
	}
	return nil
}

// SaveColumns writes the widths to config.yml. Only the top-level columns key changes, so the rest
// of the file, comments included, stays as you wrote it. No widths removes the key.
func SaveColumns(cols map[string]int) error {
	if err := validateColumns(cols); err != nil {
		return err
	}
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		return err
	}
	out := setColumns(data, cols)
	if _, err := ParseConfig(out); err != nil {
		return err
	}
	return os.WriteFile(ConfigPath(), out, 0o644)
}

func setColumns(data []byte, cols map[string]int) []byte {
	line := ""
	if len(cols) > 0 {
		var parts []string
		for _, k := range ColumnNames {
			if w, ok := cols[k]; ok {
				parts = append(parts, fmt.Sprintf("%s: %d", k, w))
			}
		}
		line = "columns: {" + strings.Join(parts, ", ") + "}"
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	var out []string
	for i := 0; i < len(lines); i++ {
		if !strings.HasPrefix(lines[i], "columns:") {
			out = append(out, lines[i])
			continue
		}
		for i+1 < len(lines) && (strings.HasPrefix(lines[i+1], " ") || strings.HasPrefix(lines[i+1], "\t")) {
			i++ // the block-style entries under it
		}
		if line != "" {
			out, line = append(out, line), ""
		}
	}
	if line != "" {
		out = append(out, "", line)
	}
	return []byte(strings.TrimRight(strings.Join(out, "\n"), "\n") + "\n")
}
