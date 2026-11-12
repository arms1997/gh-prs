import {marked} from 'marked'
import DOMPurify from 'dompurify'

// PR descriptions are written by other people: render Markdown, then strip anything executable.
export function markdown(src: string): string {
    return DOMPurify.sanitize(marked.parse(src || '', {async: false}) as string)
}

export function markdownInline(src: string): string {
    return DOMPurify.sanitize(marked.parseInline(src || '', {async: false}) as string)
}

export type Task = { done: boolean, text: string }

const TASK = /^\s*[-*]\s+\[([ xX])\]\s+(.*)$/

// taskList splits a PR body into its "- [ ]" checklist and the rest of the description.
export function taskList(body: string): { tasks: Task[], rest: string } {
    const tasks: Task[] = []
    const rest = (body || '').split('\n').filter(line => {
        const m = TASK.exec(line)
        if (m) tasks.push({done: m[1] !== ' ', text: m[2]})
        return !m
    })
    return {tasks, rest: rest.join('\n')}
}
