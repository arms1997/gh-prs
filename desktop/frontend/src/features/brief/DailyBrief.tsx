import {markdownInline} from '../../lib/markdown'
import {useApp} from '../../state/AppState'
import {BriefItem, BriefKind, dailyBrief} from '../../lib/brief'
import {Modal} from '../../components/ui/Modal'
import {Icon, IconName} from '../../components/ui/Icon'

const KIND: Record<BriefKind, { label: string, icon: IconName }> = {
    answer: {label: 'Answer Claude', icon: 'question'},
    review: {label: 'Review', icon: 'eye'},
    merge: {label: 'Merge', icon: 'merge'},
    unblock: {label: 'Unblock', icon: 'x'},
}

const KIND_COLOR: Record<BriefKind, string> = {answer: 'text-warn-fg', review: 'text-link', merge: 'text-ok-fg', unblock: 'text-bad-fg'}

// The daily brief: what to do first across your PRs. Picking one shows it in the main view.
export function DailyBrief({onClose}: { onClose: () => void }) {
    const app = useApp()
    const {items, waiting} = dailyBrief(app.state?.Rows ?? [], app.state?.Jobs ?? {})
    const pick = (i: BriefItem) => {
        app.setFocus({prKey: i.pr.Key, tab: i.tab})
        onClose()
    }
    const counts = [
        [waiting.others, 'waiting on reviewers'], [waiting.ci, 'waiting on CI'],
        [waiting.agent, 'with an agent working'], [waiting.draft, 'in draft'],
    ].filter(([n]) => n) as [number, string][]
    return (
        <Modal label="Daily brief" onClose={onClose} top>
            <div className="flex w-[640px] max-w-[calc(100vw-32px)] flex-col">
                <header className="flex items-baseline gap-3 border-b border-border-100 px-5 pt-4 pb-3">
                    <h2 className="m-0 text-lg font-semibold">Daily brief</h2>
                    <span className="dim small">{items.length ? `${items.length} worth doing, most blocking first` : 'Nothing needs you right now'}</span>
                </header>
                <ol className="m-0 max-h-[60vh] list-none overflow-y-auto px-2 py-1.5">
                    {items.map((i, n) => (
                        <li key={i.pr.Key}>
                            <button type="button" className="grid w-full cursor-pointer grid-cols-[136px_minmax(0,1fr)] gap-x-3 gap-y-0.5 rounded-md border-0 bg-transparent px-3 py-2.5 text-left text-inherit hover:bg-muted focus-visible:bg-muted focus-visible:shadow-sel focus-visible:outline-none" onClick={() => pick(i)} autoFocus={n === 0}>
                                <span className={`row-span-2 inline-flex items-center gap-1.5 text-sm font-medium ${KIND_COLOR[i.kind]}`}><Icon name={KIND[i.kind].icon} size={14}/> {KIND[i.kind].label}</span>
                                <span className="flex min-w-0 items-baseline gap-2 text-base font-medium">
                                    <span className="ellipsis">{i.pr.Title}</span>
                                    <span className="mono small dim shrink-0 whitespace-nowrap">{i.pr.Repo.split('/')[1]}#{i.pr.Number}</span>
                                </span>
                                <span className="small dim [overflow-wrap:anywhere]" dangerouslySetInnerHTML={{__html: markdownInline(i.why)}}/>
                            </button>
                        </li>
                    ))}
                </ol>
                {counts.length > 0 && <footer className="small dim border-t border-border-100 px-5 pt-2.5 pb-3.5">Also open: {counts.map(([n, t]) => `${n} ${t}`).join(' · ')}</footer>}
            </div>
        </Modal>
    )
}
