import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it, vi } from 'vitest'

import { readListQuery } from '@/domain/ticket-query'

import { ALL, TicketListFilters, filterHref } from '.'

vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: () => {} })
}))

const own = readListQuery({ pagina: '3', status: 'aberto' }, 'own')
const queue = readListQuery({ pagina: '2', tipo: 'bug' }, 'queue')
const LOGINS = ['ana.lima', 'maria.souza']

describe('filterHref', () => {
  it('sets the type, keeps the status and goes back to page 1', () => {
    expect(filterHref(own, 'type', 'deploy')).toBe(
      '/usuarios?status=aberto&tipo=deploy'
    )
    expect(filterHref(queue, 'type', ALL)).toBe('/administradores')
  })

  it('sets a listed user in the queue only', () => {
    expect(filterHref(queue, 'user', 'maria.souza', LOGINS)).toBe(
      '/administradores?tipo=bug&usuario=maria.souza'
    )
    expect(filterHref(own, 'user', 'maria.souza', LOGINS)).toBeNull()
  })

  it.each([
    ['type', 'nope'],
    ['type', ''],
    ['user', 'joao.silva'],
    ['user', 'a&b=1']
  ] as const)('never writes an unknown %s value (%s)', (filter, code) => {
    expect(filterHref(queue, filter, code, LOGINS)).toBeNull()
  })
})

describe('TicketListFilters', () => {
  const render = (logins?: string[]) =>
    renderToStaticMarkup(
      <TicketListFilters
        logins={logins}
        query={logins ? queue : own}
        statusFilter={<nav>status</nav>}
      >
        <table />
      </TicketListFilters>
    )

  it('has the type Select with "Todos" by default, and no user Select for comum', () => {
    const html = render()
    expect(html).toContain('>Tipo</span>')
    expect(html.match(/role="combobox"/g)).toHaveLength(1)
    expect(html).toContain('Todos')
    expect(html).not.toContain('>Usuário</span>')
    expect(html).toContain('<nav>status</nav>')
    expect(html).toContain('<table></table>')
  })

  it('adds the user Select in the queue, showing the chosen type', () => {
    const html = render(LOGINS)
    expect(html.match(/role="combobox"/g)).toHaveLength(2)
    expect(html).toContain('>Usuário</span>')
    expect(html).toContain('Suporte em bug')
    expect(html).not.toContain('aria-busy')
  })
})
