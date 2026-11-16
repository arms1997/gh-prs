import {useEffect, useRef} from 'react'
import {useApp} from '../../state/AppState'
import {nextStep, Row} from '../../lib/pr'
import {AgentLabel, AttentionBadges, DiffStat, TurnDot} from '../../components/pr/PrBits'
import {Icon} from '../../components/ui/Icon'
import {useWidths} from './useWidths'

// The header, with the resizable columns (matching core.ColumnNames) at their keys; title and turn take the rest.
const COLUMNS = [
    {key: 'repo', label: 'Repository'},
    {key: 'diff', label: 'Lines', className: 'narrow:hidden'},
    {key: 'attention', label: 'Needs attention'},
    {label: 'Whose turn and why'},
    {key: 'agent', label: 'Agent'},
]

// Resizable columns are at most their width, so widths that add up to more than the window give way
// instead of pushing the last columns off-screen. The lead is the same on every row and the header, so
// they share the free space alike and stay lined up; the dot moves within it instead.
const GRID = 'grid grid-cols-[54px_minmax(240px,1.4fr)_minmax(40px,var(--w-repo,184px))_minmax(40px,var(--w-diff,104px))_minmax(40px,var(--w-attention,220px))_minmax(200px,1fr)_minmax(40px,var(--w-agent,168px))] items-center gap-3 narrow:grid-cols-[54px_minmax(160px,1.4fr)_minmax(40px,var(--w-repo,150px))_minmax(40px,var(--w-attention,180px))_minmax(160px,1fr)_minmax(40px,var(--w-agent,150px))]'
const GROUP: Record<number, string> = {
    0: 'h-10 pt-3 text-lg font-semibold text-foreground',
    1: 'h-[30px] pt-1 text-sm font-medium text-dim',
    2: 'h-7 pt-1 text-xs font-medium text-dim',
}

export function PrList({rows, cursor, collapsed, onSelect, onToggle, onOpen}: {
    rows: Row[], cursor: number, collapsed: string[], onSelect: (path: string) => void, onToggle: (path: string) => void,
    onOpen: (path: string) => void
}) {
    const app = useApp()
    const selRef = useRef<HTMLDivElement>(null)
    useEffect(() => { selRef.current?.scrollIntoView({block: 'nearest'}) }, [cursor])

    const {vars, handle} = useWidths()

    return (
        <main aria-label="Pull requests" className="scroll-list min-h-0 overflow-x-clip overflow-y-auto px-2 pb-4" style={vars}>
            <div className="sticky top-0 z-1 -mx-2 border-b border-border-100 bg-background px-2">
                <div className={`${GRID} h-8 pr-4 text-xs font-medium text-dim`}>
                    <span/><span>Title</span>
                    {COLUMNS.map(c => (
                        <span key={c.label} className={`relative flex min-w-0 ${c.className ?? ''}`}>
                            {c.key && <span {...handle(c.key, c.label)}/>}
                            <span className="min-w-0 truncate">{c.label}</span>
                        </span>
                    ))}
                </div>
            </div>
            {rows.map((r, i) => {
                const sel = i === cursor
                const ref = sel ? selRef : undefined
                if (!r.PR) {
                    const closed = collapsed.includes(r.Path)
                    return (
                        <div key={r.Path} ref={ref} // A narrow window clips headings at its edge instead of squeezing or wrapping them.
                             className={`flex cursor-pointer items-center gap-2 overflow-hidden rounded-md whitespace-nowrap [&>*]:shrink-0 ${GROUP[Math.min(r.Depth, 2)]} ${sel ? 'bg-sel-bg shadow-sel' : ''}`}
                             style={{paddingLeft: 8 + r.Depth * 18}}
                             onClick={() => { onSelect(r.Path); onToggle(r.Path) }}>
                            <Icon name={closed ? 'chevronRight' : 'chevronDown'} size={r.Depth ? 12 : 14}/>
                            <span>{r.Title}</span><span className="font-mono text-xs font-medium text-dim">{r.Count}</span>
                        </div>
                    )
                }
                const p = r.PR
                const step = nextStep(p, app.state?.Jobs[p.Key], app.state?.Commands ?? [])
                return (
                    <div key={r.Path} ref={ref} className={`${GRID} mt-0.5 h-9 cursor-default rounded-md pr-4 ${sel ? 'bg-sel-bg shadow-sel' : 'hover:bg-muted'} ${p.Hidden || p.Snoozed ? 'opacity-55' : ''}`}
                         onClick={() => onSelect(r.Path)} onDoubleClick={() => onOpen(r.Path)}>
                        {/* The dot lines up with its group heading's text: headings indent 18px a level. */}
                        <TurnDot pr={p} style={{marginRight: 18 * (2 - Math.min(r.Depth, 2))}}/>
                        <span className="flex min-w-0 items-center gap-1.5" style={{paddingLeft: r.Stack * 14}}>
                            {r.Stack > 0 && <Icon name="stack" size={14} className="text-faint"/>}
                            {p.Pinned && <Icon name="pin" className="text-warn-fg" />}
                            {p.Snoozed && <Icon name="moon" className="text-faint"/>}
                            {p.Hidden && <Icon name="eyeOff" className="text-faint"/>}
                            <span className="min-w-0 truncate">{p.Title}</span>
                        </span>
                        <span className="font-mono min-w-0 truncate text-xs text-dim">{p.Repo.split('/')[1]}#{p.Number}</span>
                        <span className="narrow:hidden"><DiffStat pr={p}/></span>
                        <AttentionBadges pr={p}/>
                        <span className={`min-w-0 truncate text-sm ${p.OnMe ? 'text-foreground' : 'text-dim'}`} title={p.TurnWhy}>{p.TurnWhy}</span>
                        <AgentLabel job={app.state?.Jobs[p.Key]} step={step} sel={sel}/>
                    </div>
                )
            })}
            {rows.length === 0 && <div className="text-dim p-6">{app.state ? 'No PRs match.' : 'Loading…'}</div>}
        </main>
    )
}
