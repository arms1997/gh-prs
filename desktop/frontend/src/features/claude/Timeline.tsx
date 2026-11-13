import {useEffect, useRef} from 'react'
import {Job} from '../../lib/pr'
import {Icon, IconName} from '../../components/ui/Icon'
import {Heading} from '../../components/ui/Heading'

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
                    <Heading>Rounds</Heading>
                    <ol className="m-0 flex list-none flex-col gap-1 p-0 text-sm">
                        {job.Rounds.map((r, i) => <li key={i} className="flex gap-2.5"><span className="font-mono text-dim">{i + 1}</span>{r}</li>)}
                    </ol>
                </section>
            )}
            <section>
                <Heading aside={<span className="text-dim text-xs">{steps.length} recent steps</span>}>What Claude did</Heading>
                {steps.length === 0 ? <p className="text-dim text-xs">Starting…</p> : (
                    <ol className="m-0 max-h-90 list-none overflow-y-auto rounded-lg border border-border-100 px-0 py-1 [--thumb:var(--text-faint)]" ref={box}>
                        {steps.map((a, i) => {
                            const current = i === steps.length - 1 && job.State === 'running'
                            return (
                                <li key={i} className={`flex items-start gap-2.5 px-3 py-[5px] text-[12.5px] ${current ? 'bg-info-bg' : ''}`}>
                                    <span className={`inline-flex size-5 shrink-0 items-center justify-center rounded-full bg-muted ${current ? 'text-info-fg' : 'text-neutral-fg'}`}><Icon name={stepIcon(a)}/></span>
                                    <span className="min-w-0 pt-px font-mono text-xs text-neutral-fg [overflow-wrap:anywhere]" title={a}>{stepText(a)}</span>
                                </li>
                            )
                        })}
                    </ol>
                )}
            </section>
        </>
    )
}
