import {useApp} from '../../state/AppState'
import {Kbd} from '../../components/ui/Kbd'

const SHORTCUTS: [string, string][] = [['jk', 'move'], ['/', 'filter'], ['o', 'open'], ['cc', 'agent'], ['p', 'organize'], ['⌘K', 'commands'], ['?', 'keys']]

export function Footer({hint}: { hint: string }) {
    const app = useApp()
    const err = app.flash?.err ?? app.state?.IsErr
    return (
        <footer className="footer">
            {hint ? <span>{hint}</span> : SHORTCUTS.map(([k, d]) => <span key={k} className="footer-key"><Kbd keys={k}/>{d}</span>)}
            <span className={'footer-status' + (err ? ' bad' : '')}>{app.flash?.text ?? app.state?.Status}</span>
        </footer>
    )
}
