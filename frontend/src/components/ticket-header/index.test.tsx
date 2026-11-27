import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'

import type { TicketDetail } from '@/domain/tickets/types'

import { TicketHeader } from '.'

const TICKET: TicketDetail = {
  createdAt: '2026-11-20T12:15:00Z',
  data: {
    ambiente: 'prod',
    observacoes: null,
    projeto: 'api',
    repositorio: 'https://github.com/org/api'
  },
  id: 42,
  status: 'em_andamento',
  title: 'Deploy de api em Produção',
  type: 'deploy',
  updatedAt: '2026-11-25T17:04:00Z'
}

const render = (ticket: TicketDetail = TICKET, actions?: string) =>
  renderToStaticMarkup(
    <TicketHeader actions={actions} ticket={ticket} titleId="ticket-title" />
  )

describe('TicketHeader', () => {
  it('shows the number, the title as h1, the badge, the type and the dates', () => {
    const html = render()
    expect(html).toContain('>#42</p>')
    expect(html).toMatch(
      /<h1[^>]*id="ticket-title"[^>]*tabindex="-1"[^>]*>Deploy de api em Produção<\/h1>/
    )
    expect(html).toContain('>Em andamento</span>')
    expect(html).toContain('Deploy de projeto')
    expect(html).toContain('Criado em:')
    expect(html).toContain(
      '<time dateTime="2026-11-25T17:04:00Z" title="25/11/2026 às 14:04">25/11/2026 às 14:04</time>'
    )
    expect(html).not.toContain('Usuário')
  })

  it('shows no focus ring on the title, which only the page focuses', () => {
    const h1 = /<h1[^>]*>/.exec(render())?.[0] ?? ''
    expect(h1).toContain('outline-none')
    expect(h1).not.toMatch(/ring/)
  })

  it('adds "Usuário: <login>" and the actions for the admin', () => {
    const html = render({ ...TICKET, authorLogin: 'maria.souza' }, 'ação')
    expect(html).toMatch(/Usuário:<\/dt><dd[^>]*>maria\.souza<\/dd>/)
    expect(html).toContain('ação')
  })

  it('shows a title with markup as text', () => {
    const html = render({ ...TICKET, title: '<img src=x onerror=alert(1)>' })
    expect(html).not.toContain('<img')
    expect(html).toContain('&lt;img src=x onerror=alert(1)&gt;')
  })
})
