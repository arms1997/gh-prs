package core

// GitHub: one GraphQL search per section, mapped to PRs.

import (
	"fmt"
	"regexp"
	"strings"
	"time"

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
	HeadOID        string
	ReviewDecision string // APPROVED, CHANGES_REQUESTED, REVIEW_REQUIRED, or "" when no review is required
	Mergeable      string // MERGEABLE, CONFLICTING, UNKNOWN
	CI             string // SUCCESS, FAILURE, ERROR, PENDING, EXPECTED, or "" when there are no checks
	FailingChecks  []Check
	Repo           string
	Author         string
	Labels         []string
	Unresolved     int
	Comments       int    // conversation comments plus reviews, to notice new activity
	Linear         string // issue ID from the title or branch, e.g. AGT-915

	// From your organize file, set when the list is built.
	Project string `json:"-"`
	Pinned  bool   `json:"-"`
	Hidden  bool   `json:"-"`
	Snoozed bool   `json:"-"`

	Mine              bool // you opened it
	UnresolvedNotMine int  // unresolved threads where someone else spoke last
	ReviewedSincePush bool // you reviewed it after its latest commit
}

type Check struct {
	Name string
	URL  string
}

type SectionData struct {
	Section Section
	PRs     []PR
}

// shortcut: at most 100 results per section, 100 review threads and 50 checks per PR, no pagination; add it if anyone hits the cap.
// GitHub charges by the page sizes asked for, not the results: about 1 point per result asked for.
const searchQuery = `query($q: String!, $n: Int!) {
  viewer { login }
  rateLimit { remaining resetAt }
  search(query: $q, type: ISSUE, first: $n) {
    nodes {
      ... on PullRequest {
        number title url body isDraft additions deletions changedFiles
        headRefName baseRefName headRefOid reviewDecision mergeable
        repository { nameWithOwner }
        author { login }
        labels(first: 20) { nodes { name } }
        comments { totalCount }
        reviewCount: reviews { totalCount }
        reviews(last: 30) { nodes { author { login } submittedAt } }
        reviewThreads(first: 100) { nodes { isResolved comments(last: 1) { nodes { author { login } } } } }
        commits(last: 1) { nodes { commit { committedDate statusCheckRollup { state
          contexts(first: 50) { nodes {
            __typename
            ... on CheckRun { name conclusion detailsUrl }
            ... on StatusContext { context state targetUrl }
          } }
        } } } }
      }
    }
  }
}`

type login struct{ Login string }

// rateLimit is what's left of the GraphQL budget (5,000 points an hour, shared by everything using your token).
type RateLimit struct {
	Remaining int
	ResetAt   time.Time
}

type searchResp struct {
	Viewer    login
	RateLimit RateLimit
	Search    struct {
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
			HeadRefOid     string
			ReviewDecision string
			Mergeable      string
			Repository     struct{ NameWithOwner string }
			Author         *login
			Labels         struct{ Nodes []struct{ Name string } }
			Comments       struct{ TotalCount int }
			ReviewCount    struct{ TotalCount int }
			Reviews        struct {
				Nodes []struct {
					Author      *login
					SubmittedAt time.Time
				}
			}
			ReviewThreads struct {
				Nodes []struct {
					IsResolved bool
					Comments   struct{ Nodes []struct{ Author *login } }
				}
			}
			Commits struct {
				Nodes []struct {
					Commit struct {
						CommittedDate     time.Time
						StatusCheckRollup *struct {
							State    string
							Contexts struct {
								Nodes []struct {
									Typename   string `json:"__typename"`
									Name       string
									Conclusion string
									DetailsURL string `json:"detailsUrl"`
									Context    string
									State      string
									TargetURL  string `json:"targetUrl"`
								}
							}
						}
					}
				}
			}
		}
	}
}

// failedConclusions are check outcomes that count as a failure.
var failedConclusions = map[string]bool{"FAILURE": true, "TIMED_OUT": true, "STARTUP_FAILURE": true, "ACTION_REQUIRED": true, "ERROR": true}

// pageSize asks for a little more than the section had last time, since cost follows the page size.
// A section not seen before gets the maximum.
func pageSize(prev []SectionData, title string) int {
	for _, sd := range prev {
		if sd.Section.Title == title {
			return min(max(len(sd.PRs)+10, 10), 100)
		}
	}
	return 100
}

