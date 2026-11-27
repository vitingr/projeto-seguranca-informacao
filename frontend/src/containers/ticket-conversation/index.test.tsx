import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it, vi } from 'vitest'

import { TicketConversation, messageCountText, refreshedAtText } from '.'

vi.mock('next/navigation', () => ({
  useRouter: () => ({ refresh: () => {} })
}))

describe('TicketConversation (spec 003 AC-3.10, D-4)', () => {
  const html = renderToStaticMarkup(
    <TicketConversation messageCount={5} renderedAt="2026-11-26T17:12:00Z">
      <ol />
    </TicketConversation>
  )

  it('has the header row: "N mensagens" and "Atualizado às HH:MM · Atualizar"', () => {
    expect(html).toContain('>5 mensagens</p>')
    expect(html).toContain('>Atualizado às 14:12</span>')
    expect(html).toMatch(/<button[^>]*type="button"[^>]*>.*Atualizar<\/button>/)
    expect(messageCountText(1)).toBe('1 mensagem')
    expect(messageCountText(0)).toBe('0 mensagens')
    expect(refreshedAtText('x')).toBe('Atualizado agora')
  })

  it('puts a hairline under the header row and the thread in a bounded scroll region', () => {
    expect(html).toContain('data-slot="separator"')
    expect(html).toMatch(
      /<div[^>]*aria-label="Mensagens do ticket"[^>]*class="thread-scroll overflow-y-auto[^"]*"[^>]*role="region"[^>]*tabindex="0"[^>]*><ol><\/ol><\/div>/
    )
    expect(html).toContain('rounded-sm')
  })
})
