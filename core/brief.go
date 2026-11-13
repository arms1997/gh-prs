package core

// The PR brief: Claude's written account of what's left before a PR can merge. Claude runs with no
// tools (RunClaude) and only gets the data fetched here, so text in the PR can't make it act.

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
)

const briefQuery = `query($owner: String!, $name: String!, $number: Int!) {
  viewer { login }
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      title body isDraft author { login } baseRefName headRefName
      mergeable mergeStateStatus reviewDecision
      reviewRequests(first: 10) { nodes { requestedReviewer { ... on User { login } ... on Team { name } } } }
      reviews(last: 20) { nodes { author { login } state body submittedAt } }
      reviewThreads(first: 50) { nodes { isResolved isOutdated path line
        comments(last: 3) { nodes { author { login } body createdAt } } } }
      comments(last: 15) { nodes { author { login } body createdAt } }
      commits(last: 1) { nodes { commit { committedDate statusCheckRollup { state
        contexts(first: 50) { nodes {
          ... on CheckRun { name status conclusion }
          ... on StatusContext { context state } } } } } } }
    }
  }
}`

// shortcut: PR data past this many bytes is cut off, which drops the last threads and comments of a
// very busy PR; summarize it in Go first if that loses things that matter.
const briefDataLimit = 60_000

// PRBrief fetches what's outstanding on a PR and asks Claude to brief you on it, in Markdown.
func PRBrief(p PR) (string, float64, error) {
	data, err := briefData(p)
	if err != nil {
		return "", 0, err
	}
	tmpl, err := readPrompt("brief.md")
	if err != nil {
		return "", 0, err
	}
	text, cost, err := RunClaude(briefPrompt(tmpl, data))
	return Sanitize(text), cost, err
}

func briefData(p PR) (string, error) {
	owner, name, ok := strings.Cut(p.Repo, "/")
	if !ok {
		return "", fmt.Errorf("bad repo %q", p.Repo)
	}
	client, err := api.DefaultGraphQLClient()
	if err != nil {
		return "", err
	}
	var resp map[string]any
	if err := client.Do(briefQuery, map[string]any{"owner": owner, "name": name, "number": p.Number}, &resp); err != nil {
		return "", fmt.Errorf("couldn't load #%d: %w", p.Number, err)
	}
	b, err := json.Marshal(resp)
	return string(b), err
}

// briefPrompt puts the fetched data after the instructions, fenced as data: it's written by other people.
func briefPrompt(tmpl, data string) string {
	if len(data) > briefDataLimit {
		data = data[:briefDataLimit] + " …(cut off)"
	}
	data = strings.ReplaceAll(data, "</pr_data>", "<\\/pr_data>")
	return tmpl + "\n<pr_data>\n" + data + "\n</pr_data>\n"
}
