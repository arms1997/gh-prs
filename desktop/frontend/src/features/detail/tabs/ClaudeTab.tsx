import {useApp} from '../../../state/AppState'
import {Job, PR} from '../../../lib/pr'
import {Button} from '../../../components/ui/Button'
import {Kbd} from '../../../components/ui/Kbd'
import {AgentBox} from '../AgentBox'

// The agent's last run and the agent commands. The full conversation view comes in part 3.
export function ClaudeTab({pr, job}: { pr: PR, job?: Job }) {
    const app = useApp()
    const agents = (app.state?.Commands ?? []).filter(c => c.Background)
    return (
        <>
            {job ? <AgentBox job={job}/> : <p className="dim">No agent has worked on this PR yet.</p>}
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
