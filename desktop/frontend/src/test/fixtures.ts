// Test fixtures: a PR, job, command or row with sensible defaults, overridden per test.
import {Command, Job, PR, Row} from '../lib/pr'

export function pr(over: Partial<PR> = {}): PR {
    return {
        Number: 1, Title: 't', URL: 'https://github.com/o/r/pull/1', Body: '', IsDraft: false,
        Additions: 1, Deletions: 1, ChangedFiles: 1, HeadRefName: 'h', BaseRefName: 'main', HeadOID: 'abc',
        ReviewDecision: 'REVIEW_REQUIRED', Mergeable: 'MERGEABLE', CI: 'SUCCESS', FailingChecks: [], Checks: [],
        Repo: 'o/r', Author: 'me', Labels: [], Unresolved: 0, Comments: 0, Linear: '', Mine: true,
        UnresolvedNotMine: 0, ReviewedSincePush: false, Key: `o/r#${over.Number ?? 1}`, Status: '', OnMe: true,
        TurnWhy: '', Project: '', Pinned: false, Hidden: false, Snoozed: false, LinearURL: '',
        ...over,
    } as PR
}

export function job(over: Partial<Job> = {}): Job {
    return {
        ID: 'j1', Name: 'manage', Key: 'cm', PRKey: 'o/r#1', Repo: 'o/r', Number: 1, State: 'running',
        Activity: [], Rounds: [], Result: '', Cost: 0, Turns: 0, Elapsed: '1m', Started: new Date().toISOString(),
        ...over,
    } as Job
}

export function cmd(over: Partial<Command>): Command {
    return {Key: 'x', Name: 'x', Background: false, Confirm: false, Builtin: '', Input: '', Session: false, ...over} as Command
}

// Default commands, as defaults/commands/*.yml defines them (only the fields the UI reads).
export const COMMANDS: Command[] = [
    cmd({Key: 'cc', Name: 'get it mergeable', Background: true}),
    cmd({Key: 'cm', Name: 'manage', Background: true}),
    cmd({Key: 'ci', Name: 'ask Claude', Background: true, Input: 'Ask Claude about this PR'}),
    cmd({Key: 'cy', Name: 'reply to agent', Background: true, Session: true, Input: 'Reply to Claude'}),
    cmd({Key: 'ct', Name: 'take over agent', Session: true}),
    cmd({Key: 'cx', Name: 'stop agent', Builtin: 'stop_agent', Confirm: true}),
    cmd({Key: 'M', Name: 'merge', Confirm: true}),
]

export const group = (Path: string, Depth: number, Count: number): Row => ({Title: Path.split('/').pop(), Path, Depth, Count, Stack: 0, StackedOn: 0} as Row)
export const prRow = (group: string, p: PR, Depth = 2): Row => ({Path: `${group}/${p.Key}`, Depth, Count: 0, Stack: 0, StackedOn: 0, PR: p} as Row)
