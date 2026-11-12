import './ui.css'

// A centered dialog over a shaded backdrop; clicking the backdrop closes it.
export function Modal({label, onClose, top, children}: {
    label: string, onClose: () => void, top?: boolean, children: React.ReactNode
}) {
    return (
        <div className={'overlay' + (top ? ' top' : '')} onMouseDown={e => e.target === e.currentTarget && onClose()}>
            <div role="dialog" aria-label={label} className="modal">{children}</div>
        </div>
    )
}
