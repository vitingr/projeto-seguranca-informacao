import { describe, expect, it } from 'vitest'

import {
  AUDIT_EVENT_LABELS,
  AUDIT_EVENT_TYPES,
  isAuditEventType,
  parseAuditTypeParam
} from '.'

describe('audit events', () => {
  it('has a label for every event', () => {
    expect(Object.keys(AUDIT_EVENT_LABELS).sort()).toEqual(
      [...AUDIT_EVENT_TYPES].sort()
    )
  })

  it.each([
    ['login_failed', 'login_failed'],
    ['post_removed', 'post_removed'],
    [undefined, null],
    ['', null],
    ['LOGIN_FAILED', null],
    ['login_failed ', null],
    ['toString', null],
    ['constructor', null],
    [['login_failed', 'logout'], null]
  ] as const)('parseAuditTypeParam(%j) = %j', (raw, expected) => {
    expect(parseAuditTypeParam(raw as string | string[] | undefined)).toBe(
      expected
    )
  })

  it('isAuditEventType rejects non-strings', () => {
    expect(isAuditEventType(1)).toBe(false)
    expect(isAuditEventType(null)).toBe(false)
  })
})
