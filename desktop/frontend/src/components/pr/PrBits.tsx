// Small pieces that show one PR's state, shared by the list and the detail pane.
import {Badge} from '../ui/Badge'
import {Icon} from '../ui/Icon'
import {Kbd} from '../ui/Kbd'
import {attention, NextStep, PR} from '../../lib/pr'

// Filled when it's your turn, hollow when it's theirs.
export function TurnDot({pr}: { pr: PR }) {
    return <span className={'size-2 justify-self-end rounded-full border-[1.5px] ' + (pr.OnMe ? 'border-warn-fg bg-warn-fg' : 'border-faint')} title={(pr.OnMe ? 'Your turn: ' : 'Their turn: ') + pr.TurnWhy}/>
}

export function DiffStat({pr}: { pr: PR }) {
    return <span className="inline-flex gap-1.5 font-mono text-xs whitespace-nowrap"><span className="text-ok-fg">+{pr.Additions}</span><span className="text-bad-fg">−{pr.Deletions}</span></span>
}

export function AttentionBadges({pr}: { pr: PR }) {
    return <span className="flex gap-1 overflow-hidden">{attention(pr).map(a => <Badge key={a.text} tone={a.tone} icon={a.icon}>{a.text}</Badge>)}</span>
}

const NEXT = 'inline-flex items-center gap-1.5 overflow-hidden text-sm text-ellipsis whitespace-nowrap'

export function NextStepLabel({step, showKey}: { step: NextStep, showKey?: boolean }) {
    switch (step.kind) {
        case 'agent':
            return <span className={`${NEXT} font-medium text-info-fg`}><Icon name={step.waiting ? 'clock' : 'terminal'}/>{step.text}</span>
        case 'command':
            return <span className={`${NEXT} font-medium text-link`}><Icon name="chevronRight"/>{step.text}{showKey && <Kbd keys="⏎"/>}</span>
        case 'wait':
            return <span className={`${NEXT} text-dim`}>{step.text}</span>
    }
    return <span className={`${NEXT} text-dim`}>—</span>
}
