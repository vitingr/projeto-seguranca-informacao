import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'

import type { ThreadEntry } from '@/domain/tickets/types'

import { Thread } from '.'

const NOW = '2026-11-26T17:30:00Z'
const OPENED = '2026-11-26T17:02:00Z'

const OWN: ThreadEntry[] = [
  {
    createdAt: '2026-11-26T17:04:00Z',
    id: 1,
    kind: 'message',
    mine: true,
    side: 'author',
    text: 'Primeira linha\nsegunda linha'
  },
  {
    createdAt: '2026-11-26T17:06:00Z',
    from: 'aberto',
    id: 2,
    kind: 'status_change',
    mine: false,
    side: 'support',
    to: 'em_andamento'
  },
  {
    createdAt: '2026-11-26T17:08:00Z',
    id: 3,
    kind: 'message',
    mine: false,
    side: 'support',
    text: 'Estamos olhando.'
  }
]

const ADMIN: ThreadEntry[] = [
  { ...OWN[0], actorLogin: 'maria.souza', mine: false },
  { ...OWN[1], actorLogin: 'ana.lima', mine: true },
  { ...OWN[2], actorLogin: 'joao.admin', mine: false }
] as ThreadEntry[]

const render = (props: Partial<Parameters<typeof Thread>[0]> = {}) =>
  renderToStaticMarkup(
    <Thread entries={OWN} now={NOW} openedAt={OPENED} view="own" {...props} />
  )

const items = (html: string) =>
  [
    ...html.matchAll(
      /<li[^>]*data-(side|event)="([^"]*)"[^>]*>([\s\S]*?)<\/li>/g
    )
  ].map(([, , kind, body]) => [kind, body] as const)

describe('Thread (spec 003 AC-3.1 to AC-3.3)', () => {
  it('is an ol labelled "Conversa do ticket", oldest first, opened by "Você"', () => {
    const html = render()
    expect(html).toMatch(/^<ol aria-label="Conversa do ticket"/)
    const found = items(html)
    expect(found.map(([kind]) => kind)).toEqual([
      'opened',
      'own',
      'status_change',
      'other'
    ])
    expect(found[0][1]).toMatch(/>Você<\/span> <span>abriu o ticket<\/span>/)
    expect(found[0][1]).toContain('14:02')
  })

  it('puts own messages on the right on a gray card, with no mark', () => {
    const own = items(render())[1][1]
    const li = /<li[^>]*data-side="own"[^>]*>/.exec(render())?.[0] ?? ''
    expect(li).toContain('justify-end')
    expect(own).toContain('bg-muted')
    expect(own).toContain('>Você</p>')
    expect(own).not.toContain('data-support-mark')
    expect(own).toContain('whitespace-pre-wrap')
    expect(own).toContain('Primeira linha\nsegunda linha')
  })

  it('shows support messages on the left with the support mark, as "Suporte"', () => {
    const other = items(render())[3][1]
    expect(other).toContain('data-support-mark')
    expect(other).toContain('bg-background')
    expect(other).toContain('>Suporte</p>')
    expect(other).toMatch(/border[^"]*rounded-sm[^"]*|rounded-sm[^"]*border/)
    expect(other).toContain('max-w-4/5')
    expect(other).toContain('font-medium')
    expect(other).toMatch(
      /<time dateTime="2026-11-26T17:08:00Z" title="26\/11\/2026 às 14:08">/
    )
  })

  it('shows status lines with no card: actor, the badge and the time', () => {
    const event = items(render())[2][1]
    expect(event).toMatch(
      /class="text-foreground[^"]*">Suporte<\/span> <span>alterou o status para<\/span> <span[^>]*data-status="em_andamento"[^>]*>Em andamento<\/span>/
    )
    expect(event).toContain('14:06')
    expect(event).not.toContain('<article')
  })

  it('names everyone by login in the admin view, with the author avatar', () => {
    const html = render({
      authorLogin: 'maria.souza',
      entries: ADMIN,
      view: 'admin'
    })
    const found = items(html)
    expect(found[0][1]).toContain('>maria.souza</span> <span>abriu o ticket')
    expect(found[1][0]).toBe('other')
    expect(found[1][1]).toContain('>maria.souza</p>')
    expect(found[1][1]).toContain('data-avatar-tint')
    expect(found[2][1]).toContain('>Você</span> <span>alterou o status para')
    expect(found[3][1]).toContain('>joao.admin</p>')
    expect(found[3][1]).toContain('data-support-mark')
  })

  it('never names an admin to the comum author (S-13)', () => {
    const html = render({
      entries: [{ ...OWN[2], actorLogin: 'joao.admin' } as ThreadEntry]
    })
    expect(html).not.toContain('joao.admin')
  })

  it('shows the invitation under the opening line when there is no entry', () => {
    const html = render({ entries: [] })
    expect(html).toContain(
      'Nenhuma mensagem ainda. Escreva abaixo se quiser acrescentar algo ao ticket.'
    )
  })

  it.each([
    '<script>alert(1)</script>',
    '<img src=x onerror=alert(1)>',
    'javascript:alert(1) https://exemplo.com'
  ])('shows %s as literal text, with no element (S-10)', payload => {
    const html = render({
      entries: [{ ...OWN[2], text: payload } as ThreadEntry]
    })
    expect(html).not.toMatch(/<script|<img|<a /)
    expect(html).toContain(
      payload.replaceAll('<', '&lt;').replaceAll('>', '&gt;')
    )
  })

  it('has none of the "not copied" items: no fade, shadow, live dot or typing card', () => {
    const html = render()
    expect(html).not.toMatch(/opacity-|shadow|gradient|Writing|Live|@/)
    expect(html).toContain('hover:border-foreground/30')
  })
})
