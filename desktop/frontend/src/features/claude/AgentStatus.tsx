import {useApp} from '../../state/AppState'
import {agentCommands, ago, Job, JOB_LABEL, jobActive, PR} from '../../lib/pr'
import {Button} from '../../components/ui/Button'
import {Icon, IconName} from '../../components/ui/Icon'
import {Kbd} from '../../components/ui/Kbd'

const TONE: Record<string, string> = {
    running: 'bg-info-bg text-info-fg', waiting: 'bg-info-bg text-info-fg', done: 'bg-ok-bg text-ok-fg',
    failed: 'bg-bad-bg text-bad-fg', stopped: 'bg-neutral-bg text-neutral-fg',
}
const ICON: Record<string, IconName> = {running: 'spinner', waiting: 'clock', done: 'check', failed: 'x', stopped: 'x'}

export function AgentStatus({pr, job}: { pr: PR, job: Job }) {
    const app = useApp()
    const {stop, takeOver} = agentCommands(app.state?.Commands ?? [])
    const facts = [`Started ${ago(job.Started)}`, job.Turns > 0 && `${job.Turns} turns`, job.Cost > 0 && `$${job.Cost.toFixed(2)}`,
        job.Rounds?.length > 1 && `${job.Rounds.length} rounds`].filter(Boolean)
    return (
        <div className="flex flex-wrap items-center gap-x-2.5 gap-y-2 rounded-lg border border-border-100 px-3 py-2.5">
            <span className={`inline-flex size-7 shrink-0 items-center justify-center rounded-full ${TONE[job.State] ?? 'bg-warn-bg text-warn-fg'}`}><Icon name={ICON[job.State]} size={14} className={job.State === 'running' ? 'animate-spin' : undefined}/></span>
            <div className="flex min-w-0 flex-[1_1_220px] flex-col gap-px">
                <span className="text-base font-medium">{job.Name} · {JOB_LABEL[job.State]}</span>
                <span className="dim small">{facts.join(' · ')}</span>
            </div>
            <div className="ml-9.5 flex flex-wrap gap-1.5">
                {jobActive(job) && stop && <Button size="sm" onClick={() => app.run(stop.Key, pr.Key)}>Stop <Kbd keys={stop.Key}/></Button>}
                {!jobActive(job) && <Button size="sm" onClick={() => app.run(job.Key, pr.Key)}>Run again <Kbd keys={job.Key}/></Button>}
                {takeOver && <Button size="sm" onClick={() => app.run(takeOver.Key, pr.Key)}>Open in Claude Code <Kbd keys={takeOver.Key}/></Button>}
            </div>
        </div>
    )
}
