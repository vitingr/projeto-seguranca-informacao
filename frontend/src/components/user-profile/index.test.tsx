import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'

import { PROFILE_TEXTS, UserProfile } from '.'

const render = (props: Partial<Parameters<typeof UserProfile>[0]> = {}) =>
  renderToStaticMarkup(
    <UserProfile
      area="usuarios"
      createdAt="2026-11-20T12:15:00Z"
      login="maria.silva"
      {...props}
    />
  )

const pair = (label: string, value: string) =>
  new RegExp(`<dt[^>]*>${label}</dt><dd[^>]*>${value}</dd>`)

describe('UserProfile', () => {
  it('keeps the exact text of the assignment and the area line', () => {
    const html = render()
    expect(html).toContain(`<p>${PROFILE_TEXTS.description}</p>`)
    expect(PROFILE_TEXTS.description).toBe(
      'Página de acesso restrito aos usuários comuns'
    )
    expect(html).toMatch(
      /<p[^>]*data-testid="area"[^>]*>Área liberada pelo servidor: usuarios<\/p>/
    )
    expect(html).toMatch(/<h1[^>]*>Minha conta<\/h1>/)
  })

  it('shows the avatar next to the login, which is the text read', () => {
    const html = render()
    expect(html).toMatch(
      /data-testid="profile-identity"><span[^>]*aria-hidden="true"[^>]*>.*>MS<\/span><\/span><div[^>]*><p[^>]*>maria\.silva<\/p>/
    )
  })

  it('shows the profile rows as text, with no user number', () => {
    const html = render()
    expect(html).toMatch(pair('Login', 'maria\\.silva'))
    expect(html).toMatch(pair('Papel', 'Usuário comum'))
    expect(html).toMatch(pair('Membro desde', '20/11/2026'))
    expect(html).toContain('>Usuário comum</span>')
    expect(html).not.toMatch(/user_id|userId|ID do usuário|Número/)
  })

  it('states the session and rate facts', () => {
    const html = render()
    expect(html).toContain('Sessão e limites')
    expect(html).toContain('período sem atividade')
    expect(html).toContain('até 10 tickets por minuto')
  })

  it('links to the builder with "Novo ticket"', () => {
    expect(render()).toMatch(
      /<a[^>]*href="\/usuarios\/tickets\/novo"[^>]*>Novo ticket<\/a>/
    )
  })

  it('shows a login with markup as text', () => {
    const html = render({ login: '<img src=x onerror=alert(1)>' })
    expect(html).not.toContain('<img')
    expect(html).toContain('&lt;img src=x onerror=alert(1)&gt;')
  })

  it('shows a dash for a missing login or date, never a broken value', () => {
    const html = render({ createdAt: undefined, login: undefined })
    expect(html).toMatch(pair('Login', '—'))
    expect(html).toMatch(pair('Membro desde', '—'))
    expect(render({ createdAt: 'ontem' })).toMatch(pair('Membro desde', '—'))
  })
})
