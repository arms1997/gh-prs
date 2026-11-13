import {useEffect, useRef, useState} from 'react'
import {PRBrief} from '../../../wailsjs/go/main/App'
import {main} from '../../../wailsjs/go/models'
import {ago, PR} from '../../lib/pr'
import {markdown} from '../../lib/markdown'
import {Button} from '../../components/ui/Button'
import {Icon} from '../../components/ui/Icon'
import {Kbd} from '../../components/ui/Kbd'

// The PR brief: Claude's account of what's left, written on request since it costs a Claude call.
// The backend keeps it per PR. B can start one from anywhere, so the open tab polls (a local read).
export function BriefTab({pr}: { pr: PR }) {
    const [brief, setBrief] = useState<main.Brief>()
    const [err, setErr] = useState('')
    const key = useRef(pr.Key)
    const load = (write = false) => {
        const k = pr.Key
        PRBrief(k, write).then(b => k === key.current && setBrief(b)).catch(e => k === key.current && setErr(String(e)))
    }
    useEffect(() => {
        key.current = pr.Key
        setBrief(undefined)
        setErr('')
        load()
    }, [pr.Key])
    useEffect(() => {
        const t = setInterval(() => load(), 2000)
        return () => clearInterval(t)
    }, [pr.Key])

    if (err) return <p className="text-bad-fg">{err}</p>
    if (!brief) return null
    if (brief.Running) return <p className="text-dim flex items-center gap-2"><Icon name="spinner" className="animate-spin"/> Claude is reading the reviews, threads and checks…</p>
    if (!brief.Text) return (
        <section className="flex flex-col items-start gap-3">
            {brief.Err && <p className="text-bad-fg">{brief.Err}</p>}
            <p className="text-dim">Claude reads this PR's reviews, open threads, checks and conflicts, and writes what's left
                before it can merge. Takes about 30 seconds and a few cents.</p>
            <Button variant="primary" onClick={() => load(true)}>Write brief <Kbd keys="B"/></Button>
        </section>
    )
    return (
        <section>
            {brief.Head !== pr.HeadOID && <p className="text-warn-fg text-xs">Written before the latest push.</p>}
            <div className="brief-text text-sm leading-[1.55] select-text" dangerouslySetInnerHTML={{__html: markdown(brief.Text)}}/>
            <div className="text-xs text-dim mt-4 flex items-center justify-between border-t border-border-100 pt-3">
                <span>Written {ago(brief.At)} · ${brief.Cost.toFixed(2)}</span>
                <Button size="sm" onClick={() => load(true)}>Rewrite <Kbd keys="B"/></Button>
            </div>
        </section>
    )
}
