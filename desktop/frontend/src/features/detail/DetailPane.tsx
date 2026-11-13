import {DetailTabName, useApp} from '../../state/AppState'
import {agentCommands, needsYou, PR} from '../../lib/pr'
import {Badge} from '../../components/ui/Badge'
import {TabDef, Tabs} from '../../components/ui/Tabs'
import {DetailHeader} from './DetailHeader'
import {OverviewTab} from './tabs/OverviewTab'
import {ClaudeTab} from '../claude/ClaudeTab'
import {Composer} from '../claude/Composer'
import {ChecksTab} from './tabs/ChecksTab'
import {FilesTab} from './tabs/FilesTab'
import {BriefTab} from '../brief/BriefTab'
import './detail.css'

export type DetailTab = DetailTabName
export const DETAIL_TABS: DetailTab[] = ['overview', 'brief', 'claude', 'checks', 'files']

export function DetailPane({pr, tab, setTab, openPalette}: {
    pr?: PR, tab: DetailTab, setTab: (t: DetailTab) => void, openPalette: () => void
}) {
    const app = useApp()
    // Links in the pane, including ones in PR descriptions, open in the browser.
    const onLink = (e: React.MouseEvent) => {
        const a = (e.target as HTMLElement).closest('a')
        if (a) {
            e.preventDefault()
            app.open(a.getAttribute('href') ?? '')
        }
    }
    if (!pr) return <aside aria-label="Pull request details" className="detail-pane"><div className="detail-empty dim">Select a PR.</div></aside>

    const job = app.state?.Jobs[pr.Key]
    const failing = (pr.Checks ?? []).filter(c => c.State === 'fail').length
    const tabs: TabDef<DetailTab>[] = [
        {id: 'overview', label: 'Overview'},
        {id: 'brief', label: 'Brief'},
        {id: 'claude', label: 'Claude', extra: needsYou(job) ? <Badge tone="warn">Needs you</Badge>
                : job?.State === 'running' ? <Badge tone="info" icon="spinner">Running</Badge> : undefined},
        {id: 'checks', label: 'Checks', extra: failing > 0 ? <Badge tone="bad">{failing}</Badge> : undefined},
        {id: 'files', label: 'Files', extra: <span className="tab-count mono">{pr.ChangedFiles}</span>},
    ]
    return (
        <aside aria-label="Pull request details" className="detail-pane" onClick={onLink}>
            <div className="detail-top">
                <DetailHeader pr={pr}/>
                <Tabs tabs={tabs} active={tab} onChange={setTab}/>
            </div>
            <div className="detail-body scroll-detail">
                {tab === 'overview' && <OverviewTab pr={pr} showChecks={() => setTab('checks')} openPalette={openPalette}/>}
                {tab === 'brief' && <BriefTab pr={pr}/>}
                {tab === 'claude' && <ClaudeTab pr={pr} job={job}/>}
                {tab === 'checks' && <ChecksTab pr={pr}/>}
                {tab === 'files' && <FilesTab pr={pr}/>}
            </div>
            {tab === 'claude' && <Composer pr={pr} job={job} cmds={agentCommands(app.state?.Commands ?? [])}/>}
        </aside>
    )
}
