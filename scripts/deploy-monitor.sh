#!/bin/sh
# gh-prs: merge a PR, deploy its service(s) to prod with the repo's "Deploy Services" workflow,
# then have Claude check Datadog at intervals. Prints Claude-style stream-json so gh-prs shows
# progress in its agent column. Every check runs before anything is merged.
#
# Needs: gh, jq, claude. Reads PR_URL, PR_REPO, PR_NUMBER from gh-prs.
# Optional: DEPLOY_DRY_RUN=1 (check and plan only), DEPLOY_RUNNER (default ubuntu-latest),
#           DEPLOY_CHECK_MINUTES (default "5 10 15 20"), DATADOG_MCP (default datadog-mcp).
set -u

WORKFLOW=dispatch_deploy.yml
RUNNER=${DEPLOY_RUNNER:-ubuntu-latest}
CHECKS=${DEPLOY_CHECK_MINUTES:-5 10 15 20}
DD=${DATADOG_MCP:-datadog-mcp}
COST=0

emit() { jq -cn --arg t "$1" '{type: "assistant", message: {content: [{type: "text", text: $t}]}}'; }
finish() { # finish ok|fail <summary>
  err=true
  [ "$1" = ok ] && err=false
  jq -cn --arg r "$2" --argjson e "$err" --argjson c "$COST" \
    '{type: "result", subtype: "success", is_error: $e, result: $r, num_turns: 0, total_cost_usd: $c}'
  exit 0
}
fail() { finish fail "$1"; }

# Mirrors .github/actions/prod-deploy-check: Mon–Thu 3AM–7PM PT, Fri 3AM–2PM ET (in UTC).
in_window() {
  d=$(date -u +%u)
  h=$(date -u +%H | sed 's/^0//')
  if [ "$d" -ge 1 ] && [ "$d" -le 4 ] && [ "$h" -ge 11 ]; then return 0; fi
  if [ "$d" -ge 2 ] && [ "$d" -le 5 ] && [ "$h" -lt 3 ]; then return 0; fi
  if [ "$d" = 5 ] && [ "$h" -ge 11 ] && [ "$h" -lt 19 ]; then return 0; fi
  return 1
}

# --- Checks: nothing below changes anything until "Merge" ---------------------------------

in_window || fail "Outside the prod deploy window (Mon–Thu 3AM–7PM PT, Fri until 2PM ET). Nothing was merged or deployed."

pr=$(gh pr view "$PR_URL" --json id,title,state,isDraft,mergeStateStatus,reviewDecision,baseRefName,files 2>&1) ||
  fail "Couldn't read the PR: $(printf %s "$pr" | tail -1)"
field() { printf %s "$pr" | jq -r "$1"; }

main=$(gh api "repos/$PR_REPO" --jq .default_branch) || fail "Couldn't read the repo's default branch."
[ "$(field .state)" = OPEN ] || fail "PR is $(field .state), not open."
[ "$(field .isDraft)" = false ] || fail "PR is a draft."
[ "$(field .baseRefName)" = "$main" ] || fail "PR targets $(field .baseRefName), not $main: merge its base first."
case "$(field .reviewDecision)" in APPROVED | "") ;; *) fail "PR isn't approved ($(field .reviewDecision))." ;; esac
case "$(field .mergeStateStatus)" in
  CLEAN | HAS_HOOKS) ;;
  UNSTABLE) fail "Some checks are failing (merge state UNSTABLE). Fix or re-run them first." ;;
  *) fail "PR can't be merged yet (merge state $(field .mergeStateStatus))." ;;
esac

wf=$(gh api "repos/$PR_REPO/contents/.github/workflows/$WORKFLOW?ref=$main" --jq .content 2>/dev/null | base64 -d 2>/dev/null)
services=""
if [ -n "$wf" ]; then
  valid=$(printf '%s\n' "$wf" | awk '
    /^      deploy-service:/ { s = 1; next }
    s && /^      [a-z]/ { exit }
    s && /^ *- / { sub(/^ *- */, ""); print }')
  [ -n "$valid" ] || fail "$WORKFLOW has no deploy-service options; not deploying."
  scope=$(field .title | sed -nE 's/^[a-z]+\(([^)]*)\)!?:.*/\1/p')
  [ -n "$scope" ] || fail "Can't tell which service to deploy: the PR title has no scope like fix(agent-registry)."
  files=$(field '.files[].path')
  for s in $(printf %s "$scope" | tr ',' ' '); do
    printf '%s\n' "$valid" | grep -qx "$s" || fail "Title scope '$s' isn't a deployable service in $WORKFLOW."
    printf '%s\n' "$files" | grep -qE "(^|/)$s/" || fail "Title says '$s' but no changed file is under a $s/ directory; not deploying."
    services="$services $s"
  done
fi
services=${services# }

method=squash
allowed=$(gh api "repos/$PR_REPO" --jq '[if .allow_squash_merge then "squash" else empty end, if .allow_merge_commit then "merge" else empty end, if .allow_rebase_merge then "rebase" else empty end] | join(" ")')
case " $allowed " in *" squash "*) ;; *) method=${allowed%% *} ;; esac
[ -n "$method" ] || fail "The repo allows no merge method gh can use."

if [ -n "$services" ]; then
  plan="Merge #$PR_NUMBER ($method) → deploy $services to prod → health checks at +$(printf %s "$CHECKS" | sed 's/ /, +/g') min."
else
  plan="Merge #$PR_NUMBER ($method). $PR_REPO has no $WORKFLOW, so nothing will be deployed."
fi
emit "Plan: $plan"
[ "${DEPLOY_DRY_RUN:-}" = 1 ] && finish ok "Dry run, all checks passed. $plan Nothing was merged or deployed."

# --- Merge --------------------------------------------------------------------------------

