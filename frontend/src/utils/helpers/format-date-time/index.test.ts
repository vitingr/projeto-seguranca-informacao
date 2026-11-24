import { describe, expect, it } from 'vitest'

import { dayKey, formatDateTime, formatDay, formatTime } from '.'

describe('formatDateTime', () => {
  it('prints Brasília time', () => {
    expect(formatDateTime('2026-11-22T15:04:00Z')).toBe('22/11/2026, 12:04')
  })

  it('prints nothing for an invalid timestamp', () => {
    expect(formatDateTime('not a date')).toBe('')
  })
})

describe('formatTime', () => {
  it('prints only the Brasília time of day', () => {
    expect(formatTime('2026-11-22T15:04:00Z')).toBe('12:04')
  })

  it('prints nothing for an invalid timestamp', () => {
    expect(formatTime('not a date')).toBe('')
  })
})

describe('formatDay', () => {
  it('spells out the Brasília day', () => {
    expect(formatDay('2026-11-22T15:04:00Z')).toBe(
      'domingo, 22 de novembro de 2026'
    )
  })
})

describe('dayKey', () => {
  it('uses the Brasília day, not the UTC one', () => {
    // 01:30 UTC on the 23rd is still 22:30 on the 22nd in Brasília.
    expect(dayKey('2026-11-23T01:30:00Z')).toBe('2026-11-22')
    expect(dayKey('2026-11-23T03:30:00Z')).toBe('2026-11-23')
  })

  it('is empty for an invalid timestamp', () => {
    expect(dayKey('not a date')).toBe('')
  })
})
