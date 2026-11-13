import {PR} from '../../../lib/pr'
import {Icon} from '../../../components/ui/Icon'

const GROUPS = [
    {state: 'fail', title: 'Failing'},
    {state: 'pending', title: 'Running'},
    {state: 'pass', title: 'Passed'},
    {state: 'skipped', title: 'Skipped'},
] as const

export function ChecksTab({pr}: { pr: PR }) {
    const checks = pr.Checks ?? []
    if (checks.length === 0) return <p className="dim">No checks on the latest commit.</p>
    return (
        <>
            {GROUPS.map(g => {
                const list = checks.filter(c => c.State === g.state)
                if (!list.length) return null
                return (
                    <section key={g.state}>
                        <h3>{g.title} <span className="dim small mono">{list.length}</span></h3>
                        <div className="boxlist">
                            {list.map(c => (
                                <div key={c.Name + c.URL} className="boxlist-row">
                                    <span className={'check-icon ' + c.State}>
                                        <Icon name={{fail: 'x', pending: 'spinner', pass: 'check', skipped: 'chevronRight'}[g.state] as 'x'}
                                              className={g.state === 'pending' ? 'animate-spin' : undefined}/>
                                    </span>
                                    <span className="ellipsis">{c.Name}</span>
                                    {c.URL && <a href={c.URL} className="boxlist-end small">Details <Icon name="external"/></a>}
                                </div>
                            ))}
                        </div>
                    </section>
                )
            })}
        </>
    )
}
