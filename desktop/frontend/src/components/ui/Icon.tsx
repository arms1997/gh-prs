// Stroke icons used across the app (paths from the design canvas).
const PATHS = {
    search: <><circle cx="11" cy="11" r="7"/><path d="m21 21-4.3-4.3"/></>,
    refresh: <><path d="M21 12a9 9 0 0 1-15 6.7L3 16"/><path d="M3 12a9 9 0 0 1 15-6.7L21 8"/><path d="M21 3v5h-5"/><path d="M3 21v-5h5"/></>,
    chevronDown: <path d="m6 9 6 6 6-6"/>,
    chevronRight: <path d="m9 18 6-6-6-6"/>,
    check: <path d="M20 6 9 17l-5-5"/>,
    x: <><path d="M18 6 6 18"/><path d="m6 6 12 12"/></>,
    spinner: <path d="M21 12a9 9 0 1 1-6.2-8.6"/>,
    eye: <><path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7S2 12 2 12z"/><circle cx="12" cy="12" r="3"/></>,
    thread: <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>,
    merge: <><circle cx="6" cy="6" r="3"/><circle cx="18" cy="18" r="3"/><path d="M6 9v12"/><path d="M13 6h3a2 2 0 0 1 2 2v7"/></>,
    terminal: <><path d="m4 17 6-6-6-6"/><path d="M12 19h8"/></>,
    external: <><path d="M15 3h6v6"/><path d="M10 14 21 3"/><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/></>,
    stack: <><path d="M6 3v12"/><path d="M6 15a3 3 0 0 0 3 3h9"/></>,
    question: <><circle cx="12" cy="12" r="9"/><path d="M9.1 9a3 3 0 0 1 5.8 1c0 2-3 3-3 3"/><path d="M12 17h.01"/></>,
    clock: <><circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/></>,
    pin: <><path d="M12 17v5"/><path d="M9 3h6l-1 7 4 3H6l4-3z"/></>,
    eyeOff: <><path d="M3 3l18 18"/><path d="M10.6 5.1A10 10 0 0 1 22 12s-1 2-3 4"/><path d="M6.6 6.6C3.7 8.4 2 12 2 12s3.5 7 10 7c1.7 0 3.2-.5 4.5-1.2"/></>,
    moon: <path d="M20 14.5A8 8 0 1 1 9.5 4a6.5 6.5 0 0 0 10.5 10.5z"/>,
}

export type IconName = keyof typeof PATHS

export function Icon({name, size = 12, className}: { name: IconName, size?: number, className?: string }) {
    return (
        <svg aria-hidden="true" width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor"
             strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round" className={className} style={{flexShrink: 0}}>
            {PATHS[name]}
        </svg>
    )
}
