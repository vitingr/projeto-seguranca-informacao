import { createElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'

import type { AuditEvent } from '@/domain/audit/types'

import { AuditTable } from '.'

const event = (overrides: Partial<AuditEvent> = {}): AuditEvent => ({
  actor: { login: 'admin', userId: 1 },
  createdAt: '2026-11-22T15:04:00Z',
  event: 'post_removed',
  id: 1,
  ip: '203.0.113.7',
  target: { id: 9, login: null, type: 'post' },
  ...overrides
})

const render = (events: AuditEvent[]) =>
  renderToStaticMarkup(createElement(AuditTable, { events }))

describe('AuditTable', () => {
  it('shows the event label, actor, target, IP and date', () => {
    const html = render([event()])
    expect(html).toContain('Post removido')
    expect(html).toContain('admin')
    expect(html).toContain('Post #9')
    expect(html).toContain('203.0.113.7')
    expect(html).toContain('22/11/2026, 12:04')
  })

  it('shows dashes for a failed login with no actor, target or IP', () => {
    const html = render([
      event({ actor: null, event: 'login_failed', ip: null, target: null })
    ])
    expect(html).toContain('Falha de login')
    expect(html.match(/>—</g)).toHaveLength(3)
  })

  it('names a user target by login, or by id when it was deleted', () => {
    expect(
      render([event({ target: { id: 2, login: 'maria', type: 'user' } })])
    ).toContain('maria')
    expect(
      render([event({ target: { id: 2, login: null, type: 'user' } })])
    ).toContain('Usuário #2 (excluído)')
    expect(render([event({ actor: { login: null, userId: 5 } })])).toContain(
      'Usuário #5 (excluído)'
    )
  })

  it('renders logins as text', () => {
    const html = render([
      event({ actor: { login: '<img src=x onerror=alert(1)>', userId: 1 } })
    ])
    expect(html).not.toMatch(/<img/)
  })
})
