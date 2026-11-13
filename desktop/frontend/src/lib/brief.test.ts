import {describe, expect, it} from 'vitest'
import {dailyBrief} from './brief'
import {group, job, pr, prRow} from '../test/fixtures'

// Go sends heading rows with PR: null.
const rows = (...prs: ReturnType<typeof pr>[]) => [{...group('g', 1, prs.length), PR: null} as never, ...prs.map(p => prRow('g', p))]
const asked = (Number: number) => job({PRKey: `o/r#${Number}`, State: 'waiting', Summary: {NeedsYou: 'Rebase onto main?'} as never})

describe('dailyBrief', () => {
    it('ranks an agent question, then reviews, merges, and stuck PRs', () => {
        const {items} = dailyBrief(rows(
            pr({Number: 1, CI: 'FAILURE'}),
            pr({Number: 2, ReviewDecision: 'APPROVED'}),
            pr({Number: 3, Mine: false, OnMe: true, Author: 'kim'}),
            pr({Number: 4}),
        ), {'o/r#4': asked(4)})
        expect(items.map(i => [i.pr.Number, i.kind])).toEqual([[4, 'answer'], [3, 'review'], [2, 'merge'], [1, 'unblock']])
        expect(items[0].tab).toBe('claude')
        expect(items[1].why).toBe('kim is waiting on your review')
    })

    it('names every blocker on a stuck PR', () => {
        const {items} = dailyBrief(rows(pr({Mergeable: 'CONFLICTING', CI: 'ERROR', ReviewDecision: 'CHANGES_REQUESTED', UnresolvedNotMine: 2})), {})
        expect(items[0].why).toBe('conflicts with its base · CI failing · changes requested · 2 threads waiting on you')
    })

    it('counts what waits on others, and leaves out hidden, snoozed, duplicate and not-your-turn PRs', () => {
        const p = pr({Number: 9})
        const {items, waiting} = dailyBrief([...rows(
            p,
            pr({Number: 5, CI: 'PENDING'}),
            pr({Number: 6, IsDraft: true}),
            pr({Number: 7, CI: 'FAILURE'}),
            pr({Number: 8, Hidden: true, CI: 'FAILURE'}),
            pr({Number: 10, Mine: false, OnMe: false}),
        ), prRow('other', p)], {'o/r#7': job({PRKey: 'o/r#7'})})
        expect(items).toEqual([])
        expect(waiting).toEqual({others: 1, ci: 1, agent: 1, draft: 1})
    })

    it("ignores a stopped agent's question", () => {
        const {items} = dailyBrief(rows(pr({Number: 4, ReviewDecision: 'APPROVED'})), {'o/r#4': job({...asked(4), State: 'stopped'})})
        expect(items.map(i => i.kind)).toEqual(['merge'])
    })
})
