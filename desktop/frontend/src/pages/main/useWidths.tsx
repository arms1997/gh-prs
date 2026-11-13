import {useState} from 'react'
import {useApp} from '../../state/AppState'

// Widths you set (config.yml's columns key, see core.ColumnNames), overridden while a drag is under
// way. Unset ones fall back to the CSS defaults, so vars only carries what you set.
// A column's handle straddles the line before its header; the details pane's shows only while live.
const HANDLE = {
    column: "absolute -top-2 -bottom-2 -left-2.5 w-2 cursor-col-resize focus-visible:outline-none after:absolute after:top-2 after:bottom-2 after:left-[3px] after:w-px after:bg-border-200 after:content-[''] live:after:top-1.5 live:after:bottom-1.5 live:after:w-0.5 live:after:rounded-[1px] live:after:bg-sel-ring",
    detail: "absolute top-0 bottom-0 -left-1 z-2 w-2 cursor-col-resize focus-visible:outline-none after:absolute after:left-[3px] after:hidden after:content-[''] live:after:top-0 live:after:bottom-0 live:after:block live:after:w-0.5 live:after:rounded-[1px] live:after:bg-sel-ring",
}

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
            className: HANDLE[key === 'detail' ? 'detail' : 'column'], 'data-active': dragging?.[key] !== undefined || undefined,
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
