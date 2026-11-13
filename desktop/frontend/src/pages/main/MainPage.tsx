import {useEffect, useMemo, useRef, useState} from 'react'
import {Projects} from '../../../wailsjs/go/main/App'
import {useApp} from '../../state/AppState'
import {nextStep, PR, QUICK_FILTERS} from '../../lib/pr'
import {countPRs, visibleRows} from '../../lib/rows'
import {useWidths} from './useWidths'
import {DETAIL_TABS, DetailPane, DetailTab} from '../../features/detail/DetailPane'
import {CommandPalette, PaletteItem} from '../../features/palette/CommandPalette'
import {HelpDialog} from '../../features/organize/HelpDialog'
import {MoveDialog} from '../../features/organize/MoveDialog'
import {TopBar} from './TopBar'
import {FilterBar} from './FilterBar'
import {PrList} from './PrList'
import {Footer} from './Footer'
import {useMainKeys} from './useMainKeys'
import './main.css'

export function MainPage() {
    const app = useApp()
    const [quick, setQuick] = useState('')
    const [collapsed, setCollapsed] = useState<string[]>([])
    const [sel, setSel] = useState('') // selected row Path
    const [keySeq, setKeySeq] = useState('')
    const [filtering, setFiltering] = useState(false)
    const [tab, setTab] = useState<DetailTab>('overview')
    const [dialog, setDialog] = useState<'help' | 'palette' | { move: PR, projects: string[] }>()
    const filterRef = useRef<HTMLInputElement>(null)

    const all = app.state?.Rows ?? []
    const jobs = app.state?.Jobs ?? {}
    const commands = app.state?.Commands ?? []
    const total = countPRs(all, jobs, () => true)
    const counts = useMemo(() => Object.fromEntries(QUICK_FILTERS.map(f => [f.id, countPRs(all, jobs, f.match)])), [all, jobs])
    const rows = visibleRows(all, collapsed, jobs, QUICK_FILTERS.find(f => f.id === quick)?.match)

    // Another page asked to show a PR (e.g. a session): select its row and open the tab.
    useEffect(() => {
        const f = app.focus
        if (!f || !all.length) return
        const r = all.find(r => r.PR?.Key === f.prKey)
        if (r) {
            setQuick('')
            setCollapsed(c => c.filter(p => !r.Path.startsWith(p + '/')))
            setSel(r.Path)
        } else app.say('That PR is no longer in your list', true)
        setTab(f.tab)
        app.setFocus(undefined)
    }, [app.focus, all])

    let cursor = rows.findIndex(r => r.Path === sel)
    if (cursor < 0) cursor = rows.findIndex(r => r.PR) // first PR
    const pr = rows[cursor]?.PR
    const step = pr && nextStep(pr, jobs[pr.Key], commands)

    const toggleGroup = (path: string, open?: boolean) =>
        setCollapsed(c => (open ?? c.includes(path)) ? c.filter(p => p !== path) : [...c, path])
    const organize = (seq: string) => {
        if (!pr) return
        if (seq === 'pm') return Projects().then(p => setDialog({move: pr, projects: p ?? []}))
        app.organize({pp: 'pin', ps: 'snooze', ph: 'hide'}[seq]!, pr.Key)
    }

    const hint = useMainKeys({
        app, rows, cursor, step, keySeq, setKeySeq, toggleGroup, organize,
        paused: filtering || !!dialog || !!app.ask,
        select: setSel,
        focusFilter: () => filterRef.current?.focus(),
        clear: () => { app.setFilter(''); setQuick('') },
        openPalette: () => setDialog('palette'),
        openHelp: () => setDialog('help'),
        openSessions: () => app.setPage('sessions'),
        focusComposer: () => {
            setTab('claude')
            setTimeout(() => document.getElementById('composer')?.focus())
        },
        cycleTab: (d: number) => setTab(t => DETAIL_TABS[(DETAIL_TABS.indexOf(t) + d + DETAIL_TABS.length) % DETAIL_TABS.length]),
    })

    const paletteItems: PaletteItem[] = [
        ...(pr ? [
            ...commands.map(c => ({section: 'This pull request', label: c.Name, hint: c.Background ? 'hand to agent' : undefined,
                keys: c.Key, icon: c.Background ? 'terminal' as const : undefined, run: () => app.run(c.Key, pr.Key)})),
            {section: 'This pull request', label: 'Open in browser', keys: 'o', icon: 'external' as const, run: () => app.open(pr.URL)},
        ] : []),
        ...QUICK_FILTERS.map(f => ({section: 'Filter the list', label: f.label, hint: `${counts[f.id]} PRs`, run: () => setQuick(f.id)})),
        ...(pr ? [
            {section: 'Organize', label: pr.Pinned ? 'Unpin' : 'Pin', keys: 'pp', icon: 'pin' as const, run: () => organize('pp')},
            {section: 'Organize', label: pr.Snoozed ? 'Unsnooze' : 'Snooze until it changes', keys: 'ps', icon: 'moon' as const, run: () => organize('ps')},
            {section: 'Organize', label: pr.Hidden ? 'Unhide' : 'Hide', keys: 'ph', icon: 'eyeOff' as const, run: () => organize('ph')},
            {section: 'Organize', label: 'Move to project', keys: 'pm', icon: 'stack' as const, run: () => organize('pm')},
        ] : []),
        {section: 'Organize', label: app.showAll ? 'Leave hidden & snoozed out' : 'Show hidden & snoozed', keys: 'pa', run: () => app.setShowAll(s => !s)},
        ...(pr ? [{section: 'This pull request', label: 'Message Claude', keys: 'i', icon: 'thread' as const, run: () => {
            setTab('claude')
            setTimeout(() => document.getElementById('composer')?.focus())
        }}] : []),
        {section: 'App', label: 'All Claude sessions', keys: 'ga', icon: 'terminal', run: () => app.setPage('sessions')},
        {section: 'App', label: 'Refresh', keys: 'r', icon: 'refresh', run: () => app.refresh()},
        {section: 'App', label: 'Keyboard shortcuts', keys: '?', run: () => setDialog('help')},
    ]

    const widths = useWidths()
    return (
        <div className="main-page">
            <TopBar ref={filterRef} total={total} onFocus={() => setFiltering(true)} onDone={() => setFiltering(false)}/>
            <FilterBar quick={quick} setQuick={setQuick} counts={counts} total={total}/>
            <div className="split" style={widths.vars}>
                <PrList rows={rows} cursor={cursor} collapsed={collapsed} onSelect={setSel} onToggle={p => toggleGroup(p)}/>
                <div className="detail-wrap">
                    <span {...widths.handle('detail', 'details pane')}/>
                    <DetailPane pr={pr} tab={tab} setTab={setTab} openPalette={() => setDialog('palette')}/>
                </div>
            </div>
            <Footer hint={hint}/>
            {dialog === 'help' && <HelpDialog commands={commands} onClose={() => setDialog(undefined)}/>}
            {dialog === 'palette' && <CommandPalette items={paletteItems} onClose={() => setDialog(undefined)}
                                                     context={pr && `${pr.Repo.split('/')[1]}#${pr.Number}`}/>}
            {typeof dialog === 'object' && <MoveDialog pr={dialog.move} projects={dialog.projects} onClose={() => setDialog(undefined)}
                                                       onMove={v => app.organize('move', dialog.move.Key, v)}/>}
        </div>
    )
}
