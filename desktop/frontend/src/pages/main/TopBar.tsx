import {forwardRef} from 'react'
import {useApp} from '../../state/AppState'
import {Icon} from '../../components/ui/Icon'
import {Kbd} from '../../components/ui/Kbd'
import {Button} from '../../components/ui/Button'

export const TopBar = forwardRef<HTMLInputElement, { total: number, onFocus: () => void, onDone: () => void, onBrief: () => void }>(
    function TopBar({total, onFocus, onDone, onBrief}, ref) {
        const app = useApp()
        return (
            <header className="topbar">
                <span className="topbar-title">gh-prs</span>
                <span className="dim small">{total} open PRs</span>
                <label className="search">
                    <Icon name="search" size={14}/>
                    <span className="sr-only">Filter pull requests</span>
                    <input ref={ref} placeholder="Filter by title, repo, branch, label" value={app.filter}
                           onFocus={onFocus} onBlur={onDone}
                           onChange={e => app.setFilter(e.target.value)}
                           onKeyDown={e => {
                               if (e.key === 'Escape') app.setFilter('')
                               if (e.key === 'Escape' || e.key === 'Enter') (e.target as HTMLInputElement).blur()
                           }}/>
                    <Kbd keys="/"/>
                </label>
                <Button onClick={onBrief}>
                    Daily brief <Kbd keys="b"/>
                </Button>
                <Button aria-label="Refresh" onClick={() => app.refresh()}>
                    <Icon name="refresh" size={14} className={app.state?.Loading ? 'animate-spin' : undefined}/>
                    <Kbd keys="r"/>
                </Button>
            </header>
        )
    })
