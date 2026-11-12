import {Job} from '../../lib/pr'
import {markdownInline} from '../../lib/markdown'

// The agent's last report: its summary block when it wrote one, else its raw final message.
export function LastResult({job}: { job: Job }) {
    const s = job.Summary
    const pushed = s?.Pushed ?? ''
    return (
        <section>
            <h3>{job.State === 'waiting' ? 'Last round' : 'Result'}</h3>
            {s ? (
                <div className="result-card">
                    <span className="result-headline" dangerouslySetInnerHTML={{__html: markdownInline(s.Headline)}}/>
                    {s.Changes?.length > 0 && (
                        <ul>{s.Changes.map((c, i) => <li key={i} dangerouslySetInnerHTML={{__html: markdownInline(c)}}/>)}</ul>
                    )}
                    {pushed && (/^no/i.test(pushed)
                        ? <span className="warn small">Not pushed: {pushed.replace(/^no:?\s*/i, '')}</span>
                        : <span className="ok small">Pushed {pushed}</span>)}
                </div>
            ) : (
                <pre className="result-raw">{job.Result || 'No result yet.'}</pre>
            )}
        </section>
    )
}
