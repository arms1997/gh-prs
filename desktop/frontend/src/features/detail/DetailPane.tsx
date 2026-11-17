import {DetailTabName, useApp} from '../../state/AppState'
import {agentCommands, needsYou, nextStep, PR} from '../../lib/pr'
import {Badge} from '../../components/ui/Badge'
import {TabDef, Tabs} from '../../components/ui/Tabs'
import {DetailHeader} from './DetailHeader'
import {OverviewTab} from './tabs/OverviewTab'
import {PrRail} from './PrRail'
import {ClaudeTab} from '../claude/ClaudeTab'
import {Composer} from '../claude/Composer'
import {ChecksTab} from './tabs/ChecksTab'
import {FilesTab} from './tabs/FilesTab'
import {BriefTab} from '../brief/BriefTab'

export type DetailTab = DetailTabName
export const DETAIL_TABS: DetailTab[] = ['overview', 'brief', 'claude', 'checks', 'files']

// overflow-wrap: long paths, branch names and URLs in agent output wrap instead of running past their box.
const PANE = 'flex min-h-0 min-w-0 flex-col [overflow-wrap:anywhere]'

export function DetailPane({pr, stackedOn, tab, setTab}: {
    pr?: PR, stackedOn: number, tab: DetailTab, setTab: (t: DetailTab) => void
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
    if (!pr) return <aside aria-label="Pull request details" className={PANE}><div className="text-dim p-6">Select a PR.</div></aside>

    const job = app.state?.Jobs[pr.Key]
    const failing = (pr.Checks ?? []).filter(c => c.State === 'fail').length
    const tabs: TabDef<DetailTab>[] = [
        {id: 'overview', label: 'Overview'},
        {id: 'brief', label: 'Brief'},
        {id: 'claude', label: 'Claude', extra: needsYou(job) ? <Badge tone="warn">Needs you</Badge>
                : job?.State === 'running' ? <Badge tone="info" icon="spinner">Running</Badge> : undefined},
        {id: 'checks', label: 'Checks', extra: failing > 0 ? <Badge tone="bad">{failing}</Badge> : undefined},
        {id: 'files', label: 'Files', extra: <span className="font-mono text-xs text-dim">{pr.ChangedFiles}</span>},
    ]
    return (
        <aside aria-label="Pull request details" className={PANE} onClick={onLink}>
            <div className="mx-auto flex w-full max-w-[1240px] flex-col gap-4 px-8 pt-6">
                <DetailHeader pr={pr} job={job} step={nextStep(pr, job, app.state?.Commands ?? [])}/>
                <Tabs tabs={tabs} active={tab} onChange={setTab}/>
            </div>
            <div className="scroll-detail min-h-0 flex-1 overflow-x-hidden overflow-y-auto select-text [scrollbar-gutter:stable_both-edges]">
                {tab === 'overview' ? (
                    <div className="mx-auto grid max-w-[1240px] grid-cols-[minmax(0,1fr)_380px] items-start gap-10 px-8 pt-5 pb-10 narrow:grid-cols-[minmax(0,1fr)_320px] narrow:gap-8">
                        <div className="flex min-w-0 flex-col gap-6"><OverviewTab pr={pr} showChecks={() => setTab('checks')}/></div>
                        <PrRail pr={pr} job={job} stackedOn={stackedOn} showTab={setTab}/>
                    </div>
                ) : (
                    <div className="mx-auto flex max-w-[924px] flex-col gap-6 px-8 pt-5 pb-10">
                        {tab === 'brief' && <BriefTab pr={pr}/>}
                        {tab === 'claude' && <ClaudeTab pr={pr} job={job}/>}
                        {tab === 'checks' && <ChecksTab pr={pr}/>}
                        {tab === 'files' && <FilesTab pr={pr}/>}
                    </div>
                )}
            </div>
            {tab === 'claude' && <div className="mx-auto w-full max-w-[924px]"><Composer pr={pr} job={job} cmds={agentCommands(app.state?.Commands ?? [])}/></div>}
        </aside>
    )
}
