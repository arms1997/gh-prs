import {useApp} from '../../state/AppState'
import {Kbd} from '../../components/ui/Kbd'

// The key bar at the bottom of a page.
export const FOOTER = 'flex items-center gap-4 overflow-hidden border-t border-border-100 bg-bar px-4 text-xs whitespace-nowrap text-dim'
export const FOOTER_KEY = 'inline-flex items-center gap-1'
export const FOOTER_STATUS = 'ml-auto overflow-hidden text-ellipsis'

const SHORTCUTS: [string, string][] = [['jk', 'move'], ['/', 'filter'], ['o', 'open'], ['cc', 'agent'], ['p', 'organize'], ['⌘K', 'commands'], ['?', 'keys']]

export function Footer({hint}: { hint: string }) {
    const app = useApp()
    const err = app.flash?.err ?? app.state?.IsErr
    return (
        <footer className={FOOTER}>
            {hint ? <span>{hint}</span> : SHORTCUTS.map(([k, d]) => <span key={k} className={FOOTER_KEY}><Kbd keys={k}/>{d}</span>)}
            <span className={`${FOOTER_STATUS} ${err ? 'text-bad-fg' : ''}`}>{app.flash?.text ?? app.state?.Status}</span>
        </footer>
    )
}
