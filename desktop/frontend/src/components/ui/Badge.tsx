import {Icon, IconName} from './Icon'

export type Tone = 'ok' | 'bad' | 'warn' | 'info' | 'neutral'

const TONE: Record<Tone, string> = {
    ok: 'bg-ok-bg text-ok-fg',
    bad: 'bg-bad-bg text-bad-fg',
    warn: 'bg-warn-bg text-warn-fg',
    info: 'bg-info-bg text-info-fg',
    neutral: 'bg-neutral-bg text-neutral-fg',
}

export function Badge({tone, icon, children}: { tone: Tone, icon?: IconName, children: React.ReactNode }) {
    return (
        <span className={`inline-flex h-5 items-center gap-1 rounded-full px-2 text-xs/none font-medium whitespace-nowrap ${TONE[tone]}`}>
            {icon && <Icon name={icon} className={icon === 'spinner' ? 'animate-spin' : undefined}/>}
            {children}
        </span>
    )
}
