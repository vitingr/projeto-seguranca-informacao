import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'

import { readListQuery } from '@/domain/ticket-query'

import { Pager } from '.'

const render = (page: number, pages: number) =>
  renderToStaticMarkup(
    <Pager
      pages={pages}
      query={readListQuery({ pagina: String(page), tipo: 'bug' }, 'own')}
    />
  ).replaceAll('&amp;', '&')

const hrefOf = (html: string, text: string) =>
  new RegExp(`<a[^>]*href="([^"]*)"[^>]*>(?:<svg.*?</svg>)?${text}`).exec(
    html
  )?.[1]

describe('Pager', () => {
  it('shows "Anterior", "Página N de M" and "Próxima" as links in the middle', () => {
    const html = render(2, 3)
    expect(html).toContain('Página 2 de 3')
    expect(hrefOf(html, 'Anterior')).toBe('/usuarios?tipo=bug')
    expect(hrefOf(html, 'Próxima')).toBe('/usuarios?tipo=bug&pagina=3')
  })

  it('shows the ends as text, not links', () => {
    const first = render(1, 2)
    expect(hrefOf(first, 'Anterior')).toBeUndefined()
    expect(first).toMatch(/<span[^>]*>.*Anterior<\/span>/)
    expect(hrefOf(first, 'Próxima')).toBe('/usuarios?tipo=bug&pagina=2')
    const last = render(2, 2)
    expect(hrefOf(last, 'Próxima')).toBeUndefined()
    expect(render(1, 1)).not.toContain('<a')
  })

  it('past the end, says the page does not exist and offers the first page', () => {
    const html = render(5, 2)
    expect(html).toContain('Esta página não existe.')
    expect(hrefOf(html, 'Ir para a primeira página')).toBe('/usuarios?tipo=bug')
    expect(html).not.toContain('Página 5 de 2')
  })
})
