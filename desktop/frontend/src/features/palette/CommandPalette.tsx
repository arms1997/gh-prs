import {useEffect, useRef, useState} from 'react'
import {Modal} from '../../components/ui/Modal'
import {Icon, IconName} from '../../components/ui/Icon'
import {Kbd} from '../../components/ui/Kbd'

export type PaletteItem = { section: string, label: string, hint?: string, keys?: string, icon?: IconName, run: () => void }

const SECTION = 'px-3 pt-3 pb-1 text-xs font-medium text-dim'

// ⌘K: search every action by name; each shows its shortcut so you learn the keys.
export function CommandPalette({items, context, onClose}: { items: PaletteItem[], context?: string, onClose: () => void }) {
    const [q, setQ] = useState('')
    const [i, setI] = useState(0)
    const list = useRef<HTMLDivElement>(null)
    const shown = items.filter(it => (it.label + ' ' + (it.hint ?? '')).toLowerCase().includes(q.toLowerCase()))
    const cur = Math.min(i, shown.length - 1)
    useEffect(() => { list.current?.querySelector('[data-sel]')?.scrollIntoView({block: 'nearest'}) }, [cur])

    const pick = (it?: PaletteItem) => {
        if (!it) return
        onClose()
        it.run()
    }
    const onKey = (e: React.KeyboardEvent) => {
        if (e.key === 'ArrowDown' || (e.ctrlKey && e.key === 'n')) setI(Math.min(cur + 1, shown.length - 1))
        else if (e.key === 'ArrowUp' || (e.ctrlKey && e.key === 'p')) setI(Math.max(cur - 1, 0))
        else if (e.key === 'Enter') pick(shown[cur])
        else if (e.key === 'Escape') onClose()
        else return
        e.preventDefault()
        e.stopPropagation()
    }

    return (
        <Modal label="Command palette" onClose={onClose} top>
            <div className="flex w-[600px] flex-col">
                <label className="flex h-13 items-center gap-2.5 border-b border-border-100 px-4 text-faint">
                    <Icon name="search" size={16}/>
                    <span className="sr-only">Command</span>
                    <input autoFocus className="flex-1 border-0 bg-transparent text-lg text-foreground outline-none" value={q} placeholder="Type a command" onKeyDown={onKey}
                           onChange={e => { setQ(e.target.value); setI(0) }}/>
                    {context && <span className="text-xs whitespace-nowrap text-dim">Acting on <span className="mono">{context}</span></span>}
                </label>
                <div className="max-h-[50vh] overflow-y-auto px-2 pt-1 pb-2" ref={list}>
                    {shown.map((it, n) => (
                        <div key={it.section + it.label}>
                            {(n === 0 || shown[n - 1].section !== it.section) && <div className={SECTION}>{it.section}</div>}
                            <button type="button" data-sel={n === cur || undefined}
                                    className={`flex h-10 w-full cursor-pointer items-center gap-3 rounded-md border-0 px-3 text-left text-xs ${n === cur ? 'bg-sel-bg shadow-sel' : 'bg-transparent'}`}
                                    onMouseMove={() => setI(n)} onClick={() => pick(it)}>
                                <span className="inline-flex w-4 text-faint">{it.icon && <Icon name={it.icon} size={16}/>}</span>
                                <span className="text-base font-medium">{it.label}</span>
                                {it.hint && <span className="dim">{it.hint}</span>}
                                {it.keys && <span className="ml-auto"><Kbd keys={it.keys}/></span>}
                            </button>
                        </div>
                    ))}
                    {shown.length === 0 && <div className={SECTION}>No matching command.</div>}
                </div>
                <div className="flex h-9 items-center gap-4 border-t border-border-100 bg-bar px-4 text-xs text-dim">
                    <span className="inline-flex items-center gap-1"><Kbd keys="↑"/><Kbd keys="↓"/> select</span>
                    <span className="inline-flex items-center gap-1"><Kbd keys="⏎"/> run</span>
                    <span className="inline-flex items-center gap-1"><Kbd keys="esc"/> close</span>
                </div>
            </div>
        </Modal>
    )
}
