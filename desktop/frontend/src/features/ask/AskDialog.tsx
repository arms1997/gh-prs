import {useState} from 'react'
import {useApp} from '../../state/AppState'
import {Modal} from '../../components/ui/Modal'
import {Kbd} from '../../components/ui/Kbd'
import {TextArea} from '../../components/ui/Input'

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
            <form className="flex w-[520px] flex-col gap-2 px-5.5 py-4.5" onSubmit={e => { e.preventDefault(); send() }}>
                <h3 className="m-0 text-lg font-semibold">{cmd.Input}</h3>
                <span className="text-dim text-xs font-mono">{prKey.split('/')[1]}</span>
                <TextArea autoFocus rows={4} value={text} placeholder="Type your message"
                          onChange={e => setText(e.target.value)}
                          onKeyDown={e => {
                              if (e.key === 'Enter' && !e.shiftKey) {
                                  e.preventDefault()
                                  send()
                              }
                          }}/>
                <div className="text-dim text-xs flex gap-4 [&>span]:inline-flex [&>span]:items-center [&>span]:gap-1"><span><Kbd keys="⏎"/> send</span><span>⇧⏎ new line</span><span><Kbd keys="esc"/> cancel</span></div>
            </form>
        </Modal>
    )
}
