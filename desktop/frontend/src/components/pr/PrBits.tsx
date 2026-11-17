// Small pieces that show one PR's state, shared by the list and the detail pane.
import {Badge} from '../ui/Badge'
import {Icon} from '../ui/Icon'
import {Kbd} from '../ui/Kbd'
import {attention, Job, jobActive, needsYou, NextStep, PR} from '../../lib/pr'

// Filled when it's your turn, hollow when it's theirs.
export function TurnDot({pr, style}: { pr: PR, style?: React.CSSProperties }) {
    return <span style={style} className={'size-2 justify-self-end rounded-full border-[1.5px] ' + (pr.OnMe ? 'border-warn-fg bg-warn-fg' : 'border-faint')} title={(pr.OnMe ? 'Your turn: ' : 'Their turn: ') + pr.TurnWhy}/>
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

// The list's Agent column: the PR's agent, or on the selected row the next step ⏎ runs.
export function AgentLabel({job, step, sel}: { job?: Job, step: NextStep, sel: boolean }) {
    if (job && job.State !== 'stopped' && needsYou(job))
        return <span className={`${NEXT} font-medium text-warn-fg`}><Icon name="thread"/>{job.Name} · needs you</span>
    if (jobActive(job)) return <NextStepLabel step={step}/>
    if (sel && step.kind === 'command') return <NextStepLabel step={step} showKey/>
    if (job) return <span className={`${NEXT} text-dim`}>{job.Name}{job.State === 'failed' ? ' · failed' : ''}</span>
    return <span/>
}
