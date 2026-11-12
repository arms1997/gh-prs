// @vitest-environment jsdom
import {describe, expect, it} from 'vitest'
import {markdown, markdownInline, taskList} from './markdown'

// PR descriptions are written by other people and rendered into the app: nothing in them may run.
describe('markdown sanitizing', () => {
    it.each([
        ['<script>alert(1)</script>hi', /<script/i],
        ['<img src=x onerror="alert(1)">', /onerror/i],
        ['[x](javascript:alert(1))', /javascript:/i],
        ['<iframe src="https://evil.example"></iframe>', /<iframe/i],
        ['<a href="#" onclick="steal()">x</a>', /onclick/i],
    ])('strips %s', (src, bad) => {
        expect(markdown(src)).not.toMatch(bad)
        expect(markdownInline(src)).not.toMatch(bad)
    })
    it('keeps ordinary Markdown', () => {
        expect(markdown('**bold** and [link](https://github.com)')).toContain('<strong>bold</strong>')
        expect(markdown('[link](https://github.com)')).toContain('href="https://github.com"')
    })
})

describe('taskList', () => {
    it('splits the checklist from the rest of the description', () => {
        const {tasks, rest} = taskList('## Testing\nran it\n- [x] autogen files\n- [ ] staging\n  * [X] nested\nmore')
        expect(tasks).toEqual([{done: true, text: 'autogen files'}, {done: false, text: 'staging'}, {done: true, text: 'nested'}])
        expect(rest).toBe('## Testing\nran it\nmore')
    })
    it('leaves plain bullets and empty bodies alone', () => {
        expect(taskList('- not a task\n- [link]').tasks).toEqual([])
        expect(taskList('')).toEqual({tasks: [], rest: ''})
    })
})
