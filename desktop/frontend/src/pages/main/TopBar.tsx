import {forwardRef} from 'react'
import {useApp} from '../../state/AppState'
import {Icon} from '../../components/ui/Icon'
import {Kbd} from '../../components/ui/Kbd'
import {Button} from '../../components/ui/Button'

export const TopBar = forwardRef<HTMLInputElement, { total: number, onFocus: () => void, onDone: () => void, onBrief: () => void }>(
    function TopBar({total, onFocus, onDone, onBrief}, ref) {
        const app = useApp()
        return (
            <header className="flex items-center gap-3 border-b border-border-100 px-4">
                <span className="font-semibold">gh-prs</span>
                <span className="dim text-xs">{total} open PRs</span>
                <label className="mx-auto flex h-8 max-w-[560px] flex-1 items-center gap-2 rounded-md border border-input pr-2 pl-2.5 text-dim focus-within:border-sel-ring">
                    <Icon name="search" size={14}/>
                    <span className="sr-only">Filter pull requests</span>
                    <input ref={ref} className="min-w-0 flex-1 border-0 bg-transparent text-sm text-foreground outline-none" placeholder="Filter by title, repo, branch, label" value={app.filter}
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
