import {Fragment} from 'react'
import {useApp} from '../../state/AppState'
import {QUICK_FILTERS} from '../../lib/pr'

const CHIP = 'inline-flex h-7 shrink-0 cursor-pointer items-center gap-1.5 rounded-full border px-2.5 text-xs font-medium'
const DOT: Record<string, string> = {ok: 'bg-dot-ok', bad: 'bg-dot-bad'}

function Chip({active, dot, label, count, onClick}: {
    active: boolean, dot?: string, label: string, count: number, onClick: () => void
}) {
    return (
        <button type="button" className={`${CHIP} ${active ? 'border-foreground bg-foreground text-background' : 'border-border-200 bg-background hover:bg-muted'}`} aria-pressed={active} onClick={onClick}>
            {dot && <span className={`size-1.5 rounded-full ${DOT[dot]}`}/>}{label}<span className={`mono ${active ? 'opacity-75' : 'text-dim'}`}>{count}</span>
        </button>
    )
}

export function FilterBar({quick, setQuick, counts, total}: {
    quick: string, setQuick: (id: string) => void, counts: Record<string, number>, total: number
}) {
    const app = useApp()
    return (
        <nav aria-label="Quick filters" className="flex items-center gap-2 overflow-x-auto border-b border-border-100 px-4 [&::-webkit-scrollbar]:h-0">
            <Chip active={!quick} label="All" count={total} onClick={() => setQuick('')}/>
            {QUICK_FILTERS.map((f, i) => (
                <Fragment key={f.id}>
                    {i === 3 && <span className="mx-1 h-5 w-px shrink-0 bg-border-100"/>}
                    <Chip active={quick === f.id} dot={f.dot} label={f.label} count={counts[f.id] ?? 0}
                          onClick={() => setQuick(quick === f.id ? '' : f.id)}/>
                </Fragment>
            ))}
            {app.showAll && (
                <button type="button" className={`${CHIP} ml-auto border-current bg-background text-warn-fg hover:bg-muted`} onClick={() => app.setShowAll(false)}>Showing hidden & snoozed ✕</button>
            )}
        </nav>
    )
}
