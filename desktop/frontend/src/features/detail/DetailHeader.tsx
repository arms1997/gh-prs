import {PR, statusBadges} from '../../lib/pr'
import {DiffStat} from '../../components/pr/PrBits'
import {Badge} from '../../components/ui/Badge'
import {Icon} from '../../components/ui/Icon'

export function DetailHeader({pr}: { pr: PR }) {
    return (
        <div className="detail-header">
            <div className="detail-meta">
                <span className="mono">{pr.Repo.split('/')[1]}#{pr.Number}</span>
                {pr.Project && <><span aria-hidden="true">·</span><span className="ellipsis">{pr.Project}</span></>}
                <span aria-hidden="true">·</span><DiffStat pr={pr}/>
                <a href={pr.URL} className="detail-gh">GitHub <Icon name="external"/></a>
            </div>
            <h2><a href={pr.URL}>{pr.Title}</a></h2>
            <div className="detail-sub dim">
                {pr.Author} · <code>{pr.HeadRefName}</code> → <code>{pr.BaseRefName}</code>
            </div>
            <div className="detail-badges">
                {statusBadges(pr).map(b => <Badge key={b.text} tone={b.tone} icon={b.icon}>{b.text}</Badge>)}
                {pr.LinearURL && <a href={pr.LinearURL} className="badge-link"><Badge tone="info">{pr.Linear}</Badge></a>}
            </div>
        </div>
    )
}
