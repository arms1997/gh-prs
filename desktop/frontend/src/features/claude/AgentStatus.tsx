import {useApp} from '../../state/AppState'
import {agentCommands, ago, Job, JOB_LABEL, jobActive, PR} from '../../lib/pr'
import {Button} from '../../components/ui/Button'
import {Icon, IconName} from '../../components/ui/Icon'
import {Kbd} from '../../components/ui/Kbd'

const ICON: Record<string, IconName> = {running: 'spinner', waiting: 'clock', done: 'check', failed: 'x', stopped: 'x'}

export function AgentStatus({pr, job}: { pr: PR, job: Job }) {
    const app = useApp()
    const {stop, takeOver} = agentCommands(app.state?.Commands ?? [])
    const facts = [`Started ${ago(job.Started)}`, job.Turns > 0 && `${job.Turns} turns`, job.Cost > 0 && `$${job.Cost.toFixed(2)}`,
        job.Rounds?.length > 1 && `${job.Rounds.length} rounds`].filter(Boolean)
    return (
        <div className={'agent-status ' + job.State}>
            <span className="agent-icon"><Icon name={ICON[job.State]} size={14} className={job.State === 'running' ? 'spin' : undefined}/></span>
            <div className="agent-status-text">
                <span className="agent-status-title">{job.Name} · {JOB_LABEL[job.State]}</span>
                <span className="dim small">{facts.join(' · ')}</span>
            </div>
            <div className="agent-status-actions">
                {jobActive(job) && stop && <Button size="sm" onClick={() => app.run(stop.Key, pr.Key)}>Stop <Kbd keys={stop.Key}/></Button>}
                {!jobActive(job) && <Button size="sm" onClick={() => app.run(job.Key, pr.Key)}>Run again <Kbd keys={job.Key}/></Button>}
                {takeOver && <Button size="sm" onClick={() => app.run(takeOver.Key, pr.Key)}>Open in Claude Code <Kbd keys={takeOver.Key}/></Button>}
            </div>
        </div>
    )
}
