import {useApp} from '../../state/AppState'
import {agentCommands, Job, NextStep, PR, statusBadges} from '../../lib/pr'
import {DiffStat, TurnDot} from '../../components/pr/PrBits'
import {Badge} from '../../components/ui/Badge'
import {Button} from '../../components/ui/Button'
import {Icon} from '../../components/ui/Icon'
import {Kbd} from '../../components/ui/Kbd'

// The PR page's head: where it is, its title, what you can do now, and whose turn it is.
export function DetailHeader({pr, job, step}: { pr: PR, job?: Job, step: NextStep }) {
    const app = useApp()
    const {takeOver} = agentCommands(app.state?.Commands ?? [])
    return (
        <div className="flex flex-col gap-2.5 select-text">
            <div className="flex min-w-0 items-center gap-2 overflow-hidden text-xs whitespace-nowrap text-dim">
                <span className="font-mono">{pr.Repo.split('/')[1]}#{pr.Number}</span>
                <span aria-hidden="true">·</span><span>{pr.Author}</span>
                <span aria-hidden="true">·</span><span className="min-w-0 truncate"><code>{pr.HeadRefName}</code> → <code>{pr.BaseRefName}</code></span>
                <span aria-hidden="true">·</span><DiffStat pr={pr}/>
                <span aria-hidden="true">·</span><span>{pr.ChangedFiles} files</span>
            </div>
            <div className="flex flex-wrap items-start gap-x-6 gap-y-3">
                <h2 className="m-0 min-w-0 flex-[1_1_420px] text-xl leading-[1.3] font-semibold">{pr.Title}</h2>
                <div className="flex flex-wrap gap-2">
                    <Button onClick={() => app.open(pr.URL)}>Open on GitHub <Kbd keys="o"/></Button>
                    {job && takeOver && <Button onClick={() => app.run(takeOver.Key, pr.Key)}>Take over agent <Kbd keys={takeOver.Key}/></Button>}
                    {step.kind === 'command' && step.key !== 'o' && (
                        <Button variant="primary" onClick={() => app.run(step.key, pr.Key)}><Icon name="terminal" size={14}/>{step.text}<Kbd keys="⏎"/></Button>
                    )}
                </div>
            </div>
            <div className="flex items-center gap-2 text-sm">
                <TurnDot pr={pr}/>
                <span className={pr.OnMe ? 'font-medium' : 'text-dim'}>{pr.OnMe ? 'Your turn' : 'Their turn'}{pr.TurnWhy && `: ${pr.TurnWhy}`}</span>
            </div>
            <div className="flex flex-wrap gap-1.5">
                {statusBadges(pr).map(b => <Badge key={b.text} tone={b.tone} icon={b.icon}>{b.text}</Badge>)}
                {pr.LinearURL && <a href={pr.LinearURL} className="hover:no-underline"><Badge tone="info">{pr.Linear}</Badge></a>}
            </div>
        </div>
    )
}
