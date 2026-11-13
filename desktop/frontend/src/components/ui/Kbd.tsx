// A key or key sequence: "cc" shows as two keys.
export function Kbd({keys}: { keys: string }) {
    const parts = keys !== 'esc' && /^[a-z]{2,3}$/i.test(keys) ? keys.split('') : [keys]
    return (
        <span className="inline-flex gap-0.5">
            {parts.map((k, i) => (
                <kbd key={i} className="inline-flex h-4.5 min-w-4.5 items-center justify-center rounded-sm border border-b-2 border-border-200 bg-background px-1 font-mono text-2xs/none text-neutral-fg">{k}</kbd>
            ))}
        </span>
    )
}
