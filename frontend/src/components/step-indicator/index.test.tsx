import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'

import { StepIndicator } from '.'

const STEPS = [
  { id: 'tipo', label: 'Tipo' },
  { id: 'detalhes', label: 'Detalhes' },
  { id: 'revisao', label: 'Revisão' }
]

const render = (current: number) =>
  renderToStaticMarkup(
    <StepIndicator current={current} onSelect={() => {}} steps={STEPS} />
  )

describe('StepIndicator', () => {
  it('is its own navigation element with a numbered list', () => {
    const html = render(0)
    expect(html).toMatch(/^<nav aria-label="Etapas do ticket"><ol/)
    expect(html).toMatch(/01<\/span><span>Tipo/)
    expect(html).toMatch(/02<\/span><span>Detalhes/)
    expect(html).toMatch(/03<\/span><span>Revisão/)
  })

  it('marks only the active step', () => {
    const html = render(1)
    expect([...html.matchAll(/aria-current="step"/g)]).toHaveLength(1)
    expect(html).toMatch(/aria-current="step"[^>]*>.*?Detalhes/)
  })

  it('makes only the steps already done revisitable', () => {
    const html = render(2)
    const buttons = [...html.matchAll(/<button[^>]*>(.*?)<\/button>/g)].map(
      match => match[1]
    )
    expect(buttons).toHaveLength(2)
    expect(buttons[0]).toContain('Tipo')
    expect(buttons[1]).toContain('Detalhes')
    expect(buttons.some(button => button.includes('Revisão'))).toBe(false)
  })

  it('has no step to go back to on the first step', () => {
    expect(render(0)).not.toContain('<button')
  })
})
