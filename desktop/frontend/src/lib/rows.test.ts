import {describe, expect, it} from 'vitest'
import {countPRs, visibleRows} from './rows'
import {ciFailing} from './pr'
import {group, prRow, pr} from '../test/fixtures'

// Mine › Needs action › {#1 failing, #2 passing}; Mine › Draft › {#3 failing}
const a = pr({Number: 1, CI: 'FAILURE'}), b = pr({Number: 2}), c = pr({Number: 3, CI: 'FAILURE', IsDraft: true})
const rows = [
    group('Mine', 0, 3),
    group('Mine/Needs action', 1, 2), prRow('Mine/Needs action', a), prRow('Mine/Needs action', b),
    group('Mine/Draft', 1, 1), prRow('Mine/Draft', c),
]
const titles = (rs: typeof rows) => rs.map(r => r.PR ? `#${r.PR.Number}` : `${r.Title}(${r.Count})`)

describe('visibleRows', () => {
    it('shows everything when nothing is filtered or collapsed', () => {
        expect(visibleRows(rows, [], {})).toEqual(rows)
    })

    it('keeps matching PRs, recounts their groups and drops groups left empty', () => {
        expect(titles(visibleRows(rows, [], {}, ciFailing))).toEqual(['Mine(2)', 'Needs action(1)', '#1', 'Draft(1)', '#3'])
        expect(titles(visibleRows(rows, [], {}, p => p.Number === 2))).toEqual(['Mine(1)', 'Needs action(1)', '#2'])
    })

    it('folds a collapsed group but keeps its heading', () => {
        expect(titles(visibleRows(rows, ['Mine/Needs action'], {}))).toEqual(['Mine(3)', 'Needs action(2)', 'Draft(1)', '#3'])
        expect(titles(visibleRows(rows, ['Mine'], {}))).toEqual(['Mine(3)'])
    })

    it('does not fold a sibling whose name starts the same', () => {
        const r = [group('Mine/Draft', 1, 1), prRow('Mine/Draft', c), group('Mine/Drafts', 1, 1), prRow('Mine/Drafts', b)]
        expect(titles(visibleRows(r, ['Mine/Draft'], {}))).toEqual(['Draft(1)', 'Drafts(1)', '#2'])
    })

    it('counts against the filter, then collapses', () => {
        expect(titles(visibleRows(rows, ['Mine/Draft'], {}, ciFailing))).toEqual(['Mine(2)', 'Needs action(1)', '#1', 'Draft(1)'])
    })
})

describe('countPRs', () => {
    it('counts a PR shown in two sections once', () => {
        const twice = [...rows, group('Reviews', 0, 1), prRow('Reviews', a, 1)]
        expect(countPRs(twice, {}, ciFailing)).toBe(2)
        expect(countPRs(twice, {}, () => true)).toBe(3)
    })
})
