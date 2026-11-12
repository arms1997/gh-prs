// What a PR's state means for you: the badges worth showing, the next step, the quick filters.
import {main} from '../../wailsjs/go/models'
import type {Tone} from '../components/ui/Badge'
import type {IconName} from '../components/ui/Icon'

export type Row = main.RowView
export type PR = main.PRView
export type Job = main.JobState
export type Command = main.CommandView

export const ciFailing = (p: PR) => p.CI === 'FAILURE' || p.CI === 'ERROR'
export const ciRunning = (p: PR) => p.CI === 'PENDING' || p.CI === 'EXPECTED'
export const conflict = (p: PR) => p.Mergeable === 'CONFLICTING'
export const readyToMerge = (p: PR) =>
    !p.IsDraft && p.ReviewDecision !== 'CHANGES_REQUESTED' && p.ReviewDecision !== 'REVIEW_REQUIRED'
    && (p.CI === 'SUCCESS' || p.CI === '') && p.Mergeable === 'MERGEABLE'
export const jobActive = (j?: Job) => j?.State === 'running' || j?.State === 'waiting'

export type Attention = { tone: Tone, icon: IconName, text: string }

export function attention(p: PR): Attention[] {
    const out: Attention[] = []
    if (ciFailing(p)) out.push({tone: 'bad', icon: 'x', text: 'CI failing'})
    else if (ciRunning(p)) out.push({tone: 'info', icon: 'spinner', text: 'CI running'})
    if (conflict(p)) out.push({tone: 'bad', icon: 'merge', text: 'Conflict'})
    if (p.ReviewDecision === 'APPROVED') out.push({tone: 'ok', icon: 'check', text: 'Approved'})
    else if (p.ReviewDecision === 'CHANGES_REQUESTED') out.push({tone: 'bad', icon: 'thread', text: 'Changes requested'})
    else if (p.ReviewDecision === 'REVIEW_REQUIRED' && !p.IsDraft) out.push({tone: 'neutral', icon: 'eye', text: 'In review'})
    if (p.Unresolved > 0) out.push({tone: 'neutral', icon: 'thread', text: `${p.Unresolved} thread${p.Unresolved > 1 ? 's' : ''}`})
    return out
}

export type NextStep =
    | { kind: 'agent', text: string, waiting: boolean }
    | { kind: 'command', text: string, key: string }
    | { kind: 'wait', text: string }
    | { kind: 'none' }

// nextStep suggests what to do about a PR; a command step runs on Enter.
export function nextStep(p: PR, job: Job | undefined, commands: Command[]): NextStep {
    if (jobActive(job)) return {kind: 'agent', text: `Agent ${job!.Name} · ${job!.Elapsed}`, waiting: job!.State === 'waiting'}
    const cmd = (key: string, text: string): NextStep =>
        commands.some(c => c.Key === key) ? {kind: 'command', key, text} : {kind: 'none'}
    if (!p.Mine) return p.OnMe ? {kind: 'command', key: 'o', text: 'Review it'} : {kind: 'none'}
    if (p.IsDraft) return {kind: 'none'}
    if (conflict(p) || ciFailing(p) || p.UnresolvedNotMine > 0 || p.ReviewDecision === 'CHANGES_REQUESTED')
        return cmd('cc', 'Get it mergeable')
    if (ciRunning(p)) return {kind: 'wait', text: 'Waiting on CI'}
    if (readyToMerge(p)) return cmd('M', 'Merge')
    if (p.ReviewDecision === 'REVIEW_REQUIRED') return {kind: 'wait', text: 'Waiting on review'}
    return {kind: 'none'}
}

export type QuickFilter = { id: string, label: string, dot?: 'ok' | 'bad', match: (p: PR, j?: Job) => boolean }

export const QUICK_FILTERS: QuickFilter[] = [
    {id: 'review', label: 'Waiting for review', match: p => !p.IsDraft && p.ReviewDecision === 'REVIEW_REQUIRED'},
    {id: 'ready', label: 'Ready to merge', dot: 'ok', match: readyToMerge},
    {id: 'draft', label: 'Draft', match: p => p.IsDraft},
    {id: 'ci', label: 'CI failing', dot: 'bad', match: ciFailing},
    {id: 'conflict', label: 'Conflicts', dot: 'bad', match: conflict},
    {id: 'agent', label: 'Agent running', match: (_, j) => jobActive(j)},
]

