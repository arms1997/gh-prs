import './ui.css'

export type TabDef<T extends string> = { id: T, label: string, extra?: React.ReactNode }

export function Tabs<T extends string>({tabs, active, onChange}: { tabs: TabDef<T>[], active: T, onChange: (id: T) => void }) {
    return (
        <div role="tablist" className="tabs">
            {tabs.map(t => (
                <button key={t.id} type="button" role="tab" aria-selected={t.id === active}
                        className={'tab' + (t.id === active ? ' active' : '')} onClick={() => onChange(t.id)}>
                    {t.label}{t.extra}
                </button>
            ))}
        </div>
    )
}
