import {useMemo, useState} from 'react'
import {useApp} from '../../../state/AppState'
import {beforeMerge, Gate, nextStep, PR} from '../../../lib/pr'
import {markdown, markdownInline, taskList} from '../../../lib/markdown'
import {Badge} from '../../../components/ui/Badge'
import {Button} from '../../../components/ui/Button'
import {Icon} from '../../../components/ui/Icon'
import {Kbd} from '../../../components/ui/Kbd'
import {Heading} from '../../../components/ui/Heading'

const LINK_BTN = 'cursor-pointer border-0 bg-transparent p-0 text-xs font-medium whitespace-nowrap text-link hover:underline'

export function OverviewTab({pr, showChecks, openPalette}: { pr: PR, showChecks: () => void, openPalette: () => void }) {
    const app = useApp()
    const step = nextStep(pr, app.state?.Jobs[pr.Key], app.state?.Commands ?? [])
    const {tasks, rest} = useMemo(() => taskList(pr.Body), [pr.Body])
    const gates = beforeMerge(pr)
    const done = gates.filter(g => g.state === 'done').length

    const act = (g: Gate) => g.action === 'checks' ? showChecks() : g.action === 'ready' ? app.run('R', pr.Key) : app.open(pr.URL)
    return (
        <>
            <section>
                <Heading aside={<span className="dim text-xs">{done} of {gates.length} done</span>}>Before merge</Heading>
                <div className="mb-1 h-1 overflow-hidden rounded-full bg-muted"><div className="h-full bg-primary transition-[width] duration-300" style={{width: `${(done / gates.length) * 100}%`}}/></div>
                {gates.map(g => (
                    <div key={g.title} className="flex items-start gap-3 border-b border-border-100 py-2.5">
                        <GateIcon state={g.state}/>
                        <div className="flex min-w-0 flex-1 flex-col gap-0.5"><span className="text-base font-medium">{g.title}</span><span className="dim text-xs">{g.detail}</span></div>
                        {g.action && g.state !== 'done' && (
                            <button type="button" className={LINK_BTN} onClick={() => act(g)}>
                                {{checks: 'See checks', github: 'Open on GitHub', ready: 'Mark ready'}[g.action]}
                            </button>
                        )}
                    </div>
                ))}
            </section>

            <div className="flex flex-wrap gap-2">
                {step.kind === 'command' && step.key !== 'o' && (
                    <Button variant="primary" onClick={() => app.run(step.key, pr.Key)}>
                        <Icon name="terminal" size={14}/>{step.text}<Kbd keys={step.key}/>
                    </Button>
                )}
                <Button onClick={() => app.open(pr.URL)}>Open in browser <Kbd keys="o"/></Button>
                <Button onClick={openPalette}>All commands <Kbd keys="⌘K"/></Button>
            </div>

            {tasks.length > 0 && (
                <section>
                    <Heading aside={<span className="dim text-xs">{tasks.filter(t => t.done).length} of {tasks.length}</span>}>PR checklist</Heading>
                    <ul className="m-0 flex list-none flex-col gap-2 p-0">
                        {tasks.map((t, i) => (
                            <li key={i} className={`flex gap-2 text-sm leading-[1.45] ${t.done ? 'text-dim' : ''}`}>
                                <span className={`mt-0.5 inline-flex size-3.5 shrink-0 items-center justify-center rounded-sm border-[1.5px] ${t.done ? 'border-primary bg-primary text-[#fff]' : 'border-border-200'}`}>{t.done && <Icon name="check" size={10}/>}</span>
                                <span className="md-inline" dangerouslySetInnerHTML={{__html: markdownInline(t.text)}}/>
                            </li>
                        ))}
                    </ul>
                </section>
            )}

            {(pr.Labels?.length ?? 0) > 0 && (
                <section>
                    <Heading>Labels</Heading>
                    <div className="flex flex-wrap gap-1.5">{pr.Labels.map(l => <Badge key={l} tone="neutral">{l}</Badge>)}</div>
                </section>
            )}

            <Description md={rest}/>
        </>
    )
}

function GateIcon({state}: { state: Gate['state'] }) {
    if (state === 'pending') return <span className="inline-flex size-5 shrink-0 items-center justify-center rounded-full border-[1.5px] border-dashed border-faint"/>
    return <span className={`inline-flex size-5 shrink-0 items-center justify-center rounded-full ${state === 'done' ? 'bg-ok-bg text-ok-fg' : 'bg-bad-bg text-bad-fg'}`}><Icon name={state === 'done' ? 'check' : 'x'}/></span>
}

function Description({md}: { md: string }) {
    const [full, setFull] = useState(false)
    const html = useMemo(() => markdown(md), [md])
    const long = md.length > 1200
    return (
        <section>
            <Heading>Description</Heading>
            <div className={`markdown text-sm leading-[1.55] text-neutral-fg ${long && !full ? 'max-h-[280px] overflow-hidden [mask-image:linear-gradient(to_bottom,#000_75%,transparent)]' : ''}`}
                 dangerouslySetInnerHTML={{__html: md.trim() ? html : '<p class="dim">No description.</p>'}}/>
            {long && <button type="button" className={LINK_BTN} onClick={() => setFull(f => !f)}>{full ? 'Show less' : 'Show full description'}</button>}
        </section>
    )
}
