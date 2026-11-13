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

// overflow-wrap: long paths, branch names and URLs in agent output wrap instead of running past their box.
const PANE = 'flex min-h-0 min-w-0 flex-col border-l border-border-100 [overflow-wrap:anywhere]'

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
    if (!pr) return <aside aria-label="Pull request details" className={PANE}><div className="dim p-6">Select a PR.</div></aside>

    const job = app.state?.Jobs[pr.Key]
    const failing = (pr.Checks ?? []).filter(c => c.State === 'fail').length
    const tabs: TabDef<DetailTab>[] = [
        {id: 'overview', label: 'Overview'},
        {id: 'brief', label: 'Brief'},
        {id: 'claude', label: 'Claude', extra: needsYou(job) ? <Badge tone="warn">Needs you</Badge>
                : job?.State === 'running' ? <Badge tone="info" icon="spinner">Running</Badge> : undefined},
        {id: 'checks', label: 'Checks', extra: failing > 0 ? <Badge tone="bad">{failing}</Badge> : undefined},
        {id: 'files', label: 'Files', extra: <span className="mono text-xs text-dim">{pr.ChangedFiles}</span>},
    ]
    return (
        <aside aria-label="Pull request details" className={PANE} onClick={onLink}>
            <div className="flex flex-col gap-3 px-6 pt-5">
                <DetailHeader pr={pr}/>
                <Tabs tabs={tabs} active={tab} onChange={setTab}/>
            </div>
            <div className="detail-body scroll-detail flex min-h-0 flex-1 flex-col gap-6 overflow-x-hidden overflow-y-auto px-6 pt-4 pb-8 select-text">
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
