export type TabDef<T extends string> = { id: T, label: string, extra?: React.ReactNode }

export function Tabs<T extends string>({tabs, active, onChange}: { tabs: TabDef<T>[], active: T, onChange: (id: T) => void }) {
    return (
        <div role="tablist" className="flex gap-4 overflow-x-auto border-b border-border-100 [scrollbar-width:none]">
            {tabs.map(t => (
                <button key={t.id} type="button" role="tab" aria-selected={t.id === active}
                        className={`inline-flex h-10 shrink-0 cursor-pointer items-center gap-1.5 border-x-0 border-t-0 border-b-2 bg-transparent px-0.5 text-base font-medium whitespace-nowrap ${t.id === active ? 'border-primary text-foreground' : 'border-transparent text-dim hover:text-foreground'}`}
                        onClick={() => onChange(t.id)}>
                    {t.label}{t.extra}
                </button>
            ))}
        </div>
    )
}
