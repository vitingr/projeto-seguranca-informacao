import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'

import { ThreadEvent } from '.'

const NOW = '2026-11-26T17:30:00Z'

describe('ThreadEvent (spec 003 AC-3.2)', () => {
  it('is the opening line with no status', () => {
    const html = renderToStaticMarkup(
      <ThreadEvent actor="Você" now={NOW} time="2026-11-26T17:02:00Z" />
    )
    expect(html).toContain('data-event="opened"')
    expect(html).toMatch(/>Você<\/span> <span>abriu o ticket<\/span>/)
    expect(html).not.toContain('data-status')
  })

  it('is a status line with the badge of the new status', () => {
    const html = renderToStaticMarkup(
      <ThreadEvent
        actor="Suporte"
        now={NOW}
        status="fechado"
        time="2026-11-26T17:06:00Z"
      />
    )
    expect(html).toContain('alterou o status para')
    expect(html).toContain('data-status="fechado"')
    expect(html).toContain('14:06')
  })
})
