import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'

import { CONFIRMATION_TEXTS, TicketConfirmation } from '.'

const render = (title = 'Liberar porta no firewall') =>
  renderToStaticMarkup(
    <TicketConfirmation
      ticket={{
        createdAt: '2026-11-25T14:30:00.123456Z',
        id: 42,
        status: 'aberto',
        title,
        type: 'outro'
      }}
      headingId="h"
    />
  )

const pair = (label: string, value: string) =>
  new RegExp(`<dt[^>]*>${label}</dt><dd[^>]*>${value}</dd>`)

describe('TicketConfirmation', () => {
  it('says the ticket was created, with a focusable heading (AC-5.2)', () => {
    expect(render()).toMatch(
      new RegExp(
        `<h2[^>]*id="h"[^>]*tabindex="-1"[^>]*>${CONFIRMATION_TEXTS.heading}</h2>`,
        'i'
      )
    )
  })

  it('shows the number, type, title, status badge and date (AC-5.2)', () => {
    const html = render()
    expect(html).toMatch(pair('Número', '#42'))
    expect(html).toMatch(
      /<dt[^>]*>Tipo<\/dt><dd[^>]*><span[^>]*>.*Outro<\/span><\/dd>/
    )
    expect(html).toMatch(pair('Título', 'Liberar porta no firewall'))
    expect(html).toMatch(
      /<dt[^>]*>Status<\/dt><dd[^>]*><span[^>]*bg-secondary[^>]*>Aberto<\/span><\/dd>/
    )
    expect(html).toMatch(pair('Criado em', '25/11/2026 às 11:30'))
  })

  it('shows an HTML title as text (S-8)', () => {
    const html = render('<img src=x onerror=alert(1)>')
    expect(html).toContain('&lt;img src=x onerror=alert(1)&gt;')
    expect(html).not.toContain('<img')
  })
})
