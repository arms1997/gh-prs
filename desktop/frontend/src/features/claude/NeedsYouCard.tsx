import {useState} from 'react'
import {useApp} from '../../state/AppState'
import {agentCommands, PR} from '../../lib/pr'
import {markdownInline} from '../../lib/markdown'
import {Button} from '../../components/ui/Button'
import {Icon} from '../../components/ui/Icon'
import {Kbd} from '../../components/ui/Kbd'

// The agent's open question, with its suggested answers as buttons and a box for your own.
// Answering resumes the agent's session with your reply.
export function NeedsYouCard({pr, question, options}: { pr: PR, question: string, options: string[] }) {
    const app = useApp()
    const {reply} = agentCommands(app.state?.Commands ?? [])
    const [text, setText] = useState('')
    const answer = (s: string) => reply && app.run(reply.Key, pr.Key, s)
    return (
        <div className="flex flex-col gap-2.5 rounded-lg bg-warn-bg p-3.5">
            <div className="flex items-center gap-2 text-xs font-semibold text-warn-fg"><Icon name="question" size={14}/>Claude is waiting for you</div>
            <p className="md-inline m-0 text-base leading-normal" dangerouslySetInnerHTML={{__html: markdownInline(question)}}/>
            {reply ? (
                <>
                    {options.length > 0 && (
                        <div className="flex flex-wrap gap-2">
                            {options.map((o, i) => (
                                <Button key={o} size="sm" variant={i === 0 ? 'primary' : 'secondary'} onClick={() => answer(o)}>{o}</Button>
                            ))}
                        </div>
                    )}
                    <form className="flex items-center gap-2 rounded-md border border-border-200 bg-background px-2.5 focus-within:border-sel-ring h-9" onSubmit={e => {
                        e.preventDefault()
                        if (text.trim()) answer(text)
                        setText('')
                    }}>
                        <span className="sr-only">Reply to Claude</span>
                        <input className="min-w-0 flex-1 border-0 bg-transparent text-sm outline-none" value={text} placeholder={options.length ? 'Or reply in your own words' : 'Reply to Claude'}
                               onChange={e => setText(e.target.value)}/>
                        <Kbd keys="⏎"/>
                    </form>
                </>
            ) : (
                <p className="dim text-xs">Add a reply command (agent_session with an input) to answer from here.</p>
            )}
        </div>
    )
}
