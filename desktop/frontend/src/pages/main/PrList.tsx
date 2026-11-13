import {useEffect, useRef} from 'react'
import {useApp} from '../../state/AppState'
import {nextStep, Row} from '../../lib/pr'
import {AttentionBadges, DiffStat, NextStepLabel, TurnDot} from '../../components/pr/PrBits'
import {Icon} from '../../components/ui/Icon'
import {useWidths} from './useWidths'

// The resizable columns, matching core.ColumnNames; the title takes the rest.
const COLUMNS = [
    {key: 'repo', label: 'Repository'},
    {key: 'diff', label: 'Diff', className: 'narrow:hidden'},
    {key: 'attention', label: 'Needs attention'},
    {key: 'next', label: 'Next step'},
]

const GRID = 'grid grid-cols-[var(--lead,16px)_minmax(120px,1fr)_var(--w-repo,176px)_var(--w-diff,104px)_var(--w-attention,196px)_var(--w-next,152px)] items-center gap-3 narrow:grid-cols-[var(--lead,16px)_minmax(120px,1fr)_var(--w-repo,150px)_var(--w-attention,180px)_var(--w-next,140px)]'
const GROUP: Record<number, string> = {
    0: 'h-10 pt-3 text-lg font-semibold text-foreground',
    1: 'h-[30px] pt-1 text-sm font-medium text-dim',
    2: 'h-7 pt-1 text-xs font-medium text-dim',
}
// Rows keep at least their full width, so a wide details pane covers them instead of squeezing them:
// the deepest indent, the title's minimum, the columns, five gaps and the row padding.
const MIN_ROW = '[&>*]:min-w-[calc(54px+120px+var(--w-repo,176px)+var(--w-diff,104px)+var(--w-attention,196px)+var(--w-next,152px)+60px+40px)]'

export function PrList({rows, cursor, collapsed, onSelect, onToggle}: {
    rows: Row[], cursor: number, collapsed: string[], onSelect: (path: string) => void, onToggle: (path: string) => void
}) {
    const app = useApp()
    const selRef = useRef<HTMLDivElement>(null)
    useEffect(() => { selRef.current?.scrollIntoView({block: 'nearest'}) }, [cursor])

    const {vars, handle} = useWidths()

    return (
        <main aria-label="Pull requests" className={`scroll-list min-h-0 overflow-x-clip overflow-y-auto px-2 pb-4 ${MIN_ROW}`} style={vars}>
            <div className={`${GRID} sticky top-0 z-1 -mx-2 h-8 border-b border-border-100 bg-background pr-6 pl-8 text-xs font-medium text-dim`}>
                <span/><span>Title</span>
                {COLUMNS.map(c => (
                    <span key={c.key} className={`relative flex min-w-0 ${c.className ?? ''}`}>
                        <span {...handle(c.key, c.label)}/>
                        <span className="min-w-0 truncate">{c.label}</span>
                    </span>
                ))}
            </div>
            {rows.map((r, i) => {
                const sel = i === cursor
                const ref = sel ? selRef : undefined
                if (!r.PR) {
                    const closed = collapsed.includes(r.Path)
                    return (
                        <div key={r.Path} ref={ref} // A narrow list (a wide details pane) clips headings at its edge instead of squeezing or wrapping them.
                             className={`flex cursor-pointer items-center gap-2 overflow-hidden rounded-md whitespace-nowrap [&>*]:shrink-0 ${GROUP[Math.min(r.Depth, 2)]} ${sel ? 'bg-sel-bg shadow-sel' : ''}`}
                             style={{paddingLeft: 8 + r.Depth * 18}}
                             onClick={() => { onSelect(r.Path); onToggle(r.Path) }}>
                            <Icon name={closed ? 'chevronRight' : 'chevronDown'} size={r.Depth ? 12 : 14}/>
                            <span>{r.Title}</span><span className="mono text-xs font-medium text-dim">{r.Count}</span>
                        </div>
                    )
                }
                const p = r.PR
                const step = nextStep(p, app.state?.Jobs[p.Key], app.state?.Commands ?? [])
                return (
                    // The dot lines up with its group heading's text: headings indent 18px a level.
                    <div key={r.Path} ref={ref} className={`${GRID} mt-0.5 h-9 cursor-default rounded-md pr-4 ${sel ? 'bg-sel-bg shadow-sel' : 'hover:bg-muted'} ${p.Hidden || p.Snoozed ? 'opacity-55' : ''}`}
                         style={{'--lead': `${18 + 18 * r.Depth}px`} as React.CSSProperties}
                         onClick={() => onSelect(r.Path)} onDoubleClick={() => app.open(p.URL)}>
                        <TurnDot pr={p}/>
                        <span className="flex min-w-0 items-center gap-1.5" style={{paddingLeft: r.Stack * 14}}>
                            {r.Stack > 0 && <Icon name="stack" size={14} className="text-faint"/>}
                            {p.Pinned && <Icon name="pin" className="warn" />}
                            {p.Snoozed && <Icon name="moon" className="text-faint"/>}
                            {p.Hidden && <Icon name="eyeOff" className="text-faint"/>}
                            <span className="min-w-0 truncate">{p.Title}</span>
                        </span>
                        <span className="mono min-w-0 truncate text-xs text-dim">{p.Repo.split('/')[1]}#{p.Number}</span>
                        <span className="narrow:hidden"><DiffStat pr={p}/></span>
                        <AttentionBadges pr={p}/>
                        <NextStepLabel step={step} showKey={sel}/>
                    </div>
                )
            })}
            {rows.length === 0 && <div className="dim p-6">{app.state ? 'No PRs match.' : 'Loading…'}</div>}
        </main>
    )
}
