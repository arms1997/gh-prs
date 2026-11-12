import {useMemo, useState} from 'react'
import {useApp} from '../../../state/AppState'
import {beforeMerge, Gate, nextStep, PR} from '../../../lib/pr'
import {markdown, markdownInline, taskList} from '../../../lib/markdown'
import {Badge} from '../../../components/ui/Badge'
import {Button} from '../../../components/ui/Button'
import {Icon} from '../../../components/ui/Icon'
import {Kbd} from '../../../components/ui/Kbd'

export function OverviewTab({pr, showChecks, openPalette}: { pr: PR, showChecks: () => void, openPalette: () => void }) {
    const app = useApp()
    const step = nextStep(pr, app.state?.Jobs[pr.Key], app.state?.Commands ?? [])
    const {tasks, rest} = useMemo(() => taskList(pr.Body), [pr.Body])
    const gates = beforeMerge(pr)
    const done = gates.filter(g => g.state === 'done').length

    const act = (g: Gate) => g.action === 'checks' ? showChecks() : g.action === 'ready' ? app.run('R', pr.Key) : app.open(pr.URL)
    return (
        <>
            <section className="gates">
                <div className="section-head"><h3>Before merge</h3><span className="dim small">{done} of {gates.length} done</span></div>
                <div className="progress"><div style={{width: `${(done / gates.length) * 100}%`}}/></div>
                {gates.map(g => (
                    <div key={g.title} className="gate">
                        <GateIcon state={g.state}/>
                        <div className="gate-text"><span className="gate-title">{g.title}</span><span className="dim small">{g.detail}</span></div>
                        {g.action && g.state !== 'done' && (
                            <button type="button" className="link-btn" onClick={() => act(g)}>
                                {{checks: 'See checks', github: 'Open on GitHub', ready: 'Mark ready'}[g.action]}
                            </button>
                        )}
                    </div>
                ))}
            </section>

            <div className="actions">
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
                    <div className="section-head"><h3>PR checklist</h3><span className="dim small">{tasks.filter(t => t.done).length} of {tasks.length}</span></div>
                    <ul className="tasks">
                        {tasks.map((t, i) => (
                            <li key={i} className={t.done ? 'done' : ''}>
                                <span className="task-box">{t.done && <Icon name="check" size={10}/>}</span>
                                <span dangerouslySetInnerHTML={{__html: markdownInline(t.text)}}/>
                            </li>
                        ))}
                    </ul>
                </section>
            )}

            {(pr.Labels?.length ?? 0) > 0 && (
                <section>
                    <h3>Labels</h3>
                    <div className="detail-badges">{pr.Labels.map(l => <Badge key={l} tone="neutral">{l}</Badge>)}</div>
                </section>
            )}

            <Description md={rest}/>
        </>
    )
}

function GateIcon({state}: { state: Gate['state'] }) {
    if (state === 'pending') return <span className="gate-icon pending"/>
    return <span className={'gate-icon ' + state}><Icon name={state === 'done' ? 'check' : 'x'}/></span>
}

function Description({md}: { md: string }) {
    const [full, setFull] = useState(false)
    const html = useMemo(() => markdown(md), [md])
    const long = md.length > 1200
    return (
        <section>
            <h3>Description</h3>
            <div className={'markdown' + (long && !full ? ' clipped' : '')}
                 dangerouslySetInnerHTML={{__html: md.trim() ? html : '<p class="dim">No description.</p>'}}/>
            {long && <button type="button" className="link-btn" onClick={() => setFull(f => !f)}>{full ? 'Show less' : 'Show full description'}</button>}
        </section>
    )
}
