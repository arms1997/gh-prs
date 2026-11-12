import {AppProvider, useApp} from './state/AppState'
import {MainPage} from './pages/main/MainPage'
import {SessionsPage} from './pages/sessions/SessionsPage'
import {AskDialog} from './features/ask/AskDialog'

// Pages live under pages/, and dialogs any page can open are rendered here.
function Shell() {
    const app = useApp()
    return (
        <>
            {app.page === 'sessions' ? <SessionsPage/> : <MainPage/>}
            {app.ask && <AskDialog key={app.ask.cmd.Key + app.ask.prKey}/>}
        </>
    )
}

export default function App() {
    return <AppProvider><Shell/></AppProvider>
}
