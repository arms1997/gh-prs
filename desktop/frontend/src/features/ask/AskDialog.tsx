import {useState} from 'react'
import {useApp} from '../../state/AppState'
import {Modal} from '../../components/ui/Modal'
import {Kbd} from '../../components/ui/Kbd'
import './ask.css'

// The text a command's Input question asks for, e.g. a reply to Claude. ⏎ sends, ⇧⏎ adds a line.
export function AskDialog() {
    const app = useApp()
    const [text, setText] = useState('')
    if (!app.ask) return null
    const {cmd, prKey} = app.ask
    const close = () => app.setAsk(undefined)
    const send = () => {
        if (!text.trim()) return
        close()
        app.run(cmd.Key, prKey, text)
    }
    return (
        <Modal label={cmd.Input} onClose={close}>
            <form className="ask" onSubmit={e => { e.preventDefault(); send() }}>
                <h3>{cmd.Input}</h3>
                <span className="dim small mono">{prKey.split('/')[1]}</span>
                <textarea autoFocus rows={4} value={text} placeholder="Type your message"
                          onChange={e => setText(e.target.value)}
                          onKeyDown={e => {
                              if (e.key === 'Enter' && !e.shiftKey) {
                                  e.preventDefault()
                                  send()
                              }
                          }}/>
                <div className="ask-foot dim small"><span><Kbd keys="⏎"/> send</span><span>⇧⏎ new line</span><span><Kbd keys="esc"/> cancel</span></div>
            </form>
        </Modal>
    )
}
