package core

// Your local arrangement of PRs: pins, project assignments, snoozes and hidden PRs.
// Kept in ~/.config/gh-prs/organize.yml, keyed by owner/repo#number; nothing goes to GitHub.

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"gopkg.in/yaml.v3"
)

type Organize struct {
	Pinned   []string            `yaml:"pinned,omitempty"`
	Projects map[string]string   `yaml:"projects,omitempty"` // PR → project name, used instead of its proj: label
	Snoozed  map[string]snapshot `yaml:"snoozed,omitempty"`  // PR → its state when snoozed; it returns when that changes
	Hidden   []string            `yaml:"hidden,omitempty"`
}

const organizeHeader = "# gh-prs: your local PR arrangement (pp pin · ps snooze · pm move · ph hide · pc ask Claude).\n# Keys are owner/repo#number. Nothing here is sent to GitHub.\n"

func OrganizePath() string { return filepath.Join(filepath.Dir(ConfigPath()), "organize.yml") }

func LoadOrganize() (*Organize, error) {
	b, err := os.ReadFile(OrganizePath())
	if os.IsNotExist(err) {
		return &Organize{}, nil
	}
	if err != nil {
		return nil, err
	}
	o, err := parseOrganize(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", OrganizePath(), err)
	}
	return o, nil
}

func parseOrganize(b []byte) (*Organize, error) {
	o := &Organize{}
	if err := decodeStrict(b, o); err != nil {
		return nil, err
	}
	return o, nil
}

func (o *Organize) Marshal() ([]byte, error) {
	b, err := yaml.Marshal(o)
	if err != nil {
		return nil, err
	}
	if string(b) == "{}\n" {
		b = nil
	}
	return append([]byte(organizeHeader), b...), nil
}

func (o *Organize) Save() error {
	b, err := o.Marshal()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(OrganizePath()), 0o755); err != nil {
		return err
	}
	return os.WriteFile(OrganizePath(), b, 0o644)
}

func (o *Organize) IsPinned(k string) bool { return slices.Contains(o.Pinned, k) }
func (o *Organize) hidden(k string) bool   { return slices.Contains(o.Hidden, k) }

func (o *Organize) IsSnoozed(k string) bool {
	_, ok := o.Snoozed[k]
	return ok
}

// toggle adds k to list, or removes it if present; it reports whether k is now in the list.
func Toggle(list *[]string, k string) bool {
	if i := slices.Index(*list, k); i >= 0 {
		*list = slices.Delete(*list, i, i+1)
		return false
	}
	*list = append(*list, k)
	return true
}

func (o *Organize) SetProject(k, name string) {
	if name == "" {
		delete(o.Projects, k)
		return
	}
	if o.Projects == nil {
		o.Projects = map[string]string{}
	}
	o.Projects[k] = name
}

func (o *Organize) Snooze(p PR) {
	if o.Snoozed == nil {
		o.Snoozed = map[string]snapshot{}
	}
	o.Snoozed[PrKey(p.Repo, p.Number)] = SnapOf(p)
}

// wake un-snoozes every PR that changed since it was snoozed, and reports whether any did.
func (o *Organize) Wake(data []SectionData) bool {
	woke := false
	for _, sd := range data {
		for _, p := range sd.PRs {
			k := PrKey(p.Repo, p.Number)
			if s, ok := o.Snoozed[k]; ok && len(Changes(s, SnapOf(p))) > 0 {
				delete(o.Snoozed, k)
				woke = true
			}
		}
	}
	return woke
}
