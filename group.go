package main

import (
	"fmt"
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

func statusOf(p PR) Status {
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
		s := statusOf(p)
		return s.String(), fmt.Sprint(int(s))
	},
	"project": func(p PR, cfg Config) (string, string) {
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

func buildRows(data []sectionData, cfg Config, collapsed map[string]bool) []Row {
	var rows []Row
	for _, sd := range data {
		heads := map[string]int{}
		for _, p := range sd.PRs {
			heads[branchKey(p.Repo, p.HeadRefName)] = p.Number
		}
		path := sd.Section.Title
		rows = append(rows, Row{Title: sd.Section.Title, Count: len(sd.PRs), Path: path})
		if !collapsed[path] {
			rows = append(rows, groupRows(sd.PRs, sd.Section.GroupBy, 1, path, cfg, collapsed, heads)...)
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
