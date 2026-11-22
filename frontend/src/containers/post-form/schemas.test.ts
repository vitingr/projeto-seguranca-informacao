import { describe, expect, it } from 'vitest'

import { POST_MESSAGES, postSchema } from './schemas'

const firstError = (title: string, body: string) => {
  const result = postSchema.safeParse({ body, title })
  return result.success ? null : result.error.issues.map(i => i.message)
}

// 005 S-2: the same limits as the backend, in code points after trimming.
describe('postSchema', () => {
  it.each([
    ['simple', 'Olá', 'Corpo'],
    ['title at 120', 'a'.repeat(120), 'b'],
    ['title at 120 multibyte', '😀'.repeat(120), 'b'],
    ['title at 120 after trimming', ` ${'a'.repeat(120)} `, 'b'],
    ['body at 2000', 'a', 'b'.repeat(2000)],
    ['body with line breaks and tab', 'a', 'linha 1\r\nlinha 2\n\tfim'],
    ['HTML is just text', '<b>x</b>', '<img src=x onerror="alert(1)">']
  ])('accepts %s', (_, title, body) => {
    expect(firstError(title, body)).toBeNull()
  })

  it.each([
    ['blank title', '   ', 'b', POST_MESSAGES.titleEmpty],
    ['title at 121', 'a'.repeat(121), 'b', POST_MESSAGES.titleTooLong],
    ['title with a line break', 'a\nb', 'b', POST_MESSAGES.titleControl],
    ['title with NUL', 'a\u0000b', 'b', POST_MESSAGES.titleControl],
    ['empty body', 'a', '', POST_MESSAGES.bodyEmpty],
    ['whitespace body', 'a', '\n \t', POST_MESSAGES.bodyEmpty],
    ['body at 2001', 'a', 'b'.repeat(2001), POST_MESSAGES.bodyTooLong],
    ['body with escape', 'a', 'b\u001bc', POST_MESSAGES.bodyControl]
  ])('rejects %s', (_, title, body, message) => {
    expect(firstError(title, body)).toContain(message)
  })
})
