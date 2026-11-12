import {useEffect, useRef} from 'react'
import {useApp} from '../../state/AppState'
import {nextStep, Row} from '../../lib/pr'
import {AttentionBadges, DiffStat, NextStepLabel, TurnDot} from '../../components/pr/PrBits'
import {Icon} from '../../components/ui/Icon'

export function PrList({rows, cursor, collapsed, onSelect, onToggle}: {
    rows: Row[], cursor: number, collapsed: string[], onSelect: (path: string) => void, onToggle: (path: string) => void
}) {
    const app = useApp()
    const selRef = useRef<HTMLDivElement>(null)
    useEffect(() => { selRef.current?.scrollIntoView({block: 'nearest'}) }, [cursor])

    return (
        <main aria-label="Pull requests" className="prlist scroll-list">
            <div className="pr-grid prlist-head">
                <span/><span>Title</span><span>Repository</span><span className="col-diff">Diff</span><span>Needs attention</span><span>Next step</span>
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
                    <div key={r.Path} ref={ref} className={'pr-grid pr-row' + (sel ? ' sel' : '') + (p.Hidden || p.Snoozed ? ' faded' : '')}
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
