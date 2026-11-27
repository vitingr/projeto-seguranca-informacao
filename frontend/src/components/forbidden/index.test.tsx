import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'

import { Forbidden } from '.'

const linkClasses = (html: string) =>
  (/<a[^>]*class="([^"]+)"/.exec(html)?.[1] ?? '').split(' ')

describe('Forbidden', () => {
  it.each(['admin', 'comum'] as const)(
    'links the %s user to their landing page with a visible outline border',
    role => {
      const html = renderToStaticMarkup(<Forbidden role={role} />)
      expect(html).toContain(
        role === 'admin' ? 'href="/administradores"' : 'href="/usuarios"'
      )
      const classes = linkClasses(html)
      expect(classes).toContain('border-border')
      expect(classes).not.toContain('border-transparent')
    }
  )
})
