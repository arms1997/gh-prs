import {useState} from 'react'
import {Modal} from '../../components/ui/Modal'
import {PR} from '../../lib/pr'
import './organize.css'

export function MoveDialog({pr, projects, onMove, onClose}: {
    pr: PR, projects: string[], onMove: (project: string) => void, onClose: () => void
}) {
    const [value, setValue] = useState(pr.Project ?? '')
    return (
        <Modal label="Move to a project" onClose={onClose}>
            <form className="dialog" onSubmit={e => {
                e.preventDefault()
                onMove(value)
                onClose()
            }}>
                <h3>Move #{pr.Number} to a project</h3>
                <input autoFocus list="projects" value={value} placeholder="Project name (empty: use its label)"
                       onChange={e => setValue(e.target.value)} onKeyDown={e => e.key === 'Escape' && onClose()}/>
                <datalist id="projects">{projects.map(p => <option key={p} value={p}/>)}</datalist>
                <div className="dim small">Enter to move · Esc to cancel</div>
            </form>
        </Modal>
    )
}
