import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'

import { SupportMark } from '.'

describe('SupportMark (spec 003 section 7)', () => {
  it('is a decorative neutral square with a headset icon', () => {
    const html = renderToStaticMarkup(<SupportMark />)
    expect(html).toMatch(/^<span[^>]*aria-hidden="true"/)
    expect(html).toContain('rounded-sm')
    expect(html).toContain('size-6')
    expect(html).toContain('lucide-headset')
  })
})
