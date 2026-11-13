// The daily brief: what's worth doing first across your PRs, from what the app already knows.
import type {DetailTabName} from '../state/AppState'
import {ciFailing, ciRunning, conflict, Job, jobActive, needsYou, PR, readyToMerge, Row} from './pr'

export type BriefKind = 'answer' | 'review' | 'merge' | 'unblock'
export type BriefItem = { pr: PR, kind: BriefKind, why: string, tab: DetailTabName }
export type Waiting = { others: number, ci: number, agent: number, draft: number }

// Ranked by who's blocked: Claude waiting on your answer, people waiting on your review, then work
// that's one step from done (merges), then your PRs that are stuck. List order breaks ties.
const RANK: Record<BriefKind, number> = {answer: 0, review: 1, merge: 2, unblock: 3}

export function dailyBrief(rows: Row[], jobs: Record<string, Job>): { items: BriefItem[], waiting: Waiting } {
    const items: BriefItem[] = []
    const waiting: Waiting = {others: 0, ci: 0, agent: 0, draft: 0}
    const seen = new Set<string>()
    for (const p of rows.map(r => r.PR).filter(p => p != null)) {
        if (seen.has(p.Key) || p.Hidden || p.Snoozed) continue
        seen.add(p.Key)
        const job = jobs[p.Key]
        const question = needsYou(job)
        if (question && job?.State !== 'stopped') {
            items.push({pr: p, kind: 'answer', why: question, tab: 'claude'})
        } else if (!p.Mine) {
            if (p.OnMe) items.push({pr: p, kind: 'review', why: `${p.Author} is waiting on your review`, tab: 'overview'})
        } else if (jobActive(job)) waiting.agent++
        else if (p.IsDraft) waiting.draft++
        else {
            const blockers = [
                conflict(p) && 'conflicts with its base',
                ciFailing(p) && 'CI failing',
                p.ReviewDecision === 'CHANGES_REQUESTED' && 'changes requested',
                p.UnresolvedNotMine > 0 && `${p.UnresolvedNotMine} thread${p.UnresolvedNotMine > 1 ? 's' : ''} waiting on you`,
            ].filter(b => b !== false)
            if (blockers.length) items.push({pr: p, kind: 'unblock', why: blockers.join(' · '), tab: 'overview'})
            else if (readyToMerge(p)) items.push({pr: p, kind: 'merge', why: 'Approved, checks pass, no conflicts', tab: 'overview'})
            else if (ciRunning(p)) waiting.ci++
            else waiting.others++
        }
    }
    items.sort((a, b) => RANK[a.kind] - RANK[b.kind])
    return {items, waiting}
}
