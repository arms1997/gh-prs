import {useMemo} from 'react'
import {useApp} from '../../state/AppState'
import {markdown} from '../../lib/markdown'
import {PR} from '../../lib/pr'
import {AttentionBadges, DiffStat} from '../../components/pr/PrBits'
import {Badge} from '../../components/ui/Badge'
import {Button} from '../../components/ui/Button'
import {Icon} from '../../components/ui/Icon'
import {Kbd} from '../../components/ui/Kbd'
import {AgentBox} from './AgentBox'
import './detail.css'

export function DetailPane({pr}: { pr?: PR }) {
    const app = useApp()
    // Links in the pane, including ones in PR descriptions, open in the browser.
    const onLink = (e: React.MouseEvent) => {
        const a = (e.target as HTMLElement).closest('a')
        if (a) {
            e.preventDefault()
            app.open(a.getAttribute('href') ?? '')
        }
    }
    return (
        <aside aria-label="Pull request details" className="detail-pane scroll-detail" onClick={onLink}>
            {pr ? <Detail pr={pr}/> : <div className="detail-empty dim">Select a PR.</div>}
        </aside>
    )
}

function Detail({pr}: { pr: PR }) {
    const app = useApp()
    const job = app.state?.Jobs[pr.Key]
    const commands = app.state?.Commands ?? []
    const body = useMemo(() => markdown(pr.Body), [pr.Body])
    return (
        <div className="detail">
            <div className="detail-meta">
                <span className="mono">{pr.Repo.split('/')[1]}#{pr.Number}</span>
                {pr.Project && <><span aria-hidden="true">·</span><span>{pr.Project}</span></>}
                <span aria-hidden="true">·</span><DiffStat pr={pr}/>
                <a href={pr.URL} className="detail-gh">GitHub <Icon name="external"/></a>
            </div>
            <h2><a href={pr.URL}>{pr.Title}</a></h2>
            <div className="detail-sub dim">
                {pr.Author} · <code>{pr.HeadRefName}</code> → <code>{pr.BaseRefName}</code> · {pr.ChangedFiles} files
            </div>
            <div className="detail-badges">
                <AttentionBadges pr={pr}/>
                {pr.IsDraft && <Badge tone="neutral">Draft</Badge>}
            </div>
            <div className={'detail-turn' + (pr.OnMe ? ' me' : '')}>{pr.OnMe ? 'Your turn' : 'Their turn'}: {pr.TurnWhy}</div>
            {pr.FailingChecks?.length > 0 && (
                <section>
                    <h3>Failing checks</h3>
                    {pr.FailingChecks.map(c => <div key={c.Name} className="bad">✗ {c.URL ? <a href={c.URL}>{c.Name}</a> : c.Name}</div>)}
                </section>
            )}
            {(pr.Labels?.length > 0 || pr.LinearURL) && (
                <div className="detail-badges">
                    {pr.Labels?.map(l => <Badge key={l} tone="neutral">{l}</Badge>)}
                    {pr.LinearURL && <a href={pr.LinearURL}><Badge tone="info">{pr.Linear}</Badge></a>}
                </div>
            )}
            {job && <AgentBox job={job}/>}
            <div className="detail-commands">
                {commands.map(c => (
                    <Button key={c.Key} size="sm" onClick={() => app.run(c.Key, pr.Key)} title={c.Background ? 'background agent' : ''}>
                        {c.Name} <Kbd keys={c.Key}/>
                    </Button>
                ))}
                <Button size="sm" onClick={() => app.open(pr.URL)}>Open in browser <Kbd keys="o"/></Button>
            </div>
            <section className="markdown" dangerouslySetInnerHTML={{__html: body || '<p class="dim">No description.</p>'}}/>
        </div>
    )
}
