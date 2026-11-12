import {Modal} from '../../components/ui/Modal'
import {Kbd} from '../../components/ui/Kbd'
import {Command} from '../../lib/pr'
import './organize.css'

const KEYS: [string, string][] = [
    ['⌘K', 'command palette'], ['j k ↑ ↓', 'move'], ['g g / G', 'top / bottom'], ['h l', 'collapse / expand a group'],
    ['⏎', 'run the next step, or toggle a group'], ['[ ]', 'previous / next detail tab'], ['o', 'open on GitHub'], ['r', 'refresh'], ['/', 'filter'],
    ['esc', 'clear filters, close'], ['pp ps ph', 'pin, snooze, hide'], ['pm', 'move to a project'], ['pa', 'show hidden & snoozed'],
    ['?', 'this help'],
]

export function HelpDialog({commands, onClose}: { commands: Command[], onClose: () => void }) {
    return (
        <Modal label="Keys" onClose={onClose}>
            <div className="dialog">
                <h3>Keys</h3>
                {KEYS.map(([k, d]) => <div key={k} className="helprow"><kbd>{k}</kbd><span>{d}</span></div>)}
                {commands.map(c => <div key={c.Key} className="helprow"><span><Kbd keys={c.Key}/></span><span>{c.Name}</span></div>)}
            </div>
        </Modal>
    )
}
