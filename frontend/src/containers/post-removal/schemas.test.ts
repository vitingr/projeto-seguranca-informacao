import { describe, expect, it } from 'vitest'

import { REMOVAL_MESSAGES, removalSchema } from './schemas'

const errors = (reason: string) => {
  const result = removalSchema.safeParse({ reason })
  return result.success ? null : result.error.issues.map(i => i.message)
}

// 006 D-2: the same limits as the backend, in code points after trimming.
describe('removalSchema', () => {
  it.each([
    ['simple', 'Spam.'],
    ['at 500', 'a'.repeat(500)],
    ['at 500 multibyte', '😀'.repeat(500)],
    ['at 500 after trimming', ` ${'a'.repeat(500)} `],
    ['line breaks and tab', 'linha 1\r\nlinha 2\tfim'],
    ['HTML is just text', '<img src=x onerror="alert(1)">']
  ])('accepts %s', (_, reason) => {
    expect(errors(reason)).toBeNull()
  })

  it.each([
    ['empty', '', REMOVAL_MESSAGES.reasonEmpty],
    ['whitespace only', ' \n\t ', REMOVAL_MESSAGES.reasonEmpty],
    ['at 501', 'a'.repeat(501), REMOVAL_MESSAGES.reasonTooLong],
    ['NUL', 'spam\u0000', REMOVAL_MESSAGES.reasonControl],
    ['escape', 'spam\u001b[31m', REMOVAL_MESSAGES.reasonControl]
  ])('rejects %s', (_, reason, message) => {
    expect(errors(reason)).toContain(message)
  })
})
