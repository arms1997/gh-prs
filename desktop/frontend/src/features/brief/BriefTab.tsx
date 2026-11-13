import {useEffect, useRef, useState} from 'react'
import {PRBrief} from '../../../wailsjs/go/main/App'
import {main} from '../../../wailsjs/go/models'
import {ago, PR} from '../../lib/pr'
import {markdown} from '../../lib/markdown'
import {Button} from '../../components/ui/Button'
import {Icon} from '../../components/ui/Icon'
import {Kbd} from '../../components/ui/Kbd'
import './brief.css'

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

    if (err) return <p className="bad">{err}</p>
    if (!brief) return null
    if (brief.Running) return <p className="brief-wait dim"><Icon name="spinner" className="animate-spin"/> Claude is reading the reviews, threads and checks…</p>
    if (!brief.Text) return (
        <section className="brief-empty">
            {brief.Err && <p className="bad">{brief.Err}</p>}
            <p className="dim">Claude reads this PR's reviews, open threads, checks and conflicts, and writes what's left
                before it can merge. Takes about 30 seconds and a few cents.</p>
            <Button variant="primary" onClick={() => load(true)}>Write brief <Kbd keys="B"/></Button>
        </section>
    )
    return (
        <section className="brief">
            {brief.Head !== pr.HeadOID && <p className="warn small">Written before the latest push.</p>}
            <div className="brief-text" dangerouslySetInnerHTML={{__html: markdown(brief.Text)}}/>
            <div className="brief-foot small dim">
                <span>Written {ago(brief.At)} · ${brief.Cost.toFixed(2)}</span>
                <Button size="sm" onClick={() => load(true)}>Rewrite <Kbd keys="B"/></Button>
            </div>
        </section>
    )
}
