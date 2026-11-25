import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'

import { PREVIEW_TEXTS, TicketPreview } from '.'

describe('TicketPreview', () => {
  it('says what is missing before a type is chosen', () => {
    const html = renderToStaticMarkup(
      <TicketPreview title={null} type={null} typeLabel={null} />
    )
    expect(html).toContain(PREVIEW_TEXTS.noTitle)
    expect(html).toContain(PREVIEW_TEXTS.noType)
    expect(html).toContain('lucide-file')
    expect(html).not.toContain('<ul')
  })

  it('shows the type, the title and the options as badges', () => {
    const html = renderToStaticMarkup(
      <TicketPreview
        badges={['Alta']}
        title="Bug em GitLab"
        type="bug"
        typeLabel="Suporte em bug"
      />
    )
    expect(html).toContain('>Bug em GitLab<')
    expect(html).toContain('>Suporte em bug<')
    expect(html).toContain('lucide-bug')
    expect(html).toMatch(
      /<ul aria-label="Opções escolhidas"[^>]*><li><span[^>]*>Alta<\/span><\/li><\/ul>/
    )
  })

  it('renders the title as text, never as markup (S-8)', () => {
    const html = renderToStaticMarkup(
      <TicketPreview
        title={'<img src=x onerror=alert(1)>'}
        type="outro"
        typeLabel="Outro"
      />
    )
    expect(html).toContain('&lt;img src=x onerror=alert(1)&gt;')
    expect(html).not.toContain('<img')
  })
})
