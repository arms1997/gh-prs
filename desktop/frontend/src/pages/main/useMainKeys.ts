import {useEffect} from 'react'
import {App} from '../../state/AppState'
import {NextStep, Row} from '../../lib/pr'

const ORGANIZE_NAMES: Record<string, string> = {pp: 'pin', ps: 'snooze', ph: 'hide', pm: 'move', pa: 'show all', gg: 'top', ga: 'Claude sessions'}

type Opts = {
    app: App
    rows: Row[]
    cursor: number
    step?: NextStep
    paused: boolean // a dialog or the filter box has the keyboard
    keySeq: string
    setKeySeq: (s: string) => void
    select: (path: string) => void
    toggleGroup: (path: string, open?: boolean) => void
    focusFilter: () => void
    clear: () => void
    openPalette: () => void
    openHelp: () => void
    cycleTab: (delta: number) => void
    openSessions: () => void
    focusComposer: () => void
    organize: (seq: string) => void
}

// useMainKeys is the main page's keyboard: vim-style moves plus multi-key sequences (cc, pp, gg …).
// It returns the hint for a sequence in progress.
export function useMainKeys(o: Opts): string {
    const {app, rows, cursor, keySeq} = o
    const row = rows[cursor]
    const pr = row?.PR
    const commandKeys = (app.state?.Commands ?? []).map(c => c.Key)
    const allKeys = [...commandKeys, ...Object.keys(ORGANIZE_NAMES)]

    useEffect(() => {
        const onKey = (e: KeyboardEvent) => {
            if (e.metaKey && e.key === 'k') {
                e.preventDefault()
                return o.openPalette()
            }
            // Typing in any text field (filter, Claude's message box, a reply) is text, not shortcuts.
            const t = e.target as HTMLElement
            if (t.tagName === 'INPUT' || t.tagName === 'TEXTAREA' || t.isContentEditable) return
            if (o.paused || e.metaKey || e.ctrlKey || e.altKey) return
            const move = (i: number) => {
                const r = rows[Math.max(0, Math.min(rows.length - 1, i))]
                if (r) o.select(r.Path)
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
                        return pr && app.open(pr.URL)
                    case 'r':
                        return void app.refresh()
                    case '/':
                        e.preventDefault()
                        return o.focusFilter()
                    case '?':
                        return o.openHelp()
                    case 'i':
                        if (!pr) return
                        e.preventDefault()
                        return o.focusComposer()
                    case '[':
                        return o.cycleTab(-1)
                    case ']':
                        return o.cycleTab(1)
                    case 'Escape':
                        return o.clear()
                    case 'h':
                    case 'ArrowLeft':
                        return row && !row.PR && o.toggleGroup(row.Path, false)
                    case 'l':
                    case 'ArrowRight':
                        return row && !row.PR && o.toggleGroup(row.Path, true)
                    case 'Enter':
                        if (row && !row.PR) return o.toggleGroup(row.Path)
                        if (pr && o.step?.kind === 'command') return o.step.key === 'o' ? app.open(pr.URL) : app.run(o.step.key, pr.Key)
                        return
                }
            }
            if (k.length !== 1) return
            const seq = keySeq + k
            if (allKeys.includes(seq)) {
                o.setKeySeq('')
                if (seq === 'gg') return move(0)
                if (seq === 'ga') return o.openSessions()
                if (seq === 'pa') return app.setShowAll(s => !s)
                if (!pr) return app.say('Select a PR first', true)
                if (seq in ORGANIZE_NAMES) return o.organize(seq)
                return app.run(seq, pr.Key)
            }
            o.setKeySeq(allKeys.some(x => x.startsWith(seq)) ? seq : '')
        }
        window.addEventListener('keydown', onKey)
        return () => window.removeEventListener('keydown', onKey)
    })

    return keySeq ? allKeys.filter(x => x.startsWith(keySeq)).map(x => {
        const c = app.state?.Commands.find(c => c.Key === x)
        return `${x} ${c?.Name ?? ORGANIZE_NAMES[x]}`
    }).join('  ·  ') : ''
}
