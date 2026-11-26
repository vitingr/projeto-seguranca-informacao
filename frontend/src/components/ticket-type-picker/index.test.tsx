import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'

import { TICKET_TYPES } from '@/domain/tickets'

import { TicketTypePicker } from '.'

// Static markup only (Vitest runs in Node, with no DOM). Keyboard use is
// covered by e2e/tickets.spec.ts.

const render = (props: Partial<Parameters<typeof TicketTypePicker>[0]> = {}) =>
  renderToStaticMarkup(
    <TicketTypePicker
      labelledBy="heading"
      onValueChange={() => {}}
      options={TICKET_TYPES}
      value={null}
      {...props}
    />
  )

const count = (html: string, pattern: RegExp) =>
  [...html.matchAll(pattern)].length

describe('TicketTypePicker', () => {
  it('is one single-choice group with the four types (AC-2.1, AC-2.5)', () => {
    const html = render()
    expect(count(html, /role="radiogroup"/g)).toBe(1)
    expect(html).toContain('aria-labelledby="heading"')
    expect(count(html, /role="radio"/g)).toBe(4)
    for (const option of TICKET_TYPES) {
      expect(html).toContain(option.label)
      expect(html).toContain(option.description)
    }
  })

  it('has no type selected at first', () => {
    const html = render()
    expect(html).not.toContain('aria-checked="true"')
    expect(count(html, /aria-checked="false"/g)).toBe(4)
  })

  it('marks the selected type', () => {
    const html = render({ value: 'bug' })
    expect(count(html, /aria-checked="true"/g)).toBe(1)
  })

  it('shows the error under the group and links it', () => {
    const html = render({ error: 'Escolha um tipo de ticket.' })
    expect(html).toContain('Escolha um tipo de ticket.')
    expect(html).toMatch(/aria-describedby="[^"]+-error"/)
    expect(html).toContain('aria-invalid="true"')
  })
})
