import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'

import { readListQuery } from '@/domain/ticket-query'

import { StatusFilter } from '.'

const links = (html: string) =>
  [...html.matchAll(/<a([^>]*)>([^<]*)<\/a>/g)].map(([, attrs, text]) => ({
    current: attrs.includes('aria-current="page"'),
    href: /href="([^"]*)"/.exec(attrs)?.[1]?.replaceAll('&amp;', '&'),
    muted: attrs.includes('bg-muted text-foreground'),
    text
  }))

describe('StatusFilter', () => {
  it('is a row of links, "Todos" first and active by default', () => {
    const html = renderToStaticMarkup(
      <StatusFilter query={readListQuery({}, 'own')} />
    )
    expect(html).toMatch(/^<nav aria-label="Filtrar por status">/)
    const found = links(html)
    expect(found.map(link => link.text)).toEqual([
      'Todos',
      'Aberto',
      'Em andamento',
      'Resolvido',
      'Fechado'
    ])
    expect(found.filter(link => link.current).map(link => link.text)).toEqual([
      'Todos'
    ])
    expect(found[0]).toMatchObject({ href: '/usuarios', muted: true })
  })

  it('marks the active status and keeps the other filters, back on page 1', () => {
    const query = readListQuery(
      { pagina: '3', status: 'resolvido', tipo: 'bug' },
      'own'
    )
    const found = links(renderToStaticMarkup(<StatusFilter query={query} />))
    const active = found.filter(link => link.current)
    expect(active).toEqual([
      expect.objectContaining({ muted: true, text: 'Resolvido' })
    ])
    expect(found.find(link => link.text === 'Aberto')?.href).toBe(
      '/usuarios?status=aberto&tipo=bug'
    )
    expect(found.find(link => link.text === 'Todos')?.href).toBe(
      '/usuarios?tipo=bug'
    )
  })
})
