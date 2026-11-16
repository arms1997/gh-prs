import {PR, statusBadges} from '../../lib/pr'
import {DiffStat} from '../../components/pr/PrBits'
import {Badge} from '../../components/ui/Badge'
import {Icon} from '../../components/ui/Icon'

export function DetailHeader({pr}: { pr: PR }) {
    return (
        <div className="flex flex-col gap-2 select-text">
            <div className="flex min-w-0 items-center gap-2 text-xs whitespace-nowrap text-dim">
                <span className="mono">{pr.Repo.split('/')[1]}#{pr.Number}</span>
                {pr.Project && <><span aria-hidden="true">·</span><span className="ellipsis">{pr.Project}</span></>}
                <span aria-hidden="true">·</span><DiffStat pr={pr}/>
                <a href={pr.URL} className="ml-auto inline-flex items-center gap-1 font-medium">GitHub <Icon name="external"/></a>
            </div>
            <h2 className="m-0 text-xl leading-[1.3] font-semibold"><a href={pr.URL} className="text-foreground">{pr.Title}</a></h2>
            <div className="dim overflow-hidden text-xs text-ellipsis whitespace-nowrap">
                {pr.Author} · <code>{pr.HeadRefName}</code> → <code>{pr.BaseRefName}</code>
            </div>
            <div className="flex flex-wrap gap-1.5">
                {statusBadges(pr).map(b => <Badge key={b.text} tone={b.tone} icon={b.icon}>{b.text}</Badge>)}
                {pr.LinearURL && <a href={pr.LinearURL} className="hover:no-underline"><Badge tone="info">{pr.Linear}</Badge></a>}
            </div>
        </div>
    )
}
