import {ReactNode} from 'react'

// Rows in a bordered box, divided by lines: checks, files.
export function BoxList({children}: { children: ReactNode }) {
    return <div className="flex flex-col rounded-lg border border-border-100 [&>*+*]:border-t [&>*+*]:border-border-100">{children}</div>
}

export function BoxRow({children, className = '', title}: { children: ReactNode, className?: string, title?: string }) {
    return <div className={`flex min-h-9 items-center gap-2 px-3 text-sm ${className}`} title={title}>{children}</div>
}

// What sits at a row's right edge.
export const BOX_END = 'ml-auto inline-flex shrink-0 items-center gap-1 whitespace-nowrap'
