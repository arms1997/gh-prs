import {Icon, IconName} from './Icon'
import './ui.css'

export type Tone = 'ok' | 'bad' | 'warn' | 'info' | 'neutral'

export function Badge({tone, icon, children}: { tone: Tone, icon?: IconName, children: React.ReactNode }) {
    return (
        <span className={'badge ' + tone}>
            {icon && <Icon name={icon} className={icon === 'spinner' ? 'spin' : undefined}/>}
            {children}
        </span>
    )
}
