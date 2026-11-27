import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'

import { readTicketTab } from '@/domain/ticket-query'

import { TicketTabs, tabHref } from '.'

describe('readTicketTab and tabHref (spec 003 AC-2.3)', () => {
  it('reads only "conversa"; anything else is "detalhes"', () => {
    expect(readTicketTab('conversa')).toBe('conversa')
    for (const value of ['detalhes', 'x', '', undefined, ['conversa']]) {
      expect(readTicketTab(value)).toBe('detalhes')
    }
  })

  it('writes ?aba=conversa, leaves it out for "detalhes" and keeps ?de=', () => {
    const base =
      'https://localhost:3000/usuarios/tickets/7?de=%2Fusuarios%3Fstatus%3Daberto'
    expect(tabHref(base, 'conversa')).toBe(
      '/usuarios/tickets/7?de=%2Fusuarios%3Fstatus%3Daberto&aba=conversa'
    )
    expect(tabHref(`${base}&aba=conversa`, 'detalhes')).toBe(
      '/usuarios/tickets/7?de=%2Fusuarios%3Fstatus%3Daberto'
    )
  })
})

describe('TicketTabs (spec 003 AC-2.1)', () => {
  const render = (initialTab: 'conversa' | 'detalhes') =>
    renderToStaticMarkup(
      <TicketTabs
        conversation={<p>conversa aqui</p>}
        details={<p>detalhes aqui</p>}
        initialTab={initialTab}
        titleId="ticket-title"
      />
    )

  it('has the two tabs, "Detalhes" selected by default, and both panels', () => {
    const html = render('detalhes')
    expect(html).toContain('role="tablist"')
    const tabs = [
      ...html.matchAll(/<button[^>]*role="tab"[^>]*>([^<]*)<\/button>/g)
    ]
    expect(tabs.map(tab => tab[1])).toEqual(['Detalhes', 'Conversa'])
    expect(tabs[0][0]).toContain('aria-selected="true"')
    expect(tabs[1][0]).toContain('aria-selected="false"')
    expect(html.match(/role="tabpanel"/g)).toHaveLength(2)
    expect(html).toContain('detalhes aqui')
    expect(html).toContain('conversa aqui')
  })

  it('opens on "Conversa" when the address asked for it', () => {
    const html = render('conversa')
    const selected = /<button[^>]*aria-selected="true"[^>]*>([^<]*)</.exec(html)
    expect(selected?.[1]).toBe('Conversa')
  })
})
