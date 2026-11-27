import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'

import { ThreadMessage, ThreadTime } from '.'

const NOW = '2026-11-26T17:30:00Z'

describe('ThreadTime', () => {
  it.each([
    ['2026-11-26T17:04:00Z', '14:04'],
    ['2026-11-25T17:04:00Z', '25/11 14:04'],
    ['2025-11-25T17:04:00Z', '25/11/2025 14:04']
  ])(
    'shows %s as %s, with the full date for assistive technology',
    (iso, short) => {
      const html = renderToStaticMarkup(<ThreadTime iso={iso} now={NOW} />)
      expect(html).toContain(`<span aria-hidden="true">${short}</span>`)
      expect(html).toMatch(
        /<span class="sr-only">\d{2}\/\d{2}\/\d{4} às \d{2}:\d{2}<\/span>/
      )
      expect(html).toContain(`dateTime="${iso}"`)
    }
  )
})

describe('ThreadMessage', () => {
  it('shows a mark only for the other side', () => {
    const props = { name: 'Suporte', now: NOW, text: 'Oi', time: NOW }
    const other = renderToStaticMarkup(
      <ThreadMessage {...props} mark={<i data-mark />} own={false} />
    )
    const own = renderToStaticMarkup(
      <ThreadMessage {...props} mark={<i data-mark />} own />
    )
    expect(other).toContain('data-mark')
    expect(own).not.toContain('data-mark')
  })
})
