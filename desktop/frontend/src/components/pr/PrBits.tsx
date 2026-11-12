// Small pieces that show one PR's state, shared by the list and the detail pane.
import {Badge} from '../ui/Badge'
import {Icon} from '../ui/Icon'
import {Kbd} from '../ui/Kbd'
import {attention, NextStep, PR} from '../../lib/pr'
import './pr.css'

// Filled when it's your turn, hollow when it's theirs.
export function TurnDot({pr}: { pr: PR }) {
    return <span className={'turn-dot' + (pr.OnMe ? ' me' : '')} title={(pr.OnMe ? 'Your turn: ' : 'Their turn: ') + pr.TurnWhy}/>
}

export function DiffStat({pr}: { pr: PR }) {
    return <span className="diffstat mono"><span className="ok">+{pr.Additions}</span><span className="bad">−{pr.Deletions}</span></span>
}

export function AttentionBadges({pr}: { pr: PR }) {
    return <span className="badges">{attention(pr).map(a => <Badge key={a.text} tone={a.tone} icon={a.icon}>{a.text}</Badge>)}</span>
}

export function NextStepLabel({step, showKey}: { step: NextStep, showKey?: boolean }) {
    switch (step.kind) {
        case 'agent':
            return <span className="next agent"><Icon name={step.waiting ? 'clock' : 'terminal'}/>{step.text}</span>
        case 'command':
            return <span className="next command"><Icon name="chevronRight"/>{step.text}{showKey && <Kbd keys="⏎"/>}</span>
        case 'wait':
            return <span className="next dim">{step.text}</span>
    }
    return <span className="next dim">—</span>
}
