package core

// Grouping: status buckets, group_by keys, stacking, and the rows the list shows.

import (
	"fmt"
	"slices"
	"sort"
	"strings"
)

type Status int

const (
	NeedsAction Status = iota
	WaitingReview
	ReadyToMerge
	Draft
)

var statusNames = [...]string{"Needs action", "Waiting for review", "Ready to merge", "Draft"}

func (s Status) String() string { return statusNames[s] }

func StatusOf(p PR) Status {
	switch {
	case p.IsDraft:
		return Draft
	case p.ReviewDecision == "CHANGES_REQUESTED", p.CI == "FAILURE", p.CI == "ERROR", p.Mergeable == "CONFLICTING":
		return NeedsAction
	case p.ReviewDecision == "APPROVED" && (p.CI == "SUCCESS" || p.CI == ""):
		return ReadyToMerge
	}
	return WaitingReview
}

// A grouper returns the group a PR belongs to and a key that orders groups.
type grouper func(p PR, cfg Config) (name, rank string)

var groupers = map[string]grouper{
	"status": func(p PR, _ Config) (string, string) {
		s := StatusOf(p)
		return s.String(), fmt.Sprint(int(s))
	},
	"project": func(p PR, cfg Config) (string, string) {
		if p.Project != "" { // assigned by you (pm), which wins over labels
			return p.Project, "0" + p.Project
		}
		if cfg.ProjectLabelPrefix != "" {
			for _, l := range p.Labels {
				if strings.HasPrefix(l, cfg.ProjectLabelPrefix) {
					name := strings.TrimPrefix(l, cfg.ProjectLabelPrefix)
					return name, "0" + name
				}
			}
		}
		return "No project", "1"
	},
	"repo": func(p PR, _ Config) (string, string) { return p.Repo, p.Repo },
	"turn": func(p PR, _ Config) (string, string) {
		if mine, _ := TurnOf(p); mine {
			return "On me", "0"
		}
		return "On them", "1"
	},
}

// turnOf says whether the next move on a PR is yours, and why. Your PR: a draft, anything
// that needs action, a thread where someone else spoke last, or ready to merge. Someone
// else's PR (a review request): yours until you've reviewed it since its latest commit.
func TurnOf(p PR) (onMe bool, why string) {
	if !p.Mine {
		if p.ReviewedSincePush {
			return false, "you reviewed it; waiting for new commits"
		}
		return true, "your review is requested"
	}
	var reasons []string
	switch StatusOf(p) {
	case Draft:
		reasons = append(reasons, "it's a draft")
	case ReadyToMerge:
		reasons = append(reasons, "ready to merge")
	case NeedsAction:
		if p.ReviewDecision == "CHANGES_REQUESTED" {
			reasons = append(reasons, "changes requested")
		}
		if p.CI == "FAILURE" || p.CI == "ERROR" {
			reasons = append(reasons, "CI failing")
		}
		if p.Mergeable == "CONFLICTING" {
			reasons = append(reasons, "merge conflict")
		}
	}
	if n := p.UnresolvedNotMine; n > 0 {
		reasons = append(reasons, plural(n, "thread waiting on your reply", "threads waiting on your reply"))
	}
	if len(reasons) == 0 {
		return false, "waiting for review"
	}
	return true, strings.Join(reasons, ", ")
}

// matches reports whether a PR contains every space-separated term of the filter, ignoring case.
func matches(p PR, filter string) bool {
	hay := strings.ToLower(fmt.Sprintf("#%d %s %s %s %s %s", p.Number, p.Title, p.Repo, p.HeadRefName, p.Linear, p.Author))
	for _, term := range strings.Fields(strings.ToLower(filter)) {
		if !strings.Contains(hay, term) {
			return false
		}
	}
	return true
}

// Row is one line of the list: a group header when PR is nil.
type Row struct {
	Title string
	Depth int
	Count int
	Path  string // unique id, used for collapse state and keeping the cursor across refreshes
	PR    *PR

	Stack     int // stacking depth under a parent PR shown in the same group
	StackedOn int // number of the PR this one is stacked on, if that PR is in the section
}

func branchKey(repo, branch string) string { return repo + ":" + branch }

