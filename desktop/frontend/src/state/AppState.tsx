// The app's shared state: the backend snapshot (polled every second), the status message, and
// the actions every page uses. Pages read it with useApp().
import {createContext, useCallback, useContext, useEffect, useState} from 'react'
import {OpenURL, Organize, Refresh, Run, State} from '../../wailsjs/go/main/App'
import {main} from '../../wailsjs/go/models'

type Flash = { text: string, err: boolean }

function useAppState() {
    const [state, setState] = useState<main.State>()
    const [filter, setFilter] = useState('')
    const [showAll, setShowAll] = useState(false)
    const [flash, setFlash] = useState<Flash>()

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
        run: (cmdKey: string, prKey: string) => report(Run(cmdKey, prKey)),
        organize: (action: string, prKey: string, arg = '') => report(Organize(action, prKey, arg)),
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
