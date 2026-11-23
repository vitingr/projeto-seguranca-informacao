import { createElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'

import type { Post } from '@/domain/post/types'

import { PostCard } from '.'

const post = (overrides: Partial<Post> = {}): Post => ({
  author: { login: 'maria', userId: 2 },
  body: 'Corpo',
  createdAt: '2026-11-22T15:04:00Z',
  id: 1,
  mine: false,
  removal: null,
  title: 'Título',
  updatedAt: '2026-11-22T15:04:00Z',
  ...overrides
})

const render = (p: Post, showRemovedBody = false) =>
  renderToStaticMarkup(createElement(PostCard, { post: p, showRemovedBody }))

// 005 S-3: whatever the author types is shown as text. React escapes it, so
// no element, attribute or markdown formatting comes from the content.
describe('PostCard renders user content as plain text', () => {
  const payloads = [
    '<img src=x onerror="alert(1)">',
    '<script>alert(2)</script>',
    '<a href="javascript:alert(3)">link</a>',
    '**negrito** _itálico_ [link](https://example.com)'
  ]

  it.each(payloads)('body %s', payload => {
    const html = render(post({ body: payload }))
    expect(html).not.toMatch(/<(img|script|a|strong|em)\b/)
    expect(html).toContain(
      payload
        .replaceAll('&', '&amp;')
        .replaceAll('<', '&lt;')
        .replaceAll('>', '&gt;')
        .replaceAll('"', '&quot;')
    )
  })

  it.each(payloads)('title %s', payload => {
    const html = render(post({ title: payload }))
    expect(html).not.toMatch(/<(img|script|a|strong|em)\b/)
  })

  it('keeps the line breaks as text, with no <br> added', () => {
    const html = render(post({ body: 'linha 1\nlinha 2' }))
    expect(html).toContain('linha 1\nlinha 2')
    expect(html).not.toContain('<br')
    expect(html).toContain('whitespace-pre-wrap')
  })
})

describe('PostCard', () => {
  it('shows the author and the date', () => {
    const html = render(post())
    expect(html).toContain('maria')
    expect(html).toContain('22/11/2026, 12:04')
    expect(html).not.toContain('editado')
  })

  it('marks an edited post', () => {
    expect(render(post({ updatedAt: '2026-11-22T16:00:00Z' }))).toContain(
      'editado'
    )
  })
})

// 006 D-2: a removed post shows the notice with the reason, never its body.
describe('PostCard of a removed post', () => {
  const removal = { reason: 'Spam.', removedAt: '2026-11-22T16:00:00Z' }

  it('shows the notice and the reason instead of the body', () => {
    const html = render(post({ body: 'Texto removido', removal }))
    expect(html).toContain('Removido pela moderação:')
    expect(html).toContain('Spam.')
    expect(html).not.toContain('Texto removido')
    expect(html).toContain('data-removed="true"')
  })

  it('renders the reason as plain text', () => {
    const reason = '<img src=x onerror="alert(1)"> **x**'
    const html = render(post({ removal: { ...removal, reason } }))
    expect(html).not.toMatch(/<(img|strong)\b/)
    expect(html).toContain(
      '&lt;img src=x onerror=&quot;alert(1)&quot;&gt; **x**'
    )
  })
})

// The admin can open the removed body, collapsed; the author cannot.
describe('PostCard of a removed post for an admin', () => {
  const removal = { reason: 'Spam.', removedAt: '2026-11-22T16:00:00Z' }

  it('puts the body in a closed details block', () => {
    const html = render(post({ body: 'Texto removido', removal }), true)
    expect(html).toContain('Ver conteúdo removido')
    expect(html).toContain('Texto removido')
    expect(html).toMatch(/<details data-testid="post-removed-content">/)
    expect(html).not.toMatch(/<details[^>]* open/)
  })

  it('renders the removed body as plain text', () => {
    const body = '<img src=x onerror="alert(1)"> **x**'
    const html = render(post({ body, removal }), true)
    expect(html).not.toMatch(/<(img|strong)[\s>]/)
  })

  it('shows no body block for a post that is not removed', () => {
    const html = render(post(), true)
    expect(html).not.toContain('Ver conteúdo removido')
    expect(html).toContain('data-testid="post-body"')
  })

  it('shows no body block when the viewer may not see it', () => {
    const html = render(post({ body: 'Texto removido', removal }))
    expect(html).not.toContain('Ver conteúdo removido')
    expect(html).not.toContain('Texto removido')
  })
})
