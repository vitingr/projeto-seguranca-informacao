import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'

import { readListQuery } from '@/domain/ticket-query'

import { StatusSummary } from '.'

const COUNTS = { aberto: 4, em_andamento: 2, fechado: 1, resolvido: 9 }

const links = (html: string) =>
  [...html.matchAll(/<a([^>]*)>(.*?)<\/a>/g)].map(([, attrs, body]) => ({
    current: attrs.includes('aria-current="page"'),
    href: /href="([^"]*)"/.exec(attrs)?.[1]?.replaceAll('&amp;', '&'),
    text: body.replace(/<[^>]+>/g, '')
  }))

describe('StatusSummary', () => {
  it('shows the four counts in order, as links that set the status filter', () => {
    const query = readListQuery(
      { pagina: '2', tipo: 'bug', usuario: 'maria.souza' },
      'queue'
    )
    const found = links(
      renderToStaticMarkup(<StatusSummary counts={COUNTS} query={query} />)
    )
    expect(found.map(link => link.text)).toEqual([
      'Aberto 4',
      'Em andamento 2',
      'Resolvido 9',
      'Fechado 1'
    ])
    expect(found[0].href).toBe(
      '/administradores?status=aberto&tipo=bug&usuario=maria.souza'
    )
    expect(found.some(link => link.current)).toBe(false)
  })

  it('marks the status the queue is filtered by', () => {
    const query = readListQuery({ status: 'resolvido' }, 'queue')
    const found = links(
      renderToStaticMarkup(<StatusSummary counts={COUNTS} query={query} />)
    )
    expect(found.filter(link => link.current).map(link => link.text)).toEqual([
      'Resolvido 9'
    ])
  })
})
