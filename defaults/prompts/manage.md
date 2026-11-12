You manage this PR until it merges, one round at a time.
{{if not .Mine}}
This isn't your PR, so there's nothing for you to manage: report that and stop the loop.
{{else}}
Each round:
1. Check the PR: `gh pr view {{.Number}} -R {{.Repo}} --json state,mergeable,mergeStateStatus,reviewDecision`
   and its unresolved review threads.
2. Merged or closed: stop the loop and report it.
3. Otherwise do what's needed, in this order:
   - Merge conflicts (mergeable is CONFLICTING): fetch, rebase onto origin/{{.Base}}, resolve them.
     If a conflict needs a decision you can't make from the code, run `git rebase --abort` and
     put it under NEEDS YOU.
   - Behind (mergeStateStatus is BEHIND): rebase onto origin/{{.Base}}.
   - Unresolved review threads where a reviewer spoke last: address each one you haven't
     addressed in an earlier round. Don't reply to or resolve threads. A question, or a
     comment you disagree with, goes under NEEDS YOU instead.
   Run the lint and tests for what you changed. After a rebase push with --force-with-lease,
   otherwise push normally.
4. Nothing needed: change nothing. Say so in one line (DONE: nothing to do).

Then schedule the next round in 15 minutes (900 seconds), and end the round with the summary
block. Keep rounds short; don't re-check what you already handled.
{{end}}
