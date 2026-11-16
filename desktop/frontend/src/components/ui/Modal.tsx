import {useEffect} from 'react'

// A centered dialog over a shaded backdrop; Esc or clicking the backdrop closes it.
export function Modal({label, onClose, top, children}: {
    label: string, onClose: () => void, top?: boolean, children: React.ReactNode
}) {
    useEffect(() => {
        const onKey = (e: KeyboardEvent) => e.key === 'Escape' && onClose()
        window.addEventListener('keydown', onKey)
        return () => window.removeEventListener('keydown', onKey)
    }, [onClose])
    return (
        <div className={`fixed inset-0 z-50 grid bg-shade ${top ? 'items-start justify-items-center pt-[12vh]' : 'place-items-center'}`}
             onMouseDown={e => e.target === e.currentTarget && onClose()}>
            <div role="dialog" aria-label={label}
                 className="max-h-[76vh] min-w-[420px] overflow-y-auto rounded-xl bg-background shadow-pop [--thumb:var(--text-faint)]">{children}</div>
        </div>
    )
}
