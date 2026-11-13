import {PR} from '../../../lib/pr'
import {Icon} from '../../../components/ui/Icon'
import {BOX_END, BoxList, BoxRow} from '../../../components/ui/BoxList'
import {Heading} from '../../../components/ui/Heading'

const GROUPS = [
    {state: 'fail', title: 'Failing'},
    {state: 'pending', title: 'Running'},
    {state: 'pass', title: 'Passed'},
    {state: 'skipped', title: 'Skipped'},
] as const
const COLOR = {fail: 'text-bad-fg', pending: 'text-info-fg', pass: 'text-ok-fg', skipped: 'text-faint'}

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
                        <Heading>{g.title} <span className="dim text-xs mono">{list.length}</span></Heading>
                        <BoxList>
                            {list.map(c => (
                                <BoxRow key={c.Name + c.URL}>
                                    <span className={`inline-flex shrink-0 ${COLOR[g.state]}`}>
                                        <Icon name={{fail: 'x', pending: 'spinner', pass: 'check', skipped: 'chevronRight'}[g.state] as 'x'}
                                              className={g.state === 'pending' ? 'animate-spin' : undefined}/>
                                    </span>
                                    <span className="min-w-0 truncate">{c.Name}</span>
                                    {c.URL && <a href={c.URL} className={`${BOX_END} text-xs`}>Details <Icon name="external"/></a>}
                                </BoxRow>
                            ))}
                        </BoxList>
                    </section>
                )
            })}
        </>
    )
}
