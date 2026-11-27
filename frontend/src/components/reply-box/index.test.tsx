import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'

import { ReplyBox, hasText } from '.'

const render = (props: Partial<Parameters<typeof ReplyBox>[0]> = {}) =>
  renderToStaticMarkup(
    <ReplyBox
      id="reply"
      onChange={() => {}}
      onSend={() => {}}
      placeholder="Escreva uma mensagem…"
      trayText="Visível para a equipe de suporte"
      value=""
      {...props}
    />
  )

const sendButton = (html: string) =>
  /<button[^>]*data-testid="reply-send"[^>]*>/.exec(html)?.[0] ?? ''

describe('ReplyBox', () => {
  it('has the sr-only label "Mensagem", the placeholder and a growing text area', () => {
    const html = render()
    expect(html).toContain(
      '<label class="sr-only" for="reply">Mensagem</label>'
    )
    expect(html).toMatch(
      /<textarea[^>]*class="[^"]*field-sizing-content[^"]*min-h-16[^"]*max-h-48|<textarea[^>]*class="[^"]*max-h-48[^"]*min-h-16/
    )
    expect(html).toContain('placeholder="Escreva uma mensagem…"')
  })

  it('shows the counter, hidden from assistive technology, with the sr-only limit', () => {
    const html = render({ value: 'olá' })
    expect(html).toMatch(
      /data-testid="reply-counter" aria-hidden="true">3 \/ 2000</
    )
    expect(html).toContain(
      '<span class="sr-only" id="reply-limit">Até 2000 caracteres.</span>'
    )
    const over = render({ value: 'a'.repeat(2001) })
    expect(over).toMatch(/text-destructive[^"]*" data-testid="reply-counter"/)
  })

  it('has a muted, inactive send button while there is no text', () => {
    for (const value of ['', '   \n ']) {
      const button = sendButton(render({ value }))
      expect(button).toContain('aria-disabled="true"')
      expect(button).toContain('bg-muted')
      expect(button).toContain('aria-label="Enviar mensagem"')
      expect(button).toContain('size-8')
      expect(button).toContain('rounded-sm')
    }
    expect(hasText(' \n\t')).toBe(false)
  })

  it('has a solid dark send button with text, and a busy one while sending', () => {
    const ready = sendButton(render({ value: 'Oi' }))
    expect(ready).not.toContain('aria-disabled')
    expect(ready).toContain('bg-primary')
    const html = render({ sending: true, value: 'Oi' })
    expect(sendButton(html)).toContain('aria-busy="true"')
    expect(html).toContain('motion-safe:animate-spin')
    expect(html).toMatch(/<textarea[^>]*readOnly=""/)
  })

  it('has the tray: who reads the reply and the shortcut', () => {
    expect(render()).toContain('Visível para a equipe de suporte')
    expect(render()).toContain('Ctrl+Enter para enviar')
    const admin = render({ mac: true, trayText: 'Visível para maria.souza' })
    expect(admin).toContain('Visível para maria.souza')
    expect(admin).toContain('⌘+Enter para enviar')
  })

  it('shows the field message and marks the field invalid', () => {
    const html = render({ error: 'Use no máximo 2000 caracteres.' })
    expect(html).toContain('aria-invalid="true"')
    expect(html).toContain('aria-describedby="reply-error reply-limit"')
    expect(html).toContain(
      'id="reply-error">Use no máximo 2000 caracteres.</p>'
    )
  })

  it('becomes the closed line with "Novo ticket" and no box', () => {
    const html = render({
      closed: true,
      closedAction: { href: '/usuarios/tickets/novo', label: 'Novo ticket' }
    })
    expect(html).toContain(
      'Este ticket foi fechado. Abra um novo ticket se precisar de mais ajuda.'
    )
    expect(html).toMatch(
      /<a[^>]*href="\/usuarios\/tickets\/novo"[^>]*>Novo ticket<\/a>/
    )
    expect(html).not.toContain('<textarea')
  })

  it('has none of the "not taken" items of the video', () => {
    const html = render({ value: 'Oi' })
    expect(html).not.toMatch(
      /lucide-(plus|mic|at-sign|paperclip)|Auto|to everyone|@/
    )
    expect(html).toContain('rounded-sm')
  })
})
