import {InputHTMLAttributes, TextareaHTMLAttributes} from 'react'

// A bordered text field that rings on focus.
const FIELD = 'rounded-md border border-input bg-background px-2.5 text-foreground outline-none focus:border-sel-ring'

export function Input(props: InputHTMLAttributes<HTMLInputElement>) {
    return <input className={`${FIELD} h-9`} {...props}/>
}

export function TextArea(props: TextareaHTMLAttributes<HTMLTextAreaElement>) {
    return <textarea className={`${FIELD} min-h-24 resize-y py-2.5 [font:inherit]`} {...props}/>
}
