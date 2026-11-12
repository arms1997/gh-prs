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
