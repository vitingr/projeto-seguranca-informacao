import { describe, expect, it } from 'vitest'

import { formatDate, formatDateTime } from '.'

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
