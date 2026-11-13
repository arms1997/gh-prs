import {useEffect, useState} from 'react'
import {Files} from '../../../../wailsjs/go/main/App'
import {main} from '../../../../wailsjs/go/models'
import {PR} from '../../../lib/pr'
import {BOX_END, BoxList, BoxRow} from '../../../components/ui/BoxList'
import {Heading} from '../../../components/ui/Heading'

// Fetched files, by PR and head commit, so revisiting a PR doesn't call GitHub again.
const cache = new Map<string, main.FileChange[]>()

export function FilesTab({pr}: { pr: PR }) {
    const id = pr.Key + '@' + pr.HeadOID
    const [files, setFiles] = useState(cache.get(id))
    const [err, setErr] = useState('')
    useEffect(() => {
        setFiles(cache.get(id))
        setErr('')
        if (cache.has(id)) return
        let live = true
        Files(pr.Key).then(f => {
            cache.set(id, f ?? [])
            if (live) setFiles(f ?? [])
        }).catch(e => live && setErr(String(e)))
        return () => { live = false }
    }, [id])

    if (err) return <p className="bad">{err}</p>
    if (!files) return <p className="dim">Loading files…</p>
    if (!files.length) return <p className="dim">No files changed.</p>
    return (
        <section>
            <Heading aside={<a href={pr.URL + '/files'} className="text-xs">Review on GitHub</a>}>{files.length} files</Heading>
            <BoxList>
                {files.map(f => {
                    const slash = f.Path.lastIndexOf('/')
                    return (
                        <BoxRow key={f.Path} className="mono text-xs" title={f.Path}>
                            <span className="min-w-0 truncate"><span className="dim">{f.Path.slice(0, slash + 1)}</span>{f.Path.slice(slash + 1)}</span>
                            <span className={BOX_END}><span className="ok">+{f.Additions}</span> <span className="bad">−{f.Deletions}</span></span>
                        </BoxRow>
                    )
                })}
            </BoxList>
        </section>
    )
}
