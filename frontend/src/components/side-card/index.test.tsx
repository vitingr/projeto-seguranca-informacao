import { LightbulbIcon } from 'lucide-react'
import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'

import { SideCard } from '.'

const render = (defaultOpen?: boolean) =>
  renderToStaticMarkup(
    <SideCard defaultOpen={defaultOpen} icon={LightbulbIcon} title="Dicas">
      <p>corpo</p>
    </SideCard>
  )

describe('SideCard', () => {
  it('is a titled section whose button controls the body', () => {
    const html = render()
    const titleId = /<h2[^>]*id="([^"]+)"/.exec(html)?.[1]
    expect(html).toContain(`aria-labelledby="${titleId}"`)
    const bodyId = /<div id="([^"]+)">/.exec(html)?.[1]
    expect(bodyId).toBeTruthy()
    expect(html).toContain(`aria-controls="${bodyId}"`)
  })

  it('starts open by default', () => {
    const html = render()
    expect(html).toContain('aria-expanded="true"')
    expect(html).toContain('aria-label="Recolher Dicas"')
    expect(html).toMatch(/<div id="[^"]+"><p>corpo<\/p>/)
  })

  it('can start collapsed', () => {
    const html = render(false)
    expect(html).toContain('aria-expanded="false"')
    expect(html).toContain('aria-label="Mostrar Dicas"')
    expect(html).toMatch(/<div hidden="" id="[^"]+"><p>corpo<\/p>/)
  })
})
