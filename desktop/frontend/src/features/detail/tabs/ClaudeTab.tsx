import {useApp} from '../../../state/AppState'
import {Job, jobActive, PR} from '../../../lib/pr'
import {Button} from '../../../components/ui/Button'
import {Kbd} from '../../../components/ui/Kbd'
import {AgentBox} from '../AgentBox'

// The agent's last run and the agent commands. The full conversation view comes in part 3.
export function ClaudeTab({pr, job}: { pr: PR, job?: Job }) {
    const app = useApp()
    const agents = (app.state?.Commands ?? []).filter(c => c.Background)
    const stop = app.state?.Commands.find(c => c.Builtin === 'stop_agent')
    return (
        <>
            {job ? <AgentBox job={job}/> : <p className="dim">No agent has worked on this PR yet.</p>}
            {jobActive(job) && stop && (
                <div className="actions"><Button size="sm" onClick={() => app.run(stop.Key, pr.Key)}>Stop agent <Kbd keys={stop.Key}/></Button></div>
            )}
            <section>
                <h3>Hand it to Claude</h3>
                <div className="actions">
                    {agents.map(c => (
                        <Button key={c.Key} size="sm" onClick={() => app.run(c.Key, pr.Key)}>{c.Name} <Kbd keys={c.Key}/></Button>
                    ))}
                </div>
            </section>
        </>
    )
}
