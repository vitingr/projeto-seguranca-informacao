import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'

import { FIELD_MARKERS, FormField, FormInput } from '.'

const render = (props: Partial<Parameters<typeof FormField>[0]> = {}) =>
  renderToStaticMarkup(
    <FormField id="f" label="Sistema afetado" required {...props}>
      {control => <FormInput {...control} />}
    </FormField>
  )

describe('FormField', () => {
  it('marks a required field with a red asterisk read as "obrigatório"', () => {
    const html = render()
    expect(html).toMatch(
      /<label[^>]*for="f"[^>]*>Sistema afetado<span class="text-destructive" aria-hidden="true">\*<\/span><span class="sr-only">\(obrigatório\)<\/span><\/label>/
    )
    expect(html).toContain('aria-required="true"')
    expect(html).not.toContain(FIELD_MARKERS.optional)
  })

  it('says "(opcional)" on an optional field', () => {
    const html = render({ required: false })
    expect(html).toContain(`>${FIELD_MARKERS.optional}<`)
    expect(html).not.toContain('aria-required')
    expect(html).not.toContain('*<')
  })

  it('ties the hint and the error to the control', () => {
    const html = render({ error: 'Preencha este campo.', hint: 'Use https://' })
    expect(html).toContain(
      '<p class="text-muted-foreground text-xs" id="f-hint">Use https://</p>'
    )
    expect(html).toContain('id="f-error">Preencha este campo.</p>')
    expect(html).toMatch(/<input[^>]*aria-describedby="f-hint f-error"/)
    expect(html).toMatch(/<input[^>]*aria-invalid="true"/)
  })

  it('leaves the control plain without hint or error', () => {
    const html = render()
    expect(html).not.toContain('aria-describedby=')
    expect(html).not.toContain('aria-invalid=')
  })
})
