package core

import (
	"os"
	"strings"
	"testing"
)

func TestSaveColumns(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if _, err := LoadConfig(); err != nil { // writes the default config
		t.Fatal(err)
	}
	before, _ := os.ReadFile(ConfigPath())
	before = append(before, "# mine\n"...)
	_ = os.WriteFile(ConfigPath(), before, 0o644)

	if err := SaveColumns(map[string]int{"next": 200, "repo": 150}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(ConfigPath())
	if want := string(before) + "\ncolumns: {repo: 150, next: 200}\n"; string(got) != want {
		t.Fatalf("appended wrong:\n%s", got)
	}
	cfg, err := LoadConfig()
	if err != nil || cfg.Columns["repo"] != 150 || cfg.Columns["next"] != 200 {
		t.Fatalf("columns %v, %v", cfg.Columns, err)
	}

	if err := SaveColumns(nil); err != nil {
		t.Fatal(err)
	}
	if got, _ = os.ReadFile(ConfigPath()); string(got) != string(before) {
		t.Fatalf("removing left:\n%s", got)
	}

	// A block-style key you wrote is replaced in place; what follows it stays.
	block := strings.Replace(string(before), "refresh_interval: 2m\n", "columns:\n  repo: 90 # narrow\n  diff: 60\nrefresh_interval: 2m\n", 1)
	_ = os.WriteFile(ConfigPath(), []byte(block), 0o644)
	if err := SaveColumns(map[string]int{"diff": 70}); err != nil {
		t.Fatal(err)
	}
	got, _ = os.ReadFile(ConfigPath())
	if want := strings.Replace(string(before), "refresh_interval: 2m\n", "columns: {diff: 70}\nrefresh_interval: 2m\n", 1); string(got) != want {
		t.Fatalf("replaced wrong:\n%s", got)
	}

	after, _ := os.ReadFile(ConfigPath())
	for _, bad := range []map[string]int{{"title": 100}, {"repo": 5}} {
		if err := SaveColumns(bad); err == nil {
			t.Fatalf("%v saved", bad)
		}
	}
	if got, _ = os.ReadFile(ConfigPath()); string(got) != string(after) {
		t.Fatal("a refused save changed the file")
	}
}
