import { describe, expect, it } from 'vitest'

import { formatDateTime } from '.'

describe('formatDateTime', () => {
  it('prints Brasília time', () => {
    expect(formatDateTime('2026-11-22T15:04:00Z')).toBe('22/11/2026, 12:04')
  })

  it('prints nothing for an invalid timestamp', () => {
    expect(formatDateTime('not a date')).toBe('')
  })
})
