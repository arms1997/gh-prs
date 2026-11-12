import {useEffect, useRef, useState} from 'react'
import {Modal} from '../../components/ui/Modal'
import {Icon, IconName} from '../../components/ui/Icon'
import {Kbd} from '../../components/ui/Kbd'
import './palette.css'

export type PaletteItem = { section: string, label: string, hint?: string, keys?: string, icon?: IconName, run: () => void }

// ⌘K: search every action by name; each shows its shortcut so you learn the keys.
export function CommandPalette({items, context, onClose}: { items: PaletteItem[], context?: string, onClose: () => void }) {
    const [q, setQ] = useState('')
    const [i, setI] = useState(0)
    const list = useRef<HTMLDivElement>(null)
    const shown = items.filter(it => (it.label + ' ' + (it.hint ?? '')).toLowerCase().includes(q.toLowerCase()))
    const cur = Math.min(i, shown.length - 1)
    useEffect(() => list.current?.querySelector('.sel')?.scrollIntoView({block: 'nearest'}), [cur])

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
            <div className="palette">
                <label className="palette-input">
                    <Icon name="search" size={16}/>
                    <span className="sr-only">Command</span>
                    <input autoFocus value={q} placeholder="Type a command" onKeyDown={onKey}
                           onChange={e => { setQ(e.target.value); setI(0) }}/>
                    {context && <span className="palette-context">Acting on <span className="mono">{context}</span></span>}
                </label>
                <div className="palette-list" ref={list}>
                    {shown.map((it, n) => (
                        <div key={it.section + it.label}>
                            {(n === 0 || shown[n - 1].section !== it.section) && <div className="palette-section">{it.section}</div>}
                            <button type="button" className={'palette-item' + (n === cur ? ' sel' : '')}
                                    onMouseMove={() => setI(n)} onClick={() => pick(it)}>
                                <span className="palette-icon">{it.icon && <Icon name={it.icon} size={16}/>}</span>
                                <span className="palette-label">{it.label}</span>
                                {it.hint && <span className="dim">{it.hint}</span>}
                                {it.keys && <span className="palette-keys"><Kbd keys={it.keys}/></span>}
                            </button>
                        </div>
                    ))}
                    {shown.length === 0 && <div className="palette-section">No matching command.</div>}
                </div>
                <div className="palette-foot">
                    <span><Kbd keys="↑"/><Kbd keys="↓"/> select</span>
                    <span><Kbd keys="⏎"/> run</span>
                    <span><Kbd keys="esc"/> close</span>
                </div>
            </div>
        </Modal>
    )
}
