import {Job} from '../../lib/pr'
import {Heading} from '../../components/ui/Heading'
import {markdownInline} from '../../lib/markdown'

// The agent's last report: its summary block when it wrote one, else its raw final message.
export function LastResult({job}: { job: Job }) {
    const s = job.Summary
    const pushed = s?.Pushed ?? ''
    return (
        <section>
            <Heading>{job.State === 'waiting' ? 'Last round' : 'Result'}</Heading>
            {s ? (
                <div className="md-inline flex flex-col gap-1.5 rounded-lg bg-muted p-3 text-sm leading-normal">
                    <span className="font-medium" dangerouslySetInnerHTML={{__html: markdownInline(s.Headline)}}/>
                    {s.Changes?.length > 0 && (
                        <ul className="m-0 pl-4.5 text-neutral-fg">{s.Changes.map((c, i) => <li key={i} dangerouslySetInnerHTML={{__html: markdownInline(c)}}/>)}</ul>
                    )}
                    {pushed && (/^no/i.test(pushed)
                        ? <span className="text-warn-fg text-xs">Not pushed: {pushed.replace(/^no:?\s*/i, '')}</span>
                        : <span className="text-ok-fg text-xs">Pushed {pushed}</span>)}
                </div>
            ) : (
                <pre className="m-0 max-h-60 overflow-y-auto rounded-lg bg-muted p-3 text-xs whitespace-pre-wrap">{job.Result || 'No result yet.'}</pre>
            )}
        </section>
    )
}
