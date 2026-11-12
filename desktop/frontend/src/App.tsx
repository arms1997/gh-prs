import {useCallback, useEffect, useMemo, useRef, useState} from 'react'
import {marked} from 'marked'
import DOMPurify from 'dompurify'
import {OpenURL, Organize, Projects, Refresh, Run, State} from '../wailsjs/go/main/App'
import {main} from '../wailsjs/go/models'

type Row = main.RowView
type PR = main.PRView
type Job = main.JobState

// PR descriptions are written by other people: render Markdown, then strip anything executable.
function markdown(src: string): string {
    return DOMPurify.sanitize(marked.parse(src || '', {async: false}) as string)
}

const ciBadge = (ci: string) => ({
    SUCCESS: ['passing', 'ok'], FAILURE: ['failing', 'bad'], ERROR: ['failing', 'bad'],
    PENDING: ['running', 'warn'], EXPECTED: ['running', 'warn'],
} as Record<string, string[]>)[ci] ?? ['no checks', 'dim']

const reviewBadge = (r: string) => ({
    APPROVED: ['approved', 'ok'], CHANGES_REQUESTED: ['changes', 'bad'], REVIEW_REQUIRED: ['review', 'warn'],
} as Record<string, string[]>)[r] ?? ['—', 'dim']

const mergeBadge = (m: string) => ({CONFLICTING: ['conflict', 'bad'], MERGEABLE: ['clean', 'ok']} as Record<string, string[]>)[m] ?? ['—', 'dim']

const SPIN = '⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏'

function Badge({b}: { b: string[] }) {
    return <span className={'badge ' + b[1]}>{b[0]}</span>
}

function AgentCell({job, tick}: { job?: Job, tick: number }) {
    if (!job) return null
    switch (job.State) {
        case 'running':
            return <span className="agent warn">{SPIN[tick % SPIN.length]} {job.Name} {job.Elapsed}</span>
        case 'waiting':
            return <span className="agent info">◷ {job.Name} {job.Elapsed}</span>
        case 'done':
            return <span className="agent ok">✓ {job.Name}</span>
    }
    return <span className="agent bad">✗ {job.Name}</span>
}

