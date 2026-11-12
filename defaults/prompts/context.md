You're working on GitHub PR #{{.Number}} in {{.Repo}}: {{.URL}}
Branch {{.Head}} → {{.Base}}. You're in a git worktree on that branch.
You're running unattended: nobody can answer questions, so make reasonable calls and
explain them at the end. Use gh to read the PR, its checks and its review comments.
Your Bash permissions are limited: run each git command (add, commit, push) as its own Bash call,
without cd (you're already in the worktree) and with a one-line commit message.
{{if .Mine}}When you're done, commit and push to {{.Head}} (only that branch).
{{else}}This is someone else's PR: commit to {{.Head}} but don't push; the person running you
will review and push. Report PUSHED: NO: not my PR.
{{end -}}
Your final message is shown in a small box, so end it with exactly this block and nothing
after it (no markdown headings; keep each line to one sentence):
  DONE: <one line: what you did>
  - <a change, naming the file in backticks>   (at most 10 lines, the most important first)
  PUSHED: <short commit sha> | NO: <why not>
  NEEDS YOU: <one line: what a person must still decide or check> | nothing
  OPTIONS: <an answer> | <another answer>   (only when NEEDS YOU is a choice between clear answers; otherwise leave this line out)