func FetchAll(cfg Config, prev []SectionData) ([]SectionData, RateLimit, error) {
	var rate RateLimit
	client, err := api.DefaultGraphQLClient()
	if err != nil {
		return nil, rate, err
	}
	out := make([]SectionData, 0, len(cfg.Sections))
	for _, s := range cfg.Sections {
		var resp searchResp
		for n := pageSize(prev, s.Title); ; n = 100 {
			resp = searchResp{}
			if err := client.Do(searchQuery, map[string]interface{}{"q": s.Query, "n": n}, &resp); err != nil {
				return nil, rate, fmt.Errorf("%s: %w", s.Title, err)
			}
			if len(resp.Search.Nodes) < n || n == 100 { // a full page may have been cut off
				break
			}
		}
		rate = resp.RateLimit
		me := resp.Viewer.Login
		sd := SectionData{Section: s}
		for _, n := range resp.Search.Nodes {
			if n.Number == 0 { // non-PR search hit
				continue
			}
			p := PR{
				Number:         n.Number,
				Title:          Sanitize(n.Title),
				URL:            n.URL,
				Body:           Sanitize(n.Body),
				IsDraft:        n.IsDraft,
				Additions:      n.Additions,
				Deletions:      n.Deletions,
				ChangedFiles:   n.ChangedFiles,
				HeadRefName:    Sanitize(n.HeadRefName),
				BaseRefName:    Sanitize(n.BaseRefName),
				HeadOID:        n.HeadRefOid,
				ReviewDecision: n.ReviewDecision,
				Mergeable:      n.Mergeable,
				Repo:           n.Repository.NameWithOwner,
				Comments:       n.Comments.TotalCount + n.ReviewCount.TotalCount,
			}
			if n.Author != nil {
				p.Author = n.Author.Login
				p.Mine = p.Author == me
			}
			for _, l := range n.Labels.Nodes {
				p.Labels = append(p.Labels, Sanitize(l.Name))
			}
			for _, t := range n.ReviewThreads.Nodes {
				if t.IsResolved {
					continue
				}
				p.Unresolved++
				if c := t.Comments.Nodes; len(c) > 0 && (c[0].Author == nil || c[0].Author.Login != me) {
					p.UnresolvedNotMine++
				}
			}
			var committed time.Time
			if len(n.Commits.Nodes) > 0 {
				c := n.Commits.Nodes[0].Commit
				committed = c.CommittedDate
				if r := c.StatusCheckRollup; r != nil {
					p.CI = r.State
					for _, ctx := range r.Contexts.Nodes {
						switch {
						case ctx.Typename == "CheckRun" && failedConclusions[ctx.Conclusion]:
							p.FailingChecks = append(p.FailingChecks, Check{Sanitize(ctx.Name), ctx.DetailsURL})
						case ctx.Typename == "StatusContext" && failedConclusions[ctx.State]:
							p.FailingChecks = append(p.FailingChecks, Check{Sanitize(ctx.Context), ctx.TargetURL})
						}
					}
				}
			}
			for _, r := range n.Reviews.Nodes {
				if r.Author != nil && r.Author.Login == me && r.SubmittedAt.After(committed) {
					p.ReviewedSincePush = true
				}
			}
			p.Linear = linearID(p.Title, p.HeadRefName)
			sd.PRs = append(sd.PRs, p)
		}
		out = append(out, sd)
	}
	return out, rate, nil
}

var linearRe = regexp.MustCompile(`\b([A-Z][A-Z0-9]{1,9}-[0-9]{1,6})\b`)

// linearID finds an issue ID like AGT-915 in the title, else in the branch (agt-915-fix-thing).
// shortcut: any TEAM-123 shape counts, so a title mentioning e.g. "SHA-256" would match; filter by team keys if that bites.
func linearID(title, branch string) string {
	if m := linearRe.FindStringSubmatch(title); m != nil {
		return m[1]
	}
	if m := linearRe.FindStringSubmatch(strings.ToUpper(branch)); m != nil {
		return m[1]
	}
	return ""
}

// linearURL links an issue ID into your Linear workspace, or returns "" if either is unknown.
func LinearURL(workspace, id string) string {
	if workspace == "" || id == "" {
		return ""
	}
	return "https://linear.app/" + workspace + "/issue/" + id
}

// sanitize strips control characters so PR text written by others can't inject terminal escape sequences.
func Sanitize(s string) string {
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

// LowRate is the GraphQL budget below which auto-refresh waits for the hourly reset, leaving the
// rest for agents, gh and everything else using your token. r still refreshes.
const LowRate = 500

// RefreshDelay is how long until the next auto-refresh: the interval, or until the budget resets when it's low.
func RefreshDelay(every time.Duration, rate RateLimit, now time.Time) time.Duration {
	if rate.ResetAt.After(now) && rate.Remaining < LowRate {
		return max(rate.ResetAt.Sub(now), every)
	}
	return every
}
