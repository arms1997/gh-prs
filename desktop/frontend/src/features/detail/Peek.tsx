import {useEffect} from 'react'
import {useApp} from '../../state/AppState'
import {needsYou, nextStep, PR} from '../../lib/pr'
import {Modal} from '../../components/ui/Modal'
import {Kbd} from '../../components/ui/Kbd'
import {AgentStatus} from '../claude/AgentStatus'
import {answerKey, NeedsYouCard} from '../claude/NeedsYouCard'
import {DetailHeader} from './DetailHeader'
import {linkOpener} from './DetailPane'

// A quick look at the selected PR from the list (space): its header and its agent's question.
// space or esc closes it, e opens the whole page, o opens GitHub, 1–9 answer the agent.
export function Peek({pr, onClose, onOpen}: { pr: PR, onClose: () => void, onOpen: () => void }) {
    const app = useApp()
    const job = app.state?.Jobs[pr.Key]
    const question = needsYou(job)
    useEffect(() => {
        const onKey = (e: KeyboardEvent) => {
            const t = e.target as HTMLElement
            if (t.tagName === 'INPUT' || t.tagName === 'TEXTAREA' || e.metaKey || e.ctrlKey || e.altKey) return
            if (e.key === ' ') {
                e.preventDefault()
                onClose()
            } else if (e.key === 'e') onOpen()
            else if (e.key === 'o') app.open(pr.URL)
            else if (answerKey(app, pr, e.key)) onClose()
        }
        window.addEventListener('keydown', onKey)
        return () => window.removeEventListener('keydown', onKey)
    })
    return (
        <Modal label={`Peek at ${pr.Title}`} onClose={onClose}>
            <div className="flex w-[720px] max-w-[90vw] flex-col gap-5 p-6 select-text [overflow-wrap:anywhere]" onClick={linkOpener(app)}>
                <DetailHeader pr={pr} job={job} step={nextStep(pr, job, app.state?.Commands ?? [])}/>
                {job && <AgentStatus pr={pr} job={job}/>}
                {question && <NeedsYouCard pr={pr} question={question} options={job?.Summary?.Options ?? []}/>}
                <div className="flex gap-4 text-xs text-dim">
                    <span className="inline-flex items-center gap-1"><Kbd keys="e"/>open PR</span>
                    <span className="inline-flex items-center gap-1"><Kbd keys="esc"/>close</span>
                </div>
            </div>
        </Modal>
    )
}
