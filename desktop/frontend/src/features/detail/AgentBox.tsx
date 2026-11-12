import {useEffect, useRef} from 'react'
import {Job} from '../../lib/pr'

export function AgentBox({job}: { job: Job }) {
    const s = job.Summary
    const title = {running: 'running', waiting: 'last round ✓ · waiting for the next', done: '✓', failed: '✗'}[job.State]
    const pushed = s?.Pushed ?? ''
    const body = useRef<HTMLDivElement>(null)
    useEffect(() => { // keep the newest activity in view while it runs
        if (job.State === 'running' && body.current) body.current.scrollTop = body.current.scrollHeight
    }, [job.State, job.Activity?.length])
    return (
        <div className={'agentbox ' + job.State}>
            <div className="agentbox-title">Agent · {job.Name} · {title}</div>
            <div className="agentbox-body" ref={body}>
                {job.State === 'running' ? (
                    (job.Activity?.length ? job.Activity : ['starting…']).map((a, i) => <div key={i} className="dim">› {a}</div>)
                ) : s ? (
                    <>
                        <div className="headline">{s.Headline}</div>
                        {s.Changes?.map((c, i) => <div key={i} className="ok">+ {c}</div>)}
                        {pushed && (/^no/i.test(pushed)
                            ? <div className="warn">not pushed: {pushed.replace(/^no:?\s*/i, '')}</div>
                            : <div className="ok">↑ pushed {pushed}</div>)}
                        {s.NeedsYou && !/^nothing\.?$/i.test(s.NeedsYou.trim()) && <div className="warn headline">! needs you: {s.NeedsYou}</div>}
                    </>
                ) : (
                    <pre className="result">{job.Result}</pre>
                )}
            </div>
            <div className="dim small">
                {job.State === 'done' || job.State === 'failed' ? `after ${job.Elapsed}` : `running ${job.Elapsed}`}
                {job.Turns > 0 && ` · ${job.Turns} turns`}{job.Cost > 0 && ` · $${job.Cost.toFixed(2)}`}
            </div>
        </div>
    )
}
