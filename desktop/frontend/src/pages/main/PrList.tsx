import {useEffect, useRef, useState} from 'react'
import {useApp} from '../../state/AppState'
import {nextStep, Row} from '../../lib/pr'
import {AttentionBadges, DiffStat, NextStepLabel, TurnDot} from '../../components/pr/PrBits'
import {Icon} from '../../components/ui/Icon'

// The resizable columns, matching core.ColumnNames; the title takes the rest.
const COLUMNS = [
    {key: 'repo', label: 'Repository', width: 176},
    {key: 'diff', label: 'Diff', width: 104, className: 'col-diff'},
    {key: 'attention', label: 'Needs attention', width: 196},
    {key: 'next', label: 'Next step', width: 152},
]
const clamp = (w: number) => Math.round(Math.min(800, Math.max(40, w)))

export function PrList({rows, cursor, collapsed, onSelect, onToggle}: {
    rows: Row[], cursor: number, collapsed: string[], onSelect: (path: string) => void, onToggle: (path: string) => void
}) {
    const app = useApp()
    const selRef = useRef<HTMLDivElement>(null)
    useEffect(() => { selRef.current?.scrollIntoView({block: 'nearest'}) }, [cursor])

    // Widths you set (config.yml), overridden while a drag is under way.
    const [dragging, setDragging] = useState<Record<string, number>>()
    const saved = dragging ?? app.state?.Columns ?? {}
    const width = (key: string) => saved[key] ?? COLUMNS.find(c => c.key === key)!.width
    const vars = Object.fromEntries(COLUMNS.map(c => [`--w-${c.key}`, `${width(c.key)}px`]))
    const save = (cols: Record<string, number>) => app.saveColumns(cols).finally(() => setDragging(undefined))
    const reset = (key: string) => { const {[key]: _, ...rest} = saved; save(rest) }

    // A column's left edge moves with the pointer: the column grows as you drag left, the title shrinks.
    const drag = (key: string, e: React.PointerEvent<HTMLElement>) => {
        const el = e.currentTarget, x0 = e.clientX, w0 = width(key)
        let cols = saved
        el.setPointerCapture(e.pointerId)
        el.onpointermove = ev => setDragging(cols = {...saved, [key]: clamp(w0 - (ev.clientX - x0))})
        el.onpointerup = () => {
            el.onpointermove = el.onpointerup = null
            if (cols !== saved) save(cols)
        }
    }
    const nudge = (key: string, e: React.KeyboardEvent) => {
        const d = {ArrowLeft: 8, ArrowRight: -8}[e.key]
        if (d) { e.preventDefault(); save({...saved, [key]: clamp(width(key) + d)}) }
    }

    return (
        <main aria-label="Pull requests" className="prlist scroll-list" style={vars as React.CSSProperties}>
            <div className="pr-grid prlist-head">
                <span/><span>Title</span>
                {COLUMNS.map(c => (
                    <span key={c.key} className={'col-head ' + (c.className ?? '')}>
                        <span role="separator" aria-orientation="vertical" aria-label={`Resize ${c.label}`} tabIndex={0}
                              aria-valuenow={width(c.key)} title="Drag to resize · double-click to reset"
                              className={'col-resize' + (dragging?.[c.key] !== undefined ? ' active' : '')}
                              onPointerDown={e => drag(c.key, e)} onDoubleClick={() => reset(c.key)}
                              onKeyDown={e => nudge(c.key, e)}/>
                        <span className="ellipsis">{c.label}</span>
                    </span>
                ))}
            </div>
            {rows.map((r, i) => {
                const sel = i === cursor
                const ref = sel ? selRef : undefined
                if (!r.PR) {
                    const closed = collapsed.includes(r.Path)
                    return (
                        <div key={r.Path} ref={ref} className={`group depth${Math.min(r.Depth, 2)}` + (sel ? ' sel' : '')}
                             style={{paddingLeft: 8 + r.Depth * 18}}
                             onClick={() => { onSelect(r.Path); onToggle(r.Path) }}>
                            <Icon name={closed ? 'chevronRight' : 'chevronDown'} size={r.Depth ? 12 : 14}/>
                            <span>{r.Title}</span><span className="group-count mono">{r.Count}</span>
                        </div>
                    )
                }
                const p = r.PR
                const step = nextStep(p, app.state?.Jobs[p.Key], app.state?.Commands ?? [])
                return (
                    // The dot lines up with its group heading's text: headings indent 18px a level.
                    <div key={r.Path} ref={ref} className={'pr-grid pr-row' + (sel ? ' sel' : '') + (p.Hidden || p.Snoozed ? ' faded' : '')}
                         style={{'--lead': `${18 + 18 * r.Depth}px`} as React.CSSProperties}
                         onClick={() => onSelect(r.Path)} onDoubleClick={() => app.open(p.URL)}>
                        <TurnDot pr={p}/>
                        <span className="pr-title" style={{paddingLeft: r.Stack * 14}}>
                            {r.Stack > 0 && <Icon name="stack" size={14} className="faint"/>}
                            {p.Pinned && <Icon name="pin" className="warn" />}
                            {p.Snoozed && <Icon name="moon" className="faint"/>}
                            {p.Hidden && <Icon name="eyeOff" className="faint"/>}
                            <span className="ellipsis">{p.Title}</span>
                        </span>
                        <span className="pr-repo mono ellipsis">{p.Repo.split('/')[1]}#{p.Number}</span>
                        <span className="col-diff"><DiffStat pr={p}/></span>
                        <AttentionBadges pr={p}/>
                        <NextStepLabel step={step} showKey={sel}/>
                    </div>
                )
            })}
            {rows.length === 0 && <div className="prlist-empty dim">{app.state ? 'No PRs match.' : 'Loading…'}</div>}
        </main>
    )
}
