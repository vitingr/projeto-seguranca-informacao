import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'

import { InlineAlert } from '.'

const MESSAGE = 'Dados inválidos. Revise os campos e tente novamente.'

describe('InlineAlert', () => {
  it('is a polite status with the message (spec section 7 "Error")', () => {
    const html = renderToStaticMarkup(<InlineAlert>{MESSAGE}</InlineAlert>)
    expect(html).toMatch(/^<div[^>]*role="status"/)
    expect(html).not.toContain('role="alert"')
    expect(html).toContain(MESSAGE)
    expect(html).toContain('rounded-sm')
  })
})
