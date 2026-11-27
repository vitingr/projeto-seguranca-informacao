import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'

import { TextAreaField, limitText } from '.'

const render = (props: Partial<Parameters<typeof TextAreaField>[0]> = {}) =>
  renderToStaticMarkup(
    <TextAreaField count={0} id="d" max={2000} value="" readOnly {...props} />
  )

describe('TextAreaField', () => {
  it('shows the count against the limit inside the box', () => {
    const html = render({ count: 12 })
    expect(html).toMatch(
      /<span class="text-xs tabular-nums text-muted-foreground" data-testid="d-counter" aria-hidden="true">12 \/ 2000<\/span>/
    )
    expect(html).toContain(
      `<span class="sr-only" id="d-limit">${limitText(2000)}</span>`
    )
  })

  it('marks the counter when the text is over the limit', () => {
    expect(render({ count: 2001 })).toMatch(
      /class="text-xs tabular-nums text-destructive"[^>]*>2001 \/ 2000/
    )
  })

  it('puts the hint in the strip under the box and describes the text with it', () => {
    const html = render({
      'aria-describedby': 'd-error',
      hint: 'Um passo por linha.'
    })
    expect(html).toMatch(
      /<p class="[^"]*bg-muted[^"]*" id="d-strip">.*Um passo por linha\.<\/p>/
    )
    expect(html).toMatch(
      /<textarea[^>]*aria-describedby="d-error d-strip d-limit"/
    )
  })

  it('has no strip without a hint', () => {
    const html = render()
    expect(html).not.toContain('d-strip')
    expect(html).toMatch(/<textarea[^>]*aria-describedby="d-limit"/)
  })

  it('marks the whole box on focus and on error, with no shadow', () => {
    const html = render()
    expect(html).toMatch(/data-slot="text-area-field"/)
    expect(html).toContain('focus-within:ring-3')
    expect(html).toContain('has-aria-invalid:border-destructive')
    expect(html).not.toMatch(/\bshadow/)
  })
})
