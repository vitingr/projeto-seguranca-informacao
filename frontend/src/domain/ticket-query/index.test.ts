import { describe, expect, it } from 'vitest'

import { backToListHref, listQueryHref, readListQuery } from '.'

describe('readListQuery', () => {
  it('reads the known codes, the login and the page', () => {
    expect(
      readListQuery(
        { pagina: '2', status: 'aberto', tipo: 'bug', usuario: 'maria.souza' },
        'queue'
      )
    ).toEqual({
      page: 2,
      scope: 'queue',
      status: 'aberto',
      type: 'bug',
      user: 'maria.souza'
    })
  })

  it('turns unknown, empty and repeated values into "all" and page 1', () => {
    expect(
      readListQuery(
        { pagina: '0', status: 'pendente', tipo: ['bug', 'deploy'] },
        'own'
      )
    ).toEqual({ page: 1, scope: 'own', status: null, type: null, user: null })
    expect(readListQuery({ pagina: '10001', status: '' }, 'own').page).toBe(1)
  })

  it('reads the user only in the queue, and only a valid login', () => {
    expect(readListQuery({ usuario: 'maria' }, 'own').user).toBeNull()
    expect(readListQuery({ usuario: 'a&b=c' }, 'queue').user).toBeNull()
  })
})

describe('listQueryHref', () => {
  const own = readListQuery({}, 'own')

  it('leaves out "all" and page 1', () => {
    expect(listQueryHref(own)).toBe('/usuarios')
    expect(listQueryHref(own, { page: 2 })).toBe('/usuarios?pagina=2')
  })

  it('writes the filters and goes back to page 1 when one changes', () => {
    const page3 = readListQuery({ pagina: '3', status: 'aberto' }, 'queue')
    expect(listQueryHref(page3, { type: 'bug' })).toBe(
      '/administradores?status=aberto&tipo=bug'
    )
    expect(listQueryHref(page3, { user: 'ana.lima' })).toBe(
      '/administradores?status=aberto&usuario=ana.lima'
    )
  })

  it('never writes a value that breaks its rule', () => {
    expect(listQueryHref(own, { user: 'ana' })).toBe('/usuarios')
    const queue = readListQuery({}, 'queue')
    expect(listQueryHref(queue, { user: 'a#b c&d' })).toBe('/administradores')
  })
})

describe('backToListHref', () => {
  it('keeps the filters of a list address of the scope', () => {
    expect(backToListHref('/usuarios?status=aberto&pagina=2', 'own')).toBe(
      '/usuarios?status=aberto&pagina=2'
    )
    expect(backToListHref('/usuarios?status=x&evil=1', 'own')).toBe('/usuarios')
  })

  it('refuses any other address', () => {
    expect(backToListHref('https://evil.example/usuarios', 'own')).toBe(
      '/usuarios'
    )
    expect(backToListHref('//evil.example', 'own')).toBe('/usuarios')
    expect(backToListHref('/administradores?status=aberto', 'own')).toBe(
      '/usuarios'
    )
    expect(backToListHref(undefined, 'queue')).toBe('/administradores')
  })
})
