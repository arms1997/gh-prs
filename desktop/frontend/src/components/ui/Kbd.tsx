import './ui.css'

// A key or key sequence: "cc" shows as two keys.
export function Kbd({keys}: { keys: string }) {
    const parts = keys !== 'esc' && /^[a-z]{2,3}$/i.test(keys) ? keys.split('') : [keys]
    return <span className="kbds">{parts.map((k, i) => <kbd key={i}>{k}</kbd>)}</span>
}
