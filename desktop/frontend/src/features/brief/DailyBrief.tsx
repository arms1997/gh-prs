import {markdownInline} from '../../lib/markdown'
import {useApp} from '../../state/AppState'
import {BriefItem, BriefKind, dailyBrief} from '../../lib/brief'
import {Modal} from '../../components/ui/Modal'
import {Icon, IconName} from '../../components/ui/Icon'
import './brief.css'

const KIND: Record<BriefKind, { label: string, icon: IconName }> = {
    answer: {label: 'Answer Claude', icon: 'question'},
    review: {label: 'Review', icon: 'eye'},
    merge: {label: 'Merge', icon: 'merge'},
    unblock: {label: 'Unblock', icon: 'x'},
}

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
            <div className="daily">
                <header className="daily-head">
                    <h2>Daily brief</h2>
                    <span className="dim small">{items.length ? `${items.length} worth doing, most blocking first` : 'Nothing needs you right now'}</span>
                </header>
                <ol className="daily-list">
                    {items.map((i, n) => (
                        <li key={i.pr.Key}>
                            <button type="button" className="daily-item" onClick={() => pick(i)} autoFocus={n === 0}>
                                <span className={`daily-kind ${i.kind}`}><Icon name={KIND[i.kind].icon} size={14}/> {KIND[i.kind].label}</span>
                                <span className="daily-pr">
                                    <span className="ellipsis">{i.pr.Title}</span>
                                    <span className="mono small dim">{i.pr.Repo.split('/')[1]}#{i.pr.Number}</span>
                                </span>
                                <span className="daily-why small dim" dangerouslySetInnerHTML={{__html: markdownInline(i.why)}}/>
                            </button>
                        </li>
                    ))}
                </ol>
                {counts.length > 0 && <footer className="daily-foot small dim">Also open: {counts.map(([n, t]) => `${n} ${t}`).join(' · ')}</footer>}
            </div>
        </Modal>
    )
}
