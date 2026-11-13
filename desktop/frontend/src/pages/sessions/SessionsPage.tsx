import {useCallback, useEffect, useState} from 'react'
import {Sessions} from '../../../wailsjs/go/main/App'
import {main} from '../../../wailsjs/go/models'
import {useApp} from '../../state/AppState'
import {agentCommands, ago, JOB_LABEL, needsYou} from '../../lib/pr'
import {Badge, Tone} from '../../components/ui/Badge'
import {Button} from '../../components/ui/Button'
import {IconName} from '../../components/ui/Icon'
import {Kbd} from '../../components/ui/Kbd'

const TONE: Record<string, [Tone, IconName]> = {
    running: ['info', 'spinner'], waiting: ['info', 'clock'], done: ['ok', 'check'], failed: ['bad', 'x'], stopped: ['neutral', 'x'],
}

const GRID = 'grid grid-cols-[130px_220px_minmax(0,1fr)_80px_56px] items-center gap-3'

// Every agent run across your PRs, newest first: what it was asked, where it got to.
export function SessionsPage() {
    const app = useApp()
    const [jobs, setJobs] = useState<main.JobState[]>()
    const [cur, setCur] = useState(0)
    const load = useCallback(() => { Sessions().then(setJobs).catch(e => app.say(String(e), true)) }, [])
    useEffect(() => {
        load()
        const t = setInterval(load, 3000)
        return () => clearInterval(t)
    }, [load])

    const list = jobs ?? []
    const active = list.filter(j => j.State === 'running' || j.State === 'waiting')
    const finished = list.filter(j => !active.includes(j))
    const ordered = [...active, ...finished]
    const sel = ordered[Math.min(cur, ordered.length - 1)]
    const {stop, takeOver} = agentCommands(app.state?.Commands ?? [])
    const back = () => app.setPage('main')
    const open = (j?: main.JobState) => {
        if (!j) return
        app.setFocus({prKey: j.PRKey, tab: 'claude'})
        back()
    }

    useEffect(() => {
        const onKey = (e: KeyboardEvent) => {
            if (e.metaKey || e.ctrlKey || app.ask || (e.target as HTMLElement).tagName === 'INPUT') return
            const run = (key?: string) => key && sel && app.run(key, sel.PRKey)
            switch (e.key) {
                case 'j': case 'ArrowDown': return setCur(c => Math.min(c + 1, ordered.length - 1))
                case 'k': case 'ArrowUp': return setCur(c => Math.max(c - 1, 0))
                case 'Enter': return open(sel)
                case 'Escape': return back()
                case 'x': return run(stop?.Key)
                case 'a': return run(takeOver?.Key)
                case 'r': return run(sel?.Key)
            }
        }
        window.addEventListener('keydown', onKey)
        return () => window.removeEventListener('keydown', onKey)
    })

    const row = (j: main.JobState) => {
        const i = ordered.indexOf(j)
        // Only the PR's latest run can still be waiting on you; an older question was answered or superseded.
        const asks = app.state?.Jobs[j.PRKey]?.ID === j.ID && !!needsYou(j)
        const [tone, icon] = asks ? ['warn', 'question'] as [Tone, IconName] : TONE[j.State]
        const what = (asks && needsYou(j)) || j.Summary?.Headline || j.Activity?.[j.Activity.length - 1] || j.Result
        return (
            <div key={j.ID} className={`${GRID} h-11 cursor-default rounded-md px-4 text-sm ${i === cur ? 'bg-sel-bg shadow-sel' : 'hover:bg-muted'}`}
                 onClick={() => setCur(i)} onDoubleClick={() => open(j)}>
                <span><Badge tone={tone} icon={icon}>{asks ? 'Needs you' : JOB_LABEL[j.State].split(' ')[0]}</Badge></span>
                <span className="mono small dim ellipsis">{j.Repo.split('/')[1]}#{j.Number}</span>
                <span className="ellipsis"><span className="font-medium">{j.Name}:</span> {what}</span>
                <span className="small dim">{ago(j.Started)}</span>
                <span className="small dim mono">{j.Elapsed}</span>
            </div>
        )
    }

    return (
        <div className="grid h-full grid-rows-[56px_32px_minmax(0,1fr)_32px]">
            <header className="flex items-center gap-3 border-b border-border-100 px-5">
                <Button size="sm" onClick={back}>← PRs <Kbd keys="esc"/></Button>
                <h2 className="m-0 text-lg font-semibold">Claude sessions</h2>
                <span className="dim small">Every agent run across your PRs</span>
            </header>
            <div className={`${GRID} border-b border-border-100 px-6 text-xs font-medium text-dim`}><span>State</span><span>Pull request</span><span>Asked to</span><span>Started</span><span>Time</span></div>
            <main className="scroll-list overflow-y-auto px-2 pt-1 pb-4">
                {jobs === undefined && <p className="dim my-3.5 p-4">Loading…</p>}
                {jobs?.length === 0 && <p className="dim my-3.5 p-4">No agent has run yet.</p>}
                {active.length > 0 && <div className="px-4 pt-3 pb-1 text-xs font-medium text-dim">Active</div>}
                {active.map(row)}
                {finished.length > 0 && <div className="px-4 pt-3 pb-1 text-xs font-medium text-dim">Finished</div>}
                {finished.map(row)}
            </main>
            <footer className="footer">
                <span className="footer-key"><Kbd keys="⏎"/>open</span>
                {takeOver && <span className="footer-key"><Kbd keys="a"/>open in Claude Code</span>}
                {stop && <span className="footer-key"><Kbd keys="x"/>stop</span>}
                <span className="footer-key"><Kbd keys="r"/>run again</span>
                <span className="footer-key"><Kbd keys="esc"/>back</span>
                <span className={'footer-status' + (app.flash?.err ? ' bad' : '')}>{app.flash?.text}</span>
            </footer>
        </div>
    )
}
