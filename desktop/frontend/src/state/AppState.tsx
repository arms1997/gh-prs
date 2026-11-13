// The app's shared state: the backend snapshot (polled every second), the status message, the
// current page, and the actions every page uses. Pages read it with useApp().
import {createContext, useCallback, useContext, useEffect, useState} from 'react'
import {OpenURL, Organize, Refresh, Run, SaveColumns, State} from '../../wailsjs/go/main/App'
import {main} from '../../wailsjs/go/models'

type Flash = { text: string, err: boolean }
export type Page = 'main' | 'sessions'
export type DetailTabName = 'overview' | 'claude' | 'checks' | 'files'
// focus asks the main page to select a PR and show one of its tabs (e.g. from the sessions page).
export type Focus = { prKey: string, tab: DetailTabName }
// ask is a command waiting for the text its Input question asks for.
export type Ask = { cmd: main.CommandView, prKey: string }

function useAppState() {
    const [state, setState] = useState<main.State>()
    const [filter, setFilter] = useState('')
    const [showAll, setShowAll] = useState(false)
    const [flash, setFlash] = useState<Flash>()
    const [page, setPage] = useState<Page>('main')
    const [focus, setFocus] = useState<Focus>()
    const [ask, setAsk] = useState<Ask>()

    const say = useCallback((text: string, err = false) => setFlash({text, err}), [])
    // Collapsing groups is done here in the front end, so quick filters and counts see every PR.
    const load = useCallback(() => {
        State(filter, [], showAll).then(setState).catch(e => say(String(e), true))
    }, [filter, showAll, say])

    useEffect(() => {
        load()
        const t = setInterval(load, 1000)
        return () => clearInterval(t)
    }, [load])

    const report = (p: Promise<string>) => p.then(s => say(s)).catch(e => say(String(e), true)).finally(load)
    return {
        state, filter, setFilter, showAll, setShowAll, flash, say, load,
        page, setPage, focus, setFocus, ask, setAsk,
        // run runs a command on a PR. A command with an Input question asks for its text first,
        // unless the text is given.
        run: (cmdKey: string, prKey: string, input?: string) => {
            const cmd = state?.Commands.find(c => c.Key === cmdKey)
            if (cmd?.Input && input === undefined) return setAsk({cmd, prKey})
            report(Run(cmdKey, prKey, input ?? ''))
        },
        organize: (action: string, prKey: string, arg = '') => report(Organize(action, prKey, arg)),
        saveColumns: (cols: Record<string, number>) => SaveColumns(cols).catch(e => say(String(e), true)).finally(load),
        open: (url: string) => OpenURL(url).catch(e => say(String(e), true)),
        refresh: () => {
            say('refreshing…')
            return Refresh()
        },
    }
}

export type App = ReturnType<typeof useAppState>

const Ctx = createContext<App | null>(null)

export function AppProvider({children}: { children: React.ReactNode }) {
    return <Ctx.Provider value={useAppState()}>{children}</Ctx.Provider>
}

export function useApp(): App {
    const app = useContext(Ctx)
    if (!app) throw new Error('useApp outside AppProvider')
    return app
}
