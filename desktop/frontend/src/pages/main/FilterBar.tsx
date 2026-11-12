import {Fragment} from 'react'
import {useApp} from '../../state/AppState'
import {QUICK_FILTERS} from '../../lib/pr'

function Chip({active, dot, label, count, onClick}: {
    active: boolean, dot?: string, label: string, count: number, onClick: () => void
}) {
    return (
        <button type="button" className={'chip' + (active ? ' active' : '')} aria-pressed={active} onClick={onClick}>
            {dot && <span className={'chip-dot ' + dot}/>}{label}<span className="chip-count mono">{count}</span>
        </button>
    )
}

export function FilterBar({quick, setQuick, counts, total}: {
    quick: string, setQuick: (id: string) => void, counts: Record<string, number>, total: number
}) {
    const app = useApp()
    return (
        <nav aria-label="Quick filters" className="filterbar">
            <Chip active={!quick} label="All" count={total} onClick={() => setQuick('')}/>
            {QUICK_FILTERS.map((f, i) => (
                <Fragment key={f.id}>
                    {i === 3 && <span className="filterbar-sep"/>}
                    <Chip active={quick === f.id} dot={f.dot} label={f.label} count={counts[f.id] ?? 0}
                          onClick={() => setQuick(quick === f.id ? '' : f.id)}/>
                </Fragment>
            ))}
            {app.showAll && (
                <button type="button" className="chip warn" onClick={() => app.setShowAll(false)}>Showing hidden & snoozed ✕</button>
            )}
        </nav>
    )
}
