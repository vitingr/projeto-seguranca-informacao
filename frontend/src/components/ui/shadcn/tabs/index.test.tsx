import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'

import { Tabs, TabsContent, TabsList, TabsTrigger } from '.'

const markup = () =>
  renderToStaticMarkup(
    <Tabs defaultValue="detalhes">
      <TabsList aria-label="Seções do ticket">
        <TabsTrigger value="detalhes">Detalhes</TabsTrigger>
        <TabsTrigger value="conversa">Conversa</TabsTrigger>
      </TabsList>
      <TabsContent value="detalhes">Campos</TabsContent>
      <TabsContent value="conversa">Mensagens</TabsContent>
    </Tabs>
  )

describe('Tabs (WAI-ARIA tabs, spec 003 AC-2.3)', () => {
  it('renders a tablist, its tabs and the selected panel', () => {
    const html = markup()
    expect(html).toContain('role="tablist"')
    expect(html.match(/role="tab"/g)).toHaveLength(2)
    expect(html).toContain('role="tabpanel"')
    expect(html).toContain('aria-selected="true"')
  })

  // Base UI links the panel to its tab (aria-labelledby) once it hydrates;
  // the static markup only has the ids, so the link is checked in the E2E
  // of the ticket page (T8.1).
  it('gives the tabs and the panel ids, and one tab stop', () => {
    const html = markup()
    expect(html).toMatch(/role="tab"[^>]*id="[^"]+"/)
    expect(html).toMatch(/id="[^"]+" role="tabpanel"/)
    expect(html).toContain('role="tabpanel" tabindex="0"')
  })

  // No shadow class is checked for every source file by design-rules.test.ts.
  it('uses the small radius on the list and the tabs', () => {
    expect(markup().match(/rounded-sm/g)?.length).toBeGreaterThanOrEqual(3)
  })
})
