import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'

import { REVIEW_TEXTS, TicketReview } from '.'

const render = (
  rows: Parameters<typeof TicketReview>[0]['rows'],
  title = 'Bug em Portal'
) =>
  renderToStaticMarkup(
    <TicketReview
      rows={rows}
      title={title}
      type="bug"
      typeLabel="Suporte em bug"
    />
  )

const pair = (label: string, value: string) =>
  new RegExp(
    `<dt[^>]*>${label}</dt><dd[^>]*>(<span[^>]*>)?${value}(</span>)?</dd>`
  )

describe('TicketReview', () => {
  it('shows the type, the title and every label with its value (AC-4.1)', () => {
    const html = render([
      { key: 'sistema', label: 'Sistema afetado', value: 'Portal' },
      { key: 'severidade', label: 'Severidade', value: 'Alta' }
    ])
    expect(html).toMatch(/<dt[^>]*>Tipo<\/dt>/)
    expect(html).toContain('Suporte em bug')
    expect(html).toMatch(pair('Título', 'Bug em Portal'))
    expect(html).toMatch(pair('Sistema afetado', 'Portal'))
    expect(html).toMatch(pair('Severidade', 'Alta'))
  })

  it('keeps line breaks as typed', () => {
    const html = render([
      { key: 'passos', label: 'Passos', value: '1. Abra\n2. Salve' }
    ])
    expect(html).toContain('>1. Abra\n2. Salve<')
    expect(html).toMatch(/<dd class="[^"]*whitespace-pre-wrap/)
  })

  it('says "Não informado" for an empty optional field', () => {
    const html = render([
      { key: 'observacoes', label: 'Observações', value: null }
    ])
    expect(html).toMatch(pair('Observações', REVIEW_TEXTS.notInformed))
    expect(html).toContain(
      `<span class="text-muted-foreground">${REVIEW_TEXTS.notInformed}</span>`
    )
  })

  it('puts a risky choice in discreet red', () => {
    const html = render([
      { key: 'ambiente', label: 'Ambiente', risky: true, value: 'Produção' }
    ])
    expect(html).toContain('<span class="text-destructive">Produção</span>')
  })

  it.each([
    '<script>alert(1)</script>',
    '<img src=x onerror=alert(1)>',
    '**negrito** e [link](javascript:alert(1))'
  ])('shows %s as literal text, creating no element (AC-4.3, S-8)', payload => {
    const html = render(
      [{ key: 'descricao', label: 'Descrição', value: payload }],
      payload
    )
    expect(html).not.toMatch(/<(script|img|strong|a)[\s>]/)
    const escaped = payload
      .replaceAll('&', '&amp;')
      .replaceAll('<', '&lt;')
      .replaceAll('>', '&gt;')
    expect(html.split(escaped)).toHaveLength(3)
  })
})
