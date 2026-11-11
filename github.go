package main

import (
	"fmt"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
)

type PR struct {
	Number         int
	Title          string
	URL            string
	Body           string
	IsDraft        bool
	Additions      int
	Deletions      int
	ChangedFiles   int
	HeadRefName    string
	BaseRefName    string
	ReviewDecision string // APPROVED, CHANGES_REQUESTED, REVIEW_REQUIRED, or "" when no review is required
	Mergeable      string // MERGEABLE, CONFLICTING, UNKNOWN
	CI             string // SUCCESS, FAILURE, ERROR, PENDING, EXPECTED, or "" when there are no checks
	Repo           string
	Author         string
	Labels         []string
	Unresolved     int
}

type sectionData struct {
	Section Section
	PRs     []PR
}

// shortcut: first 100 results per section and 100 review threads per PR, no pagination; add it if anyone hits the cap.
const searchQuery = `query($q: String!) {
  search(query: $q, type: ISSUE, first: 100) {
    nodes {
      ... on PullRequest {
        number title url body isDraft additions deletions changedFiles
        headRefName baseRefName reviewDecision mergeable
        repository { nameWithOwner }
        author { login }
        labels(first: 20) { nodes { name } }
        reviewThreads(first: 100) { nodes { isResolved } }
        commits(last: 1) { nodes { commit { statusCheckRollup { state } } } }
      }
    }
  }
}`

type searchResp struct {
	Search struct {
		Nodes []struct {
			Number         int
			Title          string
			URL            string
			Body           string
			IsDraft        bool
			Additions      int
			Deletions      int
			ChangedFiles   int
			HeadRefName    string
			BaseRefName    string
			ReviewDecision string
			Mergeable      string
			Repository     struct{ NameWithOwner string }
			Author         *struct{ Login string }
			Labels         struct{ Nodes []struct{ Name string } }
			ReviewThreads  struct{ Nodes []struct{ IsResolved bool } }
			Commits        struct {
				Nodes []struct {
					Commit struct {
						StatusCheckRollup *struct{ State string }
					}
				}
			}
		}
	}
}

func fetchAll(cfg Config) ([]sectionData, error) {
	client, err := api.DefaultGraphQLClient()
	if err != nil {
		return nil, err
	}
	out := make([]sectionData, 0, len(cfg.Sections))
	for _, s := range cfg.Sections {
		var resp searchResp
		if err := client.Do(searchQuery, map[string]interface{}{"q": s.Query}, &resp); err != nil {
			return nil, fmt.Errorf("%s: %w", s.Title, err)
		}
		sd := sectionData{Section: s}
		for _, n := range resp.Search.Nodes {
			if n.Number == 0 { // non-PR search hit
				continue
			}
			p := PR{
				Number:         n.Number,
				Title:          sanitize(n.Title),
				URL:            n.URL,
				Body:           sanitize(n.Body),
				IsDraft:        n.IsDraft,
				Additions:      n.Additions,
				Deletions:      n.Deletions,
				ChangedFiles:   n.ChangedFiles,
				HeadRefName:    sanitize(n.HeadRefName),
				BaseRefName:    sanitize(n.BaseRefName),
				ReviewDecision: n.ReviewDecision,
				Mergeable:      n.Mergeable,
				Repo:           n.Repository.NameWithOwner,
			}
			if n.Author != nil {
				p.Author = n.Author.Login
			}
			for _, l := range n.Labels.Nodes {
				p.Labels = append(p.Labels, sanitize(l.Name))
			}
			for _, t := range n.ReviewThreads.Nodes {
				if !t.IsResolved {
					p.Unresolved++
				}
			}
			if len(n.Commits.Nodes) > 0 && n.Commits.Nodes[0].Commit.StatusCheckRollup != nil {
				p.CI = n.Commits.Nodes[0].Commit.StatusCheckRollup.State
			}
			sd.PRs = append(sd.PRs, p)
		}
		out = append(out, sd)
	}
	return out, nil
}

// sanitize strips control characters so PR text written by others can't inject terminal escape sequences.
func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) {
			return -1
		}
		return r
	}, s)
}
