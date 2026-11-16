import {Modal} from '../../components/ui/Modal'
import {Kbd} from '../../components/ui/Kbd'
import {Command} from '../../lib/pr'

const KEYS: [string, string][] = [
    ['⌘K', 'command palette'], ['j k ↑ ↓', 'move'], ['g g / G', 'top / bottom'], ['h l', 'collapse / expand a group'],
    ['⏎', 'run the next step, or toggle a group'], ['[ ]', 'previous / next detail tab'], ['i', 'message Claude about the PR'], ['g a', 'all Claude sessions'], ['b', 'daily brief'], ['B', 'write a brief of this PR (Claude)'], ['o', 'open on GitHub'], ['r', 'refresh'], ['/', 'filter'],
    ['esc', 'clear filters, close'], ['pp ps ph', 'pin, snooze, hide'], ['pm', 'move to a project'], ['pa', 'show hidden & snoozed'],
    ['?', 'this help'],
]

const ROW = 'grid grid-cols-[150px_1fr] items-center gap-3 text-sm'

export function HelpDialog({commands, onClose}: { commands: Command[], onClose: () => void }) {
    return (
        <Modal label="Keys" onClose={onClose}>
            <div className="grid gap-2 px-5.5 py-4.5">
                <h3 className="m-0 mb-1.5 text-lg font-semibold">Keys</h3>
                {KEYS.map(([k, d]) => <div key={k} className={ROW}><Kbd keys={k}/><span>{d}</span></div>)}
                {commands.map(c => <div key={c.Key} className={ROW}><span><Kbd keys={c.Key}/></span><span>{c.Name}</span></div>)}
            </div>
        </Modal>
    )
}
