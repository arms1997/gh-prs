import {describe, expect, it} from 'vitest'
import {agentCommands, attention, beforeMerge, nextStep, needsYou, QUICK_FILTERS, readyToMerge, statusBadges} from './pr'
import {cmd, COMMANDS, job, pr} from '../test/fixtures'

const texts = (xs: { text: string }[]) => xs.map(x => x.text)

describe('attention', () => {
    it('shows what blocks the PR, not what passes', () => {
        expect(texts(attention(pr({CI: 'FAILURE', Mergeable: 'CONFLICTING', Unresolved: 2})))).toEqual(['CI failing', 'Conflict', 'In review', '2 threads'])
        expect(texts(attention(pr({CI: 'SUCCESS', ReviewDecision: 'APPROVED'})))).toEqual(['Approved'])
        expect(texts(attention(pr({CI: 'PENDING', IsDraft: true, Unresolved: 1})))).toEqual(['CI running', '1 thread'])
    })
    it('the detail header also shows passing states', () => {
        expect(texts(statusBadges(pr({ReviewDecision: 'APPROVED'})))).toEqual(['CI passing', 'Approved', 'No conflicts'])
    })
})

describe('readyToMerge', () => {
    it('needs approval (or none required), green or no CI, no conflict, not a draft', () => {
        expect(readyToMerge(pr({ReviewDecision: 'APPROVED'}))).toBe(true)
        expect(readyToMerge(pr({ReviewDecision: '', CI: ''}))).toBe(true)
        expect(readyToMerge(pr({ReviewDecision: 'REVIEW_REQUIRED'}))).toBe(false)
        expect(readyToMerge(pr({ReviewDecision: 'APPROVED', CI: 'PENDING'}))).toBe(false)
        expect(readyToMerge(pr({ReviewDecision: 'APPROVED', Mergeable: 'UNKNOWN'}))).toBe(false)
        expect(readyToMerge(pr({ReviewDecision: 'APPROVED', IsDraft: true}))).toBe(false)
    })
})

describe('nextStep', () => {
    const step = (p = pr(), j?: ReturnType<typeof job>) => nextStep(p, j, COMMANDS)
    it('a running agent wins', () => {
        expect(step(pr({Mergeable: 'CONFLICTING'}), job({State: 'running', Name: 'manage', Elapsed: '3m'}))).toEqual({kind: 'agent', text: 'Agent manage · 3m', waiting: false})
        expect(step(pr(), job({State: 'waiting'}))).toMatchObject({kind: 'agent', waiting: true})
    })
    it('a finished agent no longer counts', () => {
        expect(step(pr({Mergeable: 'CONFLICTING'}), job({State: 'done'}))).toMatchObject({kind: 'command', key: 'cc'})
    })
    it('suggests fixing what blocks my PR', () => {
        for (const p of [pr({Mergeable: 'CONFLICTING'}), pr({CI: 'ERROR'}), pr({UnresolvedNotMine: 1}), pr({ReviewDecision: 'CHANGES_REQUESTED'})]) {
            expect(step(p)).toEqual({kind: 'command', key: 'cc', text: 'Get it mergeable'})
        }
    })
    it('waits, merges, or does nothing', () => {
        expect(step(pr({CI: 'PENDING'}))).toEqual({kind: 'wait', text: 'Waiting on CI'})
        expect(step(pr({ReviewDecision: 'APPROVED'}))).toEqual({kind: 'command', key: 'M', text: 'Merge'})
        expect(step(pr())).toEqual({kind: 'wait', text: 'Waiting on review'})
        expect(step(pr({IsDraft: true, Mergeable: 'CONFLICTING'}))).toEqual({kind: 'none'})
    })
    it("someone else's PR: review it when it's my turn", () => {
        expect(step(pr({Mine: false, OnMe: true}))).toEqual({kind: 'command', key: 'o', text: 'Review it'})
        expect(step(pr({Mine: false, OnMe: false, Mergeable: 'CONFLICTING'}))).toEqual({kind: 'none'})
    })
    it('never suggests a command you have not configured', () => {
        expect(nextStep(pr({Mergeable: 'CONFLICTING'}), undefined, [])).toEqual({kind: 'none'})
    })
})

describe('beforeMerge', () => {
    const gate = (p: ReturnType<typeof pr>, title: string) => beforeMerge(p).find(g => g.title === title)
    it('lists checks, review, threads and mergeable, plus ready-for-review on drafts', () => {
        expect(beforeMerge(pr()).map(g => g.title)).toEqual(['Checks pass', 'Review', 'Resolve threads', 'Mergeable'])
        expect(beforeMerge(pr({IsDraft: true}))[0]).toMatchObject({title: 'Ready for review', action: 'ready'})
    })
    it('counts failing and running checks', () => {
        const checks = [{Name: 'a', State: 'fail', URL: ''}, {Name: 'b', State: 'pending', URL: ''}, {Name: 'c', State: 'pass', URL: ''}]
        expect(gate(pr({CI: 'FAILURE', Checks: checks as never}), 'Checks pass')).toMatchObject({state: 'fail', detail: '1 failing on the current head', action: 'checks'})
        expect(gate(pr({CI: 'PENDING', Checks: checks as never}), 'Checks pass')).toMatchObject({state: 'pending', detail: '1 of 3 still running'})
        expect(gate(pr({CI: ''}), 'Checks pass')).toMatchObject({state: 'done', detail: 'No checks'})
    })
    it('reads review, threads and conflicts', () => {
        expect(gate(pr({ReviewDecision: 'CHANGES_REQUESTED'}), 'Review')?.state).toBe('fail')
        expect(gate(pr({ReviewDecision: ''}), 'Review')).toMatchObject({state: 'done', detail: 'No review required'})
        expect(gate(pr({Unresolved: 2}), 'Resolve threads')).toMatchObject({state: 'pending', detail: '2 conversations still open'})
        expect(gate(pr({Mergeable: 'CONFLICTING'}), 'Mergeable')?.state).toBe('fail')
        expect(gate(pr({Mergeable: 'UNKNOWN'}), 'Mergeable')?.state).toBe('pending')
    })
})

describe('needsYou', () => {
    it('ignores "nothing" and its variants', () => {
        for (const n of ['nothing', 'Nothing.', 'none', 'No', '  ']) expect(needsYou(job({Summary: {NeedsYou: n} as never}))).toBe('')
        expect(needsYou(job({Summary: {NeedsYou: 'Rebase or not?'} as never}))).toBe('Rebase or not?')
        expect(needsYou(undefined)).toBe('')
    })
})

describe('agentCommands', () => {
    it('finds commands by what they do, whatever their keys', () => {
        const c = agentCommands(COMMANDS)
        expect([c.ask?.Key, c.reply?.Key, c.stop?.Key, c.takeOver?.Key]).toEqual(['ci', 'cy', 'cx', 'ct'])
        const renamed = agentCommands([cmd({Key: 'zz', Background: true, Session: true, Input: 'Reply'})])
        expect(renamed.reply?.Key).toBe('zz')
        expect(renamed.ask).toBeUndefined()
    })
})

describe('quick filters', () => {
    it('agent running counts waiting loops too', () => {
        const f = QUICK_FILTERS.find(f => f.id === 'agent')!
        expect(f.match(pr(), job({State: 'waiting'}))).toBe(true)
        expect(f.match(pr(), job({State: 'stopped'}))).toBe(false)
        expect(f.match(pr(), undefined)).toBe(false)
    })
})
