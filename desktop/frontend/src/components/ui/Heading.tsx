import {ReactNode} from 'react'

// A section's title, with an optional note on its right (a count, a link).
export function Heading({children, aside}: { children: ReactNode, aside?: ReactNode }) {
    const h3 = <h3 className="m-0 mb-2 text-base font-semibold">{children}</h3>
    return aside ? <div className="flex items-baseline justify-between">{h3}{aside}</div> : h3
}
