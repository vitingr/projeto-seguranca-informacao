import { describe, expect, it } from 'vitest'

import { formatClock, formatDate, formatDateTime, formatThreadTime } from '.'

describe('formatDateTime', () => {
  it('shows a UTC instant in São Paulo time (UTC-3)', () => {
    expect(formatDateTime('2026-11-25T14:30:00.123456Z')).toBe(
      '25/11/2026 às 11:30'
    )
  })

  it('moves to the previous day when São Paulo is still there', () => {
    expect(formatDateTime('2026-01-01T02:05:00Z')).toBe('31/12/2025 às 23:05')
  })

  it('uses 24 hours with two digits', () => {
    expect(formatDateTime('2026-03-10T12:07:00Z')).toBe('10/03/2026 às 09:07')
    expect(formatDateTime('2026-03-10T23:00:00Z')).toBe('10/03/2026 às 20:00')
  })

  it('returns null for something that is not a date', () => {
    expect(formatDateTime('ontem')).toBeNull()
    expect(formatDateTime('')).toBeNull()
  })
})

describe('formatDate', () => {
  it('shows only the day, in São Paulo time', () => {
    expect(formatDate('2026-01-01T02:05:00Z')).toBe('31/12/2025')
    expect(formatDate('x')).toBeNull()
  })
})

describe('formatThreadTime (spec 003 section 7)', () => {
  const now = '2026-11-26T18:00:00Z' // 26/11/2026 15:00 in São Paulo

  it.each([
    ['the same day: time only', '2026-11-26T17:04:00Z', '14:04'],
    ['earlier this year: day and month', '2026-11-25T17:04:00Z', '25/11 14:04'],
    ['another year: full date', '2025-11-25T17:04:00Z', '25/11/2025 14:04'],
    // 02:30 UTC on the 26th is still the 25th in São Paulo
    ['the day boundary of São Paulo', '2026-11-26T02:30:00Z', '25/11 23:30']
  ])('%s', (_name, iso, want) => {
    expect(formatThreadTime(iso, now)).toBe(want)
  })

  it('returns null for something that is not a date', () => {
    expect(formatThreadTime('ontem', now)).toBeNull()
  })
})

describe('formatClock', () => {
  it('shows the São Paulo time', () => {
    expect(formatClock('2026-11-26T17:12:00Z')).toBe('14:12')
    expect(formatClock('x')).toBeNull()
  })
})