// view is how the list is being looked at: folds, the / filter, and your organize file.
type View struct {
	Collapsed map[string]bool
	Filter    string
	Org       *Organize
	ShowAll   bool // include hidden and snoozed PRs (marked)
}

// buildRows lays out every section as rows: your project assignments and pins applied,
// hidden and snoozed PRs left out (unless ShowAll), and only PRs matching the filter.
func BuildRows(data []SectionData, cfg Config, v View) []Row {
	collapsed, org := v.Collapsed, v.Org
	if org == nil {
		org = &Organize{}
	}
	var rows []Row
	for _, sd := range data {
		heads := map[string]int{} // from every PR, so a filtered-out parent still shows as "(on #N)"
		var prs []PR
		for _, p := range sd.PRs {
			heads[branchKey(p.Repo, p.HeadRefName)] = p.Number
			k := PrKey(p.Repo, p.Number)
			p.Project, p.Pinned = org.Projects[k], org.IsPinned(k)
			p.Hidden, p.Snoozed = org.hidden(k), org.IsSnoozed(k)
			if (p.Hidden || p.Snoozed) && !v.ShowAll || v.Filter != "" && !matches(p, v.Filter) {
				continue
			}
			prs = append(prs, p)
		}
		slices.SortStableFunc(prs, func(a, b PR) int { return btoi(b.Pinned) - btoi(a.Pinned) }) // pinned first, else keep GitHub's order
		path := sd.Section.Title
		rows = append(rows, Row{Title: sd.Section.Title, Count: len(prs), Path: path})
		if !collapsed[path] {
			rows = append(rows, groupRows(prs, sd.Section.GroupBy, 1, path, cfg, collapsed, heads)...)
		}
	}
	return rows
}

type stackItem struct {
	pr    *PR
	depth int
}

// stackOrder puts each PR right after the PR it is stacked on (base branch == parent's head branch).
func stackOrder(prs []PR) []stackItem {
	idx := map[string]int{}
	for i, p := range prs {
		idx[branchKey(p.Repo, p.HeadRefName)] = i
	}
	children := map[int][]int{}
	var roots []int
	for i, p := range prs {
		if j, ok := idx[branchKey(p.Repo, p.BaseRefName)]; ok && j != i {
			children[j] = append(children[j], i)
		} else {
			roots = append(roots, i)
		}
	}
	out := make([]stackItem, 0, len(prs))
	seen := make([]bool, len(prs))
	var walk func(i, d int)
	walk = func(i, d int) {
		if seen[i] {
			return
		}
		seen[i] = true
		out = append(out, stackItem{&prs[i], d})
		for _, c := range children[i] {
			walk(c, d+1)
		}
	}
	for _, r := range roots {
		walk(r, 0)
	}
	for i := range prs { // PRs in a base-branch cycle have no root
		walk(i, 0)
	}
	return out
}

func groupRows(prs []PR, by []string, depth int, path string, cfg Config, collapsed map[string]bool, heads map[string]int) []Row {
	var rows []Row
	if len(by) == 0 {
		for _, it := range stackOrder(prs) {
			p := it.pr
			rows = append(rows, Row{
				Depth:     depth,
				Path:      fmt.Sprintf("%s/%s#%d", path, p.Repo, p.Number),
				PR:        p,
				Stack:     it.depth,
				StackedOn: heads[branchKey(p.Repo, p.BaseRefName)],
			})
		}
		return rows
	}
	type bucket struct {
		name, rank string
		prs        []PR
	}
	byName := map[string]*bucket{}
	var buckets []*bucket
	for _, p := range prs {
		name, rank := groupers[by[0]](p, cfg)
		b := byName[name]
		if b == nil {
			b = &bucket{name: name, rank: rank}
			byName[name] = b
			buckets = append(buckets, b)
		}
		b.prs = append(b.prs, p)
	}
	sort.SliceStable(buckets, func(i, j int) bool { return buckets[i].rank < buckets[j].rank })
	for _, b := range buckets {
		p := path + "/" + b.name
		rows = append(rows, Row{Title: b.name, Depth: depth, Count: len(b.prs), Path: p})
		if !collapsed[p] {
			rows = append(rows, groupRows(b.prs, by[1:], depth+1, p, cfg, collapsed, heads)...)
		}
	}
	return rows
}
