import {ReactNode} from 'react'
import {useApp} from '../../state/AppState'
import {Job, needsYou, PR} from '../../lib/pr'
import {markdownInline} from '../../lib/markdown'
import {Badge, Tone} from '../../components/ui/Badge'
import {Button} from '../../components/ui/Button'
import {Heading} from '../../components/ui/Heading'
import {Kbd} from '../../components/ui/Kbd'
import {AgentStatus} from '../claude/AgentStatus'
import {NeedsYouCard} from '../claude/NeedsYouCard'

const LINK_BTN = 'cursor-pointer border-0 bg-transparent p-0 text-xs font-medium whitespace-nowrap text-link hover:underline'
const CHECKS: { state: string, label: string, tone: Tone }[] = [
    {state: 'fail', label: 'failing', tone: 'bad'}, {state: 'pending', label: 'running', tone: 'info'},
    {state: 'pass', label: 'passed', tone: 'ok'}, {state: 'skipped', label: 'skipped', tone: 'neutral'},
]

// The Overview's right column: the PR's agent and its question, a checks summary, and the PR's facts.
export function PrRail({pr, job, stackedOn, showTab}: { pr: PR, job?: Job, stackedOn: number, showTab: (t: 'claude' | 'checks') => void }) {
    const app = useApp()
    const question = needsYou(job)
    const checks = pr.Checks ?? []
    return (
        <aside aria-label="About this PR" className="flex flex-col gap-6">
            <section className="flex flex-col gap-3">
                <Heading aside={<button type="button" className={LINK_BTN} onClick={() => showTab('claude')}>Claude tab</button>}>Agent</Heading>
                {job ? <>
                    <AgentStatus pr={pr} job={job}/>
                    {job.Summary?.Headline && <p className="md-inline m-0 text-sm leading-normal" dangerouslySetInnerHTML={{__html: markdownInline(job.Summary.Headline)}}/>}
                    {question && <NeedsYouCard pr={pr} question={question} options={job.Summary?.Options ?? []}/>}
                </> : <>
                    <p className="m-0 text-xs text-dim">No agent has worked on this PR yet.</p>
                    <div className="flex flex-wrap gap-1.5">
                        {(app.state?.Commands ?? []).filter(c => c.Background && !c.Input && !c.Session && !c.Confirm).map(c => (
                            <Button key={c.Key} size="sm" onClick={() => app.run(c.Key, pr.Key)}>{c.Name} <Kbd keys={c.Key}/></Button>
                        ))}
                    </div>
                </>}
            </section>
            <section>
                <Heading aside={<button type="button" className={LINK_BTN} onClick={() => showTab('checks')}>All checks</button>}>Checks</Heading>
                {checks.length ? (
                    <div className="flex flex-wrap gap-1.5">
                        {CHECKS.map(c => [c, checks.filter(k => k.State === c.state).length] as const).filter(([, n]) => n > 0)
                            .map(([c, n]) => <Badge key={c.state} tone={c.tone}>{n} {c.label}</Badge>)}
                    </div>
                ) : <p className="m-0 text-xs text-dim">No checks on the latest commit.</p>}
            </section>
            <section>
                <Heading>Details</Heading>
                <dl className="m-0 grid grid-cols-[88px_minmax(0,1fr)] gap-x-3 gap-y-2 text-sm">
                    <Fact name="Project">{pr.Project || <None/>}</Fact>
                    <Fact name="Labels">{pr.Labels?.length ? <span className="flex flex-wrap gap-1">{pr.Labels.map(l => <Badge key={l} tone="neutral">{l}</Badge>)}</span> : <None/>}</Fact>
                    <Fact name="Stack">{stackedOn ? `Stacked on #${stackedOn}` : <None text="Not stacked"/>}</Fact>
                </dl>
            </section>
        </aside>
    )
}

function Fact({name, children}: { name: string, children: ReactNode }) {
    return <><dt className="text-dim">{name}</dt><dd className="m-0 min-w-0 [overflow-wrap:anywhere]">{children}</dd></>
}

const None = ({text = 'None'}: { text?: string }) => <span className="text-dim">{text}</span>