// statusBadges is a PR's full state for the detail header, passing states included.
export function statusBadges(p: PR): Attention[] {
    const out: Attention[] = []
    if (ciFailing(p)) out.push({tone: 'bad', icon: 'x', text: 'CI failing'})
    else if (ciRunning(p)) out.push({tone: 'info', icon: 'spinner', text: 'CI running'})
    else if (p.CI === 'SUCCESS') out.push({tone: 'ok', icon: 'check', text: 'CI passing'})
    if (p.IsDraft) out.push({tone: 'neutral', icon: 'eye', text: 'Draft'})
    else if (p.ReviewDecision === 'APPROVED') out.push({tone: 'ok', icon: 'check', text: 'Approved'})
    else if (p.ReviewDecision === 'CHANGES_REQUESTED') out.push({tone: 'bad', icon: 'thread', text: 'Changes requested'})
    else if (p.ReviewDecision === 'REVIEW_REQUIRED') out.push({tone: 'neutral', icon: 'eye', text: 'In review'})
    if (p.Unresolved > 0) out.push({tone: 'neutral', icon: 'thread', text: `${p.Unresolved} thread${p.Unresolved > 1 ? 's' : ''}`})
    if (conflict(p)) out.push({tone: 'bad', icon: 'merge', text: 'Conflict'})
    else if (p.Mergeable === 'MERGEABLE') out.push({tone: 'ok', icon: 'check', text: 'No conflicts'})
    return out
}

export type GateState = 'done' | 'fail' | 'pending'
export type Gate = { title: string, state: GateState, detail: string, action?: 'checks' | 'github' | 'ready' }

// beforeMerge lists what stands between a PR and merging.
export function beforeMerge(p: PR): Gate[] {
    const checks = p.Checks ?? []
    const failing = checks.filter(c => c.State === 'fail').length
    const pending = checks.filter(c => c.State === 'pending').length
    const gates: Gate[] = []
    if (p.IsDraft) gates.push({title: 'Ready for review', state: 'pending', detail: 'Still a draft', action: 'ready'})
    gates.push(
        ciFailing(p) ? {title: 'Checks pass', state: 'fail', detail: `${failing || 'Some'} failing on the current head`, action: 'checks'}
            : ciRunning(p) ? {title: 'Checks pass', state: 'pending', detail: `${pending} of ${checks.length} still running`, action: 'checks'}
                : {title: 'Checks pass', state: 'done', detail: checks.length ? `${checks.length} checks on the current head` : 'No checks'},
        p.ReviewDecision === 'APPROVED' ? {title: 'Review', state: 'done', detail: 'Approved'}
            : p.ReviewDecision === 'CHANGES_REQUESTED' ? {title: 'Review', state: 'fail', detail: 'Changes requested', action: 'github'}
                : p.ReviewDecision === 'REVIEW_REQUIRED' ? {title: 'Review', state: 'pending', detail: 'Awaiting an approving review'}
                    : {title: 'Review', state: 'done', detail: 'No review required'},
        p.Unresolved > 0
            ? {title: 'Resolve threads', state: 'pending', detail: `${p.Unresolved} conversation${p.Unresolved > 1 ? 's' : ''} still open`, action: 'github'}
            : {title: 'Resolve threads', state: 'done', detail: 'No open conversations'},
        conflict(p) ? {title: 'Mergeable', state: 'fail', detail: 'Conflicts with the base branch'}
            : p.Mergeable === 'MERGEABLE' ? {title: 'Mergeable', state: 'done', detail: 'No conflicts with the base branch'}
                : {title: 'Mergeable', state: 'pending', detail: 'GitHub is still checking'},
    )
    return gates
}

// needsYou is the agent's open question, if its last run left one.
export function needsYou(j?: Job): string {
    const n = j?.Summary?.NeedsYou?.trim() ?? ''
    return /^(nothing|none|no)\.?$/i.test(n) ? '' : n
}
