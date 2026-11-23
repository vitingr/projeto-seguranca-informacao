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
  title: 'Título',
  updatedAt: '2026-11-22T15:04:00Z',
  ...overrides
})

const render = (p: Post) =>
  renderToStaticMarkup(createElement(PostCard, { post: p }))

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
