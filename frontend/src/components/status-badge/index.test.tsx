import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'

import type { TicketStatus } from '@/domain/tickets/types'
import { contrastRatio, themeToken } from '@/test/contrast'

import { STATUS_BADGE_CLASSES, StatusBadge } from '.'

const STATUSES: [TicketStatus, string, string][] = [
  ['aberto', 'Aberto', 'aberto'],
  ['em_andamento', 'Em andamento', 'em-andamento'],
  ['resolvido', 'Resolvido', 'resolvido'],
  ['fechado', 'Fechado', 'fechado']
]

describe('StatusBadge (spec 003 AC-1.9)', () => {
  it.each(STATUSES)('%s shows its label on its tint', (status, label) => {
    const html = renderToStaticMarkup(<StatusBadge status={status} />)
    expect(html).toContain(`>${label}</span>`)
    expect(html).toContain(STATUS_BADGE_CLASSES[status])
    expect(html).toContain('rounded-sm')
    const classes = /class="([^"]*)"/.exec(html)?.[1]?.split(' ') ?? []
    expect(classes).not.toContain('bg-primary')
    expect(classes).not.toContain('text-primary-foreground')
  })

  it.each(STATUSES)('%s: the text passes AA on the tint', (_s, _l, token) => {
    const ratio = contrastRatio(
      themeToken(`status-${token}`),
      themeToken(`status-${token}-foreground`)
    )
    expect(ratio).toBeGreaterThanOrEqual(4.5)
  })

  it('the four tints differ', () => {
    const tints = STATUSES.map(([, , token]) => themeToken(`status-${token}`))
    expect(new Set(tints).size).toBe(4)
  })
})
