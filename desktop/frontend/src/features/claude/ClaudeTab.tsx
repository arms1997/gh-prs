import {Job, needsYou, PR} from '../../lib/pr'
import {useApp} from '../../state/AppState'
import {Button} from '../../components/ui/Button'
import {Kbd} from '../../components/ui/Kbd'
import {AgentStatus} from './AgentStatus'
import {NeedsYouCard} from './NeedsYouCard'
import {LastResult} from './LastResult'
import {Timeline} from './Timeline'
import {Heading} from '../../components/ui/Heading'

// The PR's conversation with Claude: what the agent is doing, its question for you, and what it
// did. The message box (Composer) sits under the tab, outside its scroll area.
export function ClaudeTab({pr, job}: { pr: PR, job?: Job }) {
    const app = useApp()
    const cmds = app.state?.Commands ?? []
    const question = needsYou(job)
    return (
        <div className="flex flex-col gap-5">
            {job ? (
                <>
                    <AgentStatus pr={pr} job={job}/>
                    {question && <NeedsYouCard pr={pr} question={question} options={job.Summary?.Options ?? []}/>}
                    {job.State !== 'running' && <LastResult job={job}/>}
                    <Timeline job={job}/>
                </>
            ) : (
                <section>
                    <Heading>No agent has worked on this PR yet</Heading>
                    <p className="dim text-xs">Hand it to one, or ask Claude something below.</p>
                </section>
            )}
            <section>
                <Heading>Hand it to Claude</Heading>
                <div className="flex flex-wrap gap-2">
                    {cmds.filter(c => c.Background && !c.Input).map(c => (
                        <Button key={c.Key} size="sm" onClick={() => app.run(c.Key, pr.Key)}>{c.Name} <Kbd keys={c.Key}/></Button>
                    ))}
                </div>
            </section>
        </div>
    )
}
