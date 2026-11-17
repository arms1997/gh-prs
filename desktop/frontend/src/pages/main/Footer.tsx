import {useApp} from '../../state/AppState'
import {Kbd} from '../../components/ui/Kbd'

// The key bar at the bottom of a page.
export const FOOTER = 'flex items-center gap-4 overflow-hidden border-t border-border-100 bg-bar px-4 text-xs whitespace-nowrap text-dim'
export const FOOTER_KEY = 'inline-flex items-center gap-1'
export const FOOTER_STATUS = 'ml-auto overflow-hidden text-ellipsis'

const SHORTCUTS: [string, string][] = [['jk', 'move'], ['e', 'open PR'], ['⏎', 'next step'], ['/', 'filter'], ['cc', 'agent'], ['p', 'organize'], ['⌘K', 'commands'], ['?', 'keys']]
const PR_SHORTCUTS: [string, string][] = [['esc', 'back to list'], ['jk', 'next, previous PR'], ['[ ]', 'tabs'], ['⏎', 'next step'], ['i', 'message Claude'], ['⌘K', 'commands'], ['?', 'keys']]

export function Footer({hint, viewing}: { hint: string, viewing?: boolean }) {
    const app = useApp()
    const err = app.flash?.err ?? app.state?.IsErr
    return (
        <footer className={FOOTER}>
            {hint ? <span>{hint}</span> : (viewing ? PR_SHORTCUTS : SHORTCUTS).map(([k, d]) => <span key={k} className={FOOTER_KEY}><Kbd keys={k}/>{d}</span>)}
            <span className={`${FOOTER_STATUS} ${err ? 'text-bad-fg' : ''}`}>{app.flash?.text ?? app.state?.Status}</span>
        </footer>
    )
}
