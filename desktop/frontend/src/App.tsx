import {AppProvider} from './state/AppState'
import {MainPage} from './pages/main/MainPage'

// Pages live under pages/; the main page is the only one so far.
export default function App() {
    return (
        <AppProvider>
            <MainPage/>
        </AppProvider>
    )
}
