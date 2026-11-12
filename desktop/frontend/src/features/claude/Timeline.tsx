import {useEffect, useRef} from 'react'
import {Job} from '../../lib/pr'
import {Icon, IconName} from '../../components/ui/Icon'

// stepIcon picks an icon from an activity line, e.g. "Bash git push" or "“I'll rebase first”".
function stepIcon(a: string): IconName {
    if (a.startsWith('“')) return 'thread'
    if (/^Bash (git|gh) /.test(a)) return 'merge'
    if (a.startsWith('Bash')) return 'terminal'
    if (/^(Read|Grep|Glob)/.test(a)) return 'eye'
    if (/^(Edit|Write)/.test(a)) return 'stack'
    if (a.startsWith('ScheduleWakeup')) return 'clock'
    return 'chevronRight'
}

// stepText shortens a step: a wakeup repeats the whole loop prompt, long commands get cut.
function stepText(a: string): string {
    if (a.startsWith('ScheduleWakeup')) return 'Scheduled the next round'
    return a.length > 240 ? a.slice(0, 240) + '…' : a
}

// What Claude did: each round's headline for a manage loop, then its recent steps, newest last.
export function Timeline({job}: { job: Job }) {
    const box = useRef<HTMLOListElement>(null)
    useEffect(() => { // keep the newest step in view while it runs
        if (job.State === 'running' && box.current) box.current.scrollTop = box.current.scrollHeight
    }, [job.State, job.Activity?.length])
    const steps = job.Activity ?? []
    return (
        <>
            {job.Rounds?.length > 1 && (
                <section>
                    <h3>Rounds</h3>
                    <ol className="rounds">
                        {job.Rounds.map((r, i) => <li key={i}><span className="mono dim">{i + 1}</span>{r}</li>)}
                    </ol>
                </section>
            )}
            <section>
                <div className="section-head"><h3>What Claude did</h3><span className="dim small">{steps.length} recent steps</span></div>
                {steps.length === 0 ? <p className="dim small">Starting…</p> : (
                    <ol className="timeline" ref={box}>
                        {steps.map((a, i) => (
                            <li key={i} className={i === steps.length - 1 && job.State === 'running' ? 'current' : ''}>
                                <span className="step-icon"><Icon name={stepIcon(a)}/></span>
                                <span className="step-text" title={a}>{stepText(a)}</span>
                            </li>
                        ))}
                    </ol>
                )}
            </section>
        </>
    )
}