function AgentBox({job}: { job: Job }) {
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
                    <div className="bold">{s.Headline}</div>
                    {s.Changes?.map((c, i) => <div key={i} className="ok">+ {c}</div>)}
                    {pushed && (/^no/i.test(pushed)
                        ? <div className="warn">not pushed: {pushed.replace(/^no:?\s*/i, '')}</div>
                        : <div className="ok">↑ pushed {pushed}</div>)}
                    {s.NeedsYou && !/^nothing\.?$/i.test(s.NeedsYou.trim()) && <div className="warn bold">! needs you: {s.NeedsYou}</div>}
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

function Detail({pr, job, commands, run, open, onLink}: {
    pr: PR, job?: Job, commands: main.CommandView[],
    run: (key: string) => void, open: (url: string) => void, onLink: (e: React.MouseEvent) => void
}) {
    const body = useMemo(() => markdown(pr.Body), [pr.Body])
    return (
        <div className="detail" onClick={onLink}>
            <h2><a href={pr.URL}>{pr.Title}</a></h2>
            <div className="meta">
                {pr.Repo}#{pr.Number} · {pr.Author} · <code>{pr.HeadRefName}</code> → <code>{pr.BaseRefName}</code>
            </div>
            <div className="meta">
                <span className="ok">+{pr.Additions}</span> <span className="bad">−{pr.Deletions}</span> · {pr.ChangedFiles} files
                {' '}· <Badge b={ciBadge(pr.CI)}/> <Badge b={reviewBadge(pr.ReviewDecision)}/> <Badge b={mergeBadge(pr.Mergeable)}/>
                {pr.IsDraft && <> <span className="badge dim">draft</span></>}
            </div>
            <div className={pr.OnMe ? 'turn me' : 'turn them'}>{pr.OnMe ? '● your turn' : '○ their turn'}: {pr.TurnWhy}</div>
            {pr.FailingChecks?.length > 0 && (
                <div className="section">
                    <div className="label">Failing checks</div>
                    {pr.FailingChecks.map(c => <div key={c.Name}>✗ {c.URL ? <a href={c.URL}>{c.Name}</a> : c.Name}</div>)}
                </div>
            )}
            {(pr.Labels?.length > 0 || pr.Project || pr.LinearURL) && (
                <div className="section">
                    {pr.Project && <span className="badge info">project: {pr.Project}</span>}
                    {pr.Labels?.map(l => <span key={l} className="badge dim">{l}</span>)}
                    {pr.LinearURL && <a className="badge info" href={pr.LinearURL}>{pr.Linear}</a>}
                </div>
            )}
            {job && <AgentBox job={job}/>}
            <div className="section commands">
                {commands.map(c => (
                    <button key={c.Key} onClick={() => run(c.Key)} title={c.Background ? 'background agent' : ''}>
                        <kbd>{c.Key}</kbd> {c.Name}
                    </button>
                ))}
                <button onClick={() => open(pr.URL)}><kbd>o</kbd> open on GitHub</button>
            </div>
            <div className="body markdown" dangerouslySetInnerHTML={{__html: body || '<p class="dim">No description.</p>'}}/>
        </div>
    )
}

const HELP: [string, string][] = [
    ['j k ↑ ↓', 'move'], ['g g / G', 'top / bottom'], ['h l / enter', 'collapse / expand a group'],
    ['o', 'open on GitHub'], ['r', 'refresh'], ['/', 'filter'], ['esc', 'clear filter, close'],
    ['pp ps ph', 'pin, snooze, hide'], ['pm', 'move to a project'], ['pa', 'show hidden & snoozed'],
    ['cc cf ca cr cm …', 'commands (also buttons in the detail pane)'], ['?', 'this help'],
]

export default function App() {
    const [state, setState] = useState<main.State>()
    const [filter, setFilter] = useState('')
    const [filtering, setFiltering] = useState(false)
    const [collapsed, setCollapsed] = useState<string[]>([])
    const [showAll, setShowAll] = useState(false)
    const [sel, setSel] = useState<string>('') // selected row Path
    const [keySeq, setKeySeq] = useState('')
    const [flash, setFlash] = useState<{ text: string, err: boolean }>()
    const [help, setHelp] = useState(false)
    const [moving, setMoving] = useState<{ pr: PR, value: string, projects: string[] }>()
    const [tick, setTick] = useState(0)
    const filterRef = useRef<HTMLInputElement>(null)
    const selRef = useRef<HTMLDivElement>(null)

    const load = useCallback(() => {
        State(filter, collapsed, showAll).then(setState).catch(e => setFlash({text: String(e), err: true}))
    }, [filter, collapsed, showAll])

    useEffect(() => {
        load()
        const t = setInterval(load, 1000)
        return () => clearInterval(t)
    }, [load])
    useEffect(() => {
        const t = setInterval(() => setTick(x => x + 1), 120)
        return () => clearInterval(t)
    }, [])

    const rows = state?.Rows ?? []
    let cursor = rows.findIndex(r => r.Path === sel)
    if (cursor < 0) cursor = rows.findIndex(r => r.PR) // first PR
    const row: Row | undefined = rows[cursor]
    const pr = row?.PR
    useEffect(() => selRef.current?.scrollIntoView({block: 'nearest'}), [cursor])

    const say = (text: string, err = false) => setFlash({text, err})
    const run = (key: string) => {
        if (!pr) return
        Run(key, pr.Key).then(s => say(s)).catch(e => say(String(e), true)).finally(load)
    }
    const open = (url: string) => OpenURL(url).catch(e => say(String(e), true))
    const organize = (action: string, arg = '') => {
        if (!pr) return
        Organize(action, pr.Key, arg).then(s => say(s)).catch(e => say(String(e), true)).finally(load)
    }
    const toggleGroup = (path: string, open?: boolean) =>
        setCollapsed(c => (open ?? c.includes(path)) ? c.filter(p => p !== path) : [...c, path])
    const onLink = (e: React.MouseEvent) => {
        const a = (e.target as HTMLElement).closest('a')
        if (a) {
            e.preventDefault()
            open(a.getAttribute('href') ?? '')
        }
    }

    const commandKeys = (state?.Commands ?? []).map(c => c.Key)
    const allKeys = [...commandKeys, 'pp', 'ps', 'ph', 'pm', 'pa', 'gg']

    useEffect(() => {
        const onKey = (e: KeyboardEvent) => {
            if (moving || filtering || e.metaKey || e.ctrlKey) return
            if (help) {
                setHelp(false)
                return
            }
            const move = (i: number) => {
                const r = rows[Math.max(0, Math.min(rows.length - 1, i))]
                if (r) setSel(r.Path)
            }
            const k = e.key
            if (!keySeq) {
                switch (k) {
                    case 'j':
                    case 'ArrowDown':
                        e.preventDefault()
                        return move(cursor + 1)
                    case 'k':
                    case 'ArrowUp':
                        e.preventDefault()
                        return move(cursor - 1)
                    case 'G':
                        return move(rows.length - 1)
                    case 'o':
                        return pr && open(pr.URL)
                    case 'r':
                        say('refreshing…')
                        return void Refresh()
                    case '/':
                        e.preventDefault()
                        setFiltering(true)
                        return setTimeout(() => filterRef.current?.focus())
                    case '?':
                        return setHelp(true)
                    case 'Escape':
                        return setFilter('')
                    case 'h':
                    case 'ArrowLeft':
                        return row && !row.PR && toggleGroup(row.Path, false)
                    case 'l':
                    case 'ArrowRight':
                    case 'Enter':
                        return row && !row.PR && toggleGroup(row.Path, true)
                }
            }
            if (k.length !== 1) return
            const seq = keySeq + k
            if (allKeys.includes(seq)) {
                setKeySeq('')
                if (seq === 'gg') return move(0)
                if (seq === 'pa') return setShowAll(s => !s)
                if (!pr) return say('select a PR first', true)
                if (seq === 'pp') return organize('pin')
                if (seq === 'ps') return organize('snooze')
                if (seq === 'ph') return organize('hide')
                if (seq === 'pm') return Projects().then(p => setMoving({pr, value: pr.Project ?? '', projects: p ?? []}))
                return run(seq)
            }
            setKeySeq(allKeys.some(x => x.startsWith(seq)) ? seq : '')
        }
        window.addEventListener('keydown', onKey)
        return () => window.removeEventListener('keydown', onKey)
    })

    const hint = keySeq && allKeys.filter(x => x.startsWith(keySeq)).map(x => {
        const c = state?.Commands.find(c => c.Key === x)
        const name = c?.Name ?? {pp: 'pin', ps: 'snooze', ph: 'hide', pm: 'move', pa: 'show all', gg: 'top'}[x]
        return `${x} ${name}`
    }).join('  ·  ')

    return (
        <div className="app">
            <div className="list">
                <div className="toolbar">
                    <input ref={filterRef} className="filter" placeholder="/ filter by title, repo, branch, label…" value={filter}
                           onFocus={() => setFiltering(true)}
                           onChange={e => setFilter(e.target.value)}
                           onKeyDown={e => {
                               if (e.key === 'Escape') setFilter('')
                               if (e.key === 'Escape' || e.key === 'Enter') {
                                   setFiltering(false);
                                   (e.target as HTMLInputElement).blur()
                               }
                           }}
                           onBlur={() => setFiltering(false)}/>
                    {showAll && <span className="badge warn">showing hidden & snoozed</span>}
                    <button onClick={() => Refresh()}>{state?.Loading ? 'refreshing…' : 'refresh'}</button>
                </div>
                <div className="rows">
                    {rows.map((r, i) => {
                        const isSel = i === cursor
                        if (!r.PR) {
                            const closed = collapsed.includes(r.Path)
                            return (
                                <div key={r.Path} ref={isSel ? selRef : undefined}
                                     className={`group depth${r.Depth}` + (isSel ? ' sel' : '')}
                                     style={{paddingLeft: 12 + r.Depth * 18}}
                                     onClick={() => {
                                         setSel(r.Path)
                                         toggleGroup(r.Path)
                                     }}>
                                    <span className="caret">{closed ? '▸' : '▾'}</span>{r.Title} <span className="count">{r.Count}</span>
                                </div>
                            )
                        }
                        const p = r.PR
                        const job = state?.Jobs[p.Key]
                        return (
                            <div key={r.Path} ref={isSel ? selRef : undefined}
                                 className={'pr' + (isSel ? ' sel' : '') + (p.Hidden || p.Snoozed ? ' faded' : '')}
                                 onClick={() => setSel(r.Path)} onDoubleClick={() => open(p.URL)}>
                                <span className="lead">
                                    {p.Pinned ? '★' : p.Hidden ? '⊘' : p.Snoozed ? 'z' : ''}
                                    <span className={p.OnMe ? 'me' : 'them'}>{p.OnMe ? '●' : '○'}</span>
                                </span>
                                <span className="title" style={{paddingLeft: r.Stack * 16}}>
                                    {r.Stack > 0 && <span className="dim">↳ </span>}{p.Title}
                                </span>
                                <span className="num dim">{p.Repo.split('/')[1]}#{p.Number}</span>
                                <span className="lines"><span className="ok">+{p.Additions}</span> <span className="bad">−{p.Deletions}</span></span>
                                <span className="col"><Badge b={ciBadge(p.CI)}/></span>
                                <span className="col"><Badge b={reviewBadge(p.ReviewDecision)}/></span>
                                <span className="col">{p.Unresolved > 0 ? <span className="badge warn">{p.Unresolved} open</span> : ''}</span>
                                <span className="col"><Badge b={mergeBadge(p.Mergeable)}/></span>
                                <span className="agentcol"><AgentCell job={job} tick={tick}/></span>
                            </div>
                        )
                    })}
                    {rows.length === 0 && <div className="empty dim">{state ? 'No PRs match.' : 'Loading…'}</div>}
                </div>
            </div>
            <div className="pane">
                {pr ? <Detail pr={pr} job={state?.Jobs[pr.Key]} commands={state?.Commands ?? []} run={run} open={open} onLink={onLink}/>
                    : <div className="detail dim">Select a PR.</div>}
            </div>
            <div className="statusbar">
                <span>{hint || 'j/k move · / filter · o open · cc agent · p… organize · ? keys'}</span>
                <span className={(flash?.err ?? state?.IsErr) ? 'bad' : 'dim'}>{flash?.text ?? state?.Status}</span>
            </div>
            {help && (
                <div className="overlay" onClick={() => setHelp(false)}>
                    <div className="modal">
                        <h3>Keys</h3>
                        {HELP.map(([k, d]) => <div key={k} className="helprow"><kbd>{k}</kbd><span>{d}</span></div>)}
                        {state?.Commands.map(c => <div key={c.Key} className="helprow"><kbd>{c.Key}</kbd><span>{c.Name}</span></div>)}
                    </div>
                </div>
            )}
            {moving && (
                <div className="overlay">
                    <form className="modal" onSubmit={e => {
                        e.preventDefault()
                        Organize('move', moving.pr.Key, moving.value).then(s => say(s)).catch(e => say(String(e), true)).finally(load)
                        setMoving(undefined)
                    }}>
                        <h3>Move #{moving.pr.Number} to a project</h3>
                        <input autoFocus list="projects" value={moving.value} placeholder="project name (empty: use its label)"
                               onChange={e => setMoving({...moving, value: e.target.value})}
                               onKeyDown={e => e.key === 'Escape' && setMoving(undefined)}/>
                        <datalist id="projects">{moving.projects.map(p => <option key={p} value={p}/>)}</datalist>
                        <div className="dim small">enter to move · esc to cancel</div>
                    </form>
                </div>
            )}
        </div>
    )
}
