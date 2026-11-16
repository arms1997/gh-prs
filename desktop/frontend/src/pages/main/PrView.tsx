import {PR} from '../../lib/pr'
import {Button} from '../../components/ui/Button'
import {Icon} from '../../components/ui/Icon'
import {Kbd} from '../../components/ui/Kbd'
import {DetailPane, DetailTab} from '../../features/detail/DetailPane'

// One PR on the whole page, opened from the list (e); j/k step through the list's PRs, esc goes back.
export function PrView({pr, stackedOn, groups, pos, onBack, onStep, tab, setTab}: {
    pr: PR, stackedOn: number, groups: string[], pos: { n: number, total: number }, onBack: () => void, onStep: (d: 1 | -1) => void,
    tab: DetailTab, setTab: (t: DetailTab) => void
}) {
    return (
        <div className="grid min-h-0 grid-rows-[48px_minmax(0,1fr)]">
            <nav aria-label="Pull request" className="flex items-center gap-3 border-b border-border-100 px-4 text-sm">
                <Button size="sm" onClick={onBack}><Icon name="chevronLeft"/>All PRs<Kbd keys="esc"/></Button>
                <span className="min-w-0 truncate text-dim">{groups.join(' › ')}</span>
                <span className="ml-auto inline-flex items-center gap-2">
                    <Button size="sm" onClick={() => onStep(-1)} disabled={pos.n <= 1}><Kbd keys="k"/>Previous</Button>
                    <span className="font-mono text-xs text-dim">{pos.n} of {pos.total}</span>
                    <Button size="sm" onClick={() => onStep(1)} disabled={pos.n >= pos.total}>Next<Kbd keys="j"/></Button>
                </span>
            </nav>
            <DetailPane pr={pr} stackedOn={stackedOn} tab={tab} setTab={setTab}/>
        </div>
    )
}
