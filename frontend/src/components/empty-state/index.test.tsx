import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'

import { EmptyState } from '.'

describe('EmptyState (spec 003 AC-1.8)', () => {
  it('shows its text and one action', () => {
    const html = renderToStaticMarkup(
      <EmptyState
        action={{ href: '/usuarios/tickets/novo', label: 'Novo ticket' }}
        text="Você ainda não criou nenhum ticket."
      />
    )
    expect(html).toContain('>Você ainda não criou nenhum ticket.</p>')
    expect(html.match(/<a /g)).toHaveLength(1)
    expect(html).toMatch(
      /<a[^>]*href="\/usuarios\/tickets\/novo"[^>]*>Novo ticket<\/a>/
    )
    expect(html).toContain('rounded-sm')
  })

  it('shows only the text when there is no action', () => {
    const html = renderToStaticMarkup(
      <EmptyState text="Nenhum ticket na fila." />
    )
    expect(html).toContain('Nenhum ticket na fila.')
    expect(html).not.toContain('<a')
  })
})
