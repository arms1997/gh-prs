import {Job, PR, Row} from './pr'

const under = (path: string, group: string) => path.startsWith(group + '/')

// visibleRows applies a quick filter (dropping groups left empty, recounting the rest), then
// folds collapsed groups. Rows come from the backend with every group expanded.
export function visibleRows(rows: Row[], collapsed: string[], jobs: Record<string, Job>,
                            match?: (p: PR, j?: Job) => boolean): Row[] {
    let out = rows
    if (match) {
        const kept = rows.filter(r => r.PR && match(r.PR, jobs[r.PR.Key]))
        out = rows.flatMap(r => {
            if (r.PR) return kept.includes(r) ? [r] : []
            const count = kept.filter(k => under(k.Path, r.Path)).length
            return count ? [{...r, Count: count} as Row] : []
        })
    }
    return out.filter(r => !collapsed.some(c => under(r.Path, c)))
}

// countPRs counts distinct PRs matching, since one PR can sit in several sections.
export function countPRs(rows: Row[], jobs: Record<string, Job>, match: (p: PR, j?: Job) => boolean): number {
    return new Set(rows.filter(r => r.PR && match(r.PR, jobs[r.PR.Key])).map(r => r.PR!.Key)).size
}
