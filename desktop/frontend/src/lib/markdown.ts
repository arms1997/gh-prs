import {marked} from 'marked'
import DOMPurify from 'dompurify'

// PR descriptions are written by other people: render Markdown, then strip anything executable.
export function markdown(src: string): string {
    return DOMPurify.sanitize(marked.parse(src || '', {async: false}) as string)
}
