import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'

import { AVATAR_TINT_COUNT, avatarFor } from '@/domain/avatar'
import { contrastRatio, themeToken } from '@/test/contrast'

import { AVATAR_TINT_CLASSES, Avatar } from '.'

const render = (login: string) => renderToStaticMarkup(<Avatar login={login} />)

describe('Avatar', () => {
  it('shows the initials on the tint of the login', () => {
    const html = render('maria.silva')
    const { tint } = avatarFor('maria.silva')
    expect(html).toContain('>MS</span>')
    expect(html).toContain(`data-avatar-tint="${tint}"`)
    expect(html).toContain(AVATAR_TINT_CLASSES[tint])
    expect(render('joao')).toContain('>J</span>')
  })

  it('draws one square per filled cell of the figure', () => {
    const { cells } = avatarFor('maria.silva')
    const squares = render('maria.silva').match(/<rect /g) ?? []
    expect(squares).toHaveLength(cells.filter(Boolean).length)
  })

  it('is decorative and loads nothing: no img, link, src or url()', () => {
    for (const login of ['maria.silva', 'joao', 'x', '<img src=x>']) {
      const html = render(login)
      expect(html).toMatch(/^<span[^>]*aria-hidden="true"/)
      expect(html).not.toMatch(/<img|<image|<a[\s>]|href|src=|url\(|data:/i)
    }
  })

  it('shows a login with markup as text', () => {
    expect(render('<b>.x')).toContain('&lt;')
  })

  it('has one class pair per tint of the domain', () => {
    expect(AVATAR_TINT_CLASSES).toHaveLength(AVATAR_TINT_COUNT)
  })

  it.each(Array.from({ length: AVATAR_TINT_COUNT }, (_, index) => index + 1))(
    'tint %i: the initials pass AA on the tile',
    tint => {
      const ratio = contrastRatio(
        themeToken(`avatar-${tint}`),
        themeToken(`avatar-${tint}-foreground`)
      )
      expect(ratio).toBeGreaterThanOrEqual(4.5)
    }
  )
})

describe('contrastRatio', () => {
  it('agrees with known pairs', () => {
    expect(contrastRatio('oklch(1 0 0)', 'oklch(0 0 0)')).toBeCloseTo(21, 1)
    const brand = contrastRatio(
      themeToken('brand'),
      themeToken('brand-foreground')
    )
    expect(brand).toBeGreaterThan(5.2)
    expect(brand).toBeLessThan(5.3)
  })
})
