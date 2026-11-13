import {useState} from 'react'
import {useApp} from '../../state/AppState'

// Widths you set (config.yml's columns key, see core.ColumnNames), overridden while a drag is under
// way. Unset ones fall back to the CSS defaults, so vars only carries what you set.
export function useWidths() {
    const app = useApp()
    const [dragging, setDragging] = useState<Record<string, number>>()
    const saved = dragging ?? app.state?.Columns ?? {}
    const vars = Object.fromEntries(Object.entries(saved).map(([k, w]) => [`--w-${k}`, `${w}px`])) as React.CSSProperties
    const save = (cols: Record<string, number>) => app.saveColumns(cols).finally(() => setDragging(undefined))

    // The handle sits on its element's left edge, which moves with the pointer: the element grows as
    // you drag left. It starts from the rendered width, which the CSS may have set.
    const handle = (key: string, label: string) => {
        const [min, max] = key === 'detail' ? [280, 1600] : [40, 800]
        const clamp = (w: number) => Math.round(Math.min(max, Math.max(min, w)))
        const current = (el: HTMLElement) => el.parentElement!.getBoundingClientRect().width
        return {
            role: 'separator', 'aria-orientation': 'vertical' as const, 'aria-label': `Resize ${label}`, tabIndex: 0,
            title: 'Drag to resize · double-click to reset',
            className: 'resize-handle' + (dragging?.[key] !== undefined ? ' active' : ''),
            onPointerDown: (e: React.PointerEvent<HTMLElement>) => {
                const el = e.currentTarget, x0 = e.clientX, w0 = current(el)
                let cols = saved
                el.setPointerCapture(e.pointerId)
                el.onpointermove = ev => setDragging(cols = {...saved, [key]: clamp(w0 - (ev.clientX - x0))})
                el.onpointerup = () => {
                    el.onpointermove = el.onpointerup = null
                    if (cols !== saved) save(cols)
                }
            },
            onDoubleClick: () => { const {[key]: _, ...rest} = saved; save(rest) },
            onKeyDown: (e: React.KeyboardEvent<HTMLElement>) => {
                const d = {ArrowLeft: 16, ArrowRight: -16}[e.key]
                if (d) { e.preventDefault(); save({...saved, [key]: clamp(current(e.currentTarget) + d)}) }
            },
        }
    }
    return {vars, handle}
}
