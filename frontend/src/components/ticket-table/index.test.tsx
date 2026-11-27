import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'

import type { TicketSummary } from '@/domain/tickets/types'

import { TicketTable, ticketHref } from '.'

const TICKET: TicketSummary = {
  createdAt: '2026-11-20T12:15:00Z',
  id: 42,
  status: 'em_andamento',
  title: 'Bug em Portal do aluno',
  type: 'bug',
  updatedAt: '2026-11-25T17:04:00Z'
}

const render = (props: Partial<Parameters<typeof TicketTable>[0]> = {}) =>
  renderToStaticMarkup(
    <TicketTable
      caption="Meus tickets"
      items={[TICKET]}
      listHref="/usuarios?status=em_andamento"
      scope="own"
      {...props}
    />
  )

const headers = (html: string) =>
  [...html.matchAll(/<th[^>]*scope="col"[^>]*>([^<]*)<\/th>/g)].map(
    match => match[1]
  )

describe('TicketTable (spec 003 AC-1.2, AC-4.2)', () => {
  it('has the comum columns, a hidden caption and the number of each row', () => {
    const html = render()
    expect(headers(html)).toEqual([
      'Título',
      'Tipo',
      'Status',
      'Criado em',
      'Atualizado em'
    ])
    expect(html).toContain('<caption class="sr-only">Meus tickets</caption>')
    expect(html).toContain('>#42</span>')
    expect(html).toContain('Suporte em bug')
    expect(html).toContain('>Em andamento</span>')
    expect(html).not.toContain('Usuário')
  })

  it('adds "#" and "Usuário" for the queue', () => {
    const html = render({
      items: [{ ...TICKET, authorLogin: 'maria.souza' }],
      listHref: '/administradores',
      scope: 'queue'
    })
    expect(headers(html)).toEqual([
      '#',
      'Título',
      'Tipo',
      'Status',
      'Usuário',
      'Criado em',
      'Atualizado em'
    ])
    expect(html).toContain('maria.souza')
    expect(html).toContain('href="/administradores/tickets/42"')
  })

  it('links the title to the ticket page with the list address in ?de=', () => {
    expect(render()).toContain(
      'href="/usuarios/tickets/42?de=%2Fusuarios%3Fstatus%3Dem_andamento"'
    )
    expect(ticketHref('own', 7, '/usuarios')).toBe('/usuarios/tickets/7')
  })

  it('puts the dates in <time dateTime> with the pt-BR value', () => {
    const html = render()
    expect(html).toContain(
      '<time dateTime="2026-11-25T17:04:00Z" title="25/11/2026 às 14:04">25/11/2026 às 14:04</time>'
    )
    expect(html).toContain('dateTime="2026-11-20T12:15:00Z"')
  })

  it('keeps the same information in the small-screen list', () => {
    const html = render()
    const list = /<ul[^>]*md:hidden[^>]*>.*<\/ul>/.exec(html)?.[0] ?? ''
    expect(list).toContain('Bug em Portal do aluno')
    expect(list).toContain('>Em andamento</span>')
    expect(list).toContain('Suporte em bug')
    expect(list).toContain('href="/usuarios/tickets/42?de=')
  })

  it.each([
    '<script>alert(1)</script>',
    '<img src=x onerror=alert(1)>',
    'javascript:alert(1)'
  ])('shows %s in a title or login as literal text (S-10)', payload => {
    const html = render({
      items: [{ ...TICKET, authorLogin: payload, title: payload }],
      scope: 'queue'
    })
    expect(html).not.toMatch(/<script|<img/)
    // Two title links (table and list) and nothing else.
    expect(html.match(/<a /g)).toHaveLength(2)
    expect(html).not.toContain('href="javascript:')
  })

  it('uses thin row dividers and no zebra stripes', () => {
    const html = render()
    expect(html).toContain('divide-y')
    expect(html).not.toMatch(/even:|odd:/)
  })
})
