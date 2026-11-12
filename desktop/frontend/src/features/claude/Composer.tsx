import {useState} from 'react'
import {useApp} from '../../state/AppState'
import {agentCommands, Job, jobActive, PR} from '../../lib/pr'
import {Kbd} from '../../components/ui/Kbd'

// The message box under the conversation. With an agent session on the PR it replies to it
// (resuming the session); without one it asks Claude in a new session.
export function Composer({pr, job, cmds}: { pr: PR, job?: Job, cmds: ReturnType<typeof agentCommands> }) {
    const app = useApp()
    const [text, setText] = useState('')
    const cmd = job && cmds.reply ? cmds.reply : cmds.ask
    if (!cmd) return null
    const hint = cmd === cmds.reply
        ? (jobActive(job) ? 'Message Claude (stops the running agent and continues its session)' : 'Message Claude in this session')
        : 'Ask Claude about this PR'
    return (
        <form className="composer" onSubmit={e => {
            e.preventDefault()
            if (!text.trim()) return
            app.run(cmd.Key, pr.Key, text)
            setText('')
        }}>
            <label>
                <span className="sr-only">{hint}</span>
                <input id="composer" value={text} placeholder={hint} onChange={e => setText(e.target.value)}
                       onKeyDown={e => e.key === 'Escape' && (e.target as HTMLInputElement).blur()}/>
                <Kbd keys="i"/>
            </label>
        </form>
    )
}