out=$(gh pr merge "$PR_URL" "--$method" 2>&1) || fail "Merge failed: $(printf %s "$out" | tail -1)"
emit "Merge requested ($method); waiting for GitHub to merge it."
sha=""
for _ in $(seq 180); do # up to an hour, for merge queues
  st=$(gh pr view "$PR_URL" --json state,mergeCommit --jq '.state + " " + (.mergeCommit.oid // "")' 2>/dev/null)
  case "$st" in
    MERGED*) sha=${st#MERGED } && break ;;
    CLOSED*) fail "PR was closed without merging." ;;
  esac
  sleep 20
done
[ -n "$sha" ] || fail "Still not merged after an hour; nothing was deployed."
emit "Merged as ${sha%"${sha#???????}"}."
[ -n "$services" ] || finish ok "Merged #$PR_NUMBER as ${sha%"${sha#???????}"}. No deploy workflow in $PR_REPO, so nothing was deployed."

# --- Deploy -------------------------------------------------------------------------------

me=$(gh api user --jq .login) || fail "Merged, but couldn't read your GitHub login to find the deploy runs. Deploy manually."
runs=""
for s in $services; do
  since=$(date -u +%Y-%m-%dT%H:%M:%SZ)
  out=$(gh workflow run "$WORKFLOW" -R "$PR_REPO" --ref "$main" -f deploy-service="$s" -f deploy-env=prod -f deploy-runner="$RUNNER" 2>&1) ||
    fail "Merged, but starting the $s deploy failed: $(printf %s "$out" | tail -1). Press V to open a revert PR, or deploy manually."
  id=""
  for _ in $(seq 24); do # the run takes a few seconds to appear
    sleep 5
    for cand in $(gh run list -R "$PR_REPO" --workflow "$WORKFLOW" --event workflow_dispatch --user "$me" --limit 10 \
      --json databaseId,createdAt --jq "[.[] | select(.createdAt >= \"$since\")] | sort_by(.createdAt) | .[].databaseId" 2>/dev/null); do
      case " $runs " in *" $cand "*) ;; *) id=$cand && break ;; esac
    done
    [ -n "$id" ] && break
  done
  [ -n "$id" ] || fail "Merged and started the $s deploy, but couldn't find its run. Check Actions in $PR_REPO."
  runs="$runs $id"
  emit "Deploying $s to prod: https://github.com/$PR_REPO/actions/runs/$id"
done

for _ in $(seq 120); do # up to an hour
  pending=0
  for id in $runs; do
    st=$(gh run view "$id" -R "$PR_REPO" --json status,conclusion --jq '.status + " " + .conclusion' 2>/dev/null)
    case "$st" in
      "completed success") ;;
      completed*) fail "Deploy run $id finished as '${st#completed }': https://github.com/$PR_REPO/actions/runs/$id. The PR is merged; press V to open a revert PR." ;;
      *) pending=$((pending + 1)) ;;
    esac
  done
  [ "$pending" = 0 ] && break
  sleep 30
done
[ "$pending" = 0 ] || fail "Deploys still running after an hour: check Actions in $PR_REPO."
deployed=$(date -u +%Y-%m-%dT%H:%M:%SZ)
emit "Deployed $services to prod at $deployed."

# --- Health checks ------------------------------------------------------------------------

tools=""
for t in search_datadog_logs analyze_datadog_logs search_datadog_monitors get_datadog_metric get_datadog_metric_context \
  search_datadog_metrics search_datadog_spans aggregate_spans search_datadog_events search_datadog_incidents \
  list_datadog_skills load_datadog_skill search_datadog_services search_datadog_entities; do
  tools="$tools,mcp__${DD}__$t"
done
tools=${tools#,}

prev=0
n=0
total=$(printf '%s\n' $CHECKS | wc -l | tr -d ' ')
evidence=""
for m in $CHECKS; do
  sleep $(((m - prev) * 60))
  prev=$m
  n=$((n + 1))
  prompt="Read-only prod health check $n of $total, $m minutes after deploying $services to prod at $deployed (PR $PR_URL).
Use Datadog, prod only (env:prod). For each service, compare the time since the deploy with the hour before it:
error logs, APM error rate and latency, and any monitors for the service that are alerting. Service tags may differ
slightly from these names; find the right ones. Don't change anything in Datadog.
Reply with a first line that is exactly HEALTHY or UNHEALTHY, then at most 5 short lines of evidence with numbers."
  out=$(env -u ANTHROPIC_API_KEY claude -p "$prompt" --output-format json --permission-mode dontAsk --allowedTools "$tools" 2>/dev/null)
  COST=$(awk -v a="$COST" -v b="$(printf %s "$out" | jq -r '.total_cost_usd // 0' 2>/dev/null)" 'BEGIN { print a + b }')
  result=$(printf %s "$out" | jq -r '.result // ""' 2>/dev/null)
  verdict=$(printf '%s\n' "$result" | head -1 | tr -d '*`# ')
  evidence=$(printf '%s\n' "$result" | sed 1d | sed '/^ *$/d' | head -5)
  if [ "$(printf %s "$out" | jq -r '.is_error' 2>/dev/null)" != false ] || [ -z "$verdict" ]; then
    fail "Deployed $services, but health check $n (+${m}m) couldn't run: $(printf '%s\n' "$result" | head -1). Check Datadog yourself; press V to open a revert PR if needed."
  fi
  emit "Health check $n/$total (+${m}m): $verdict"
  [ "$verdict" = HEALTHY ] || fail "UNHEALTHY at +${m}m after deploying $services:
$evidence
Press V to open a revert PR."
done

finish ok "Merged #$PR_NUMBER as ${sha%"${sha#???????}"}, deployed $services to prod, healthy through +${prev}m.
$evidence"
