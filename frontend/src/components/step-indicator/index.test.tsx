import { ClipboardCheckIcon, FileTextIcon, ListChecksIcon } from 'lucide-react'
import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'

import { StepIndicator } from '.'

const STEPS = [
  {
    description: 'Escolha do pedido',
    icon: ListChecksIcon,
    id: 'tipo',
    label: 'Tipo'
  },
  {
    description: 'Campos do tipo',
    icon: FileTextIcon,
    id: 'detalhes',
    label: 'Detalhes'
  },
  {
    description: 'Conferir e enviar',
    icon: ClipboardCheckIcon,
    id: 'revisao',
    label: 'Revisão'
  }
]

const render = (current: number) =>
  renderToStaticMarkup(
    <StepIndicator current={current} onSelect={() => {}} steps={STEPS} />
  )

const items = (html: string) =>
  [...html.matchAll(/<li[^>]*>(.*?)<\/li>/g)].map(match => match[0])

describe('StepIndicator', () => {
  it('is its own navigation element with every label and description', () => {
    const html = render(0)
    expect(html).toMatch(/^<nav aria-label="Etapas do ticket"[^>]*><ol/)
    for (const step of STEPS) {
      expect(html).toContain(`>${step.label}<`)
      expect(html).toContain(`>${step.description}<`)
    }
  })

  it('marks only the active step, with the brand accent', () => {
    const list = items(render(1))
    expect(list).toHaveLength(3)
    expect(list.filter(item => item.includes('aria-current="step"'))).toEqual([
      list[1]
    ])
    expect(list[1]).toContain('bg-brand')
    expect(list[0]).not.toContain('bg-brand')
    expect(list[2]).not.toContain('bg-brand')
  })

  it('puts a chevron between steps only', () => {
    const list = items(render(0))
    expect(list[0]).toContain('lucide-chevron-right')
    expect(list[1]).toContain('lucide-chevron-right')
    expect(list[2]).not.toContain('lucide-chevron-right')
  })

  it('makes only the steps already done revisitable, with a check', () => {
    const html = render(2)
    const buttons = [...html.matchAll(/<button[^>]*>(.*?)<\/button>/g)].map(
      match => match[1]
    )
    expect(buttons).toHaveLength(2)
    expect(buttons[0]).toContain('Tipo')
    expect(buttons[0]).toContain('(concluída)')
    expect(buttons[0]).toContain('lucide-check')
    expect(buttons[1]).toContain('Detalhes')
    expect(buttons.some(button => button.includes('Revisão'))).toBe(false)
  })

  it('has no step to go back to on the first step', () => {
    expect(render(0)).not.toContain('<button')
  })

  it('repeats the active step under the row on small screens only', () => {
    expect(render(1)).toMatch(
      /<p class="[^"]*md:hidden"[^>]*aria-hidden="true"><span[^>]*>Detalhes<\/span><span[^>]*>Campos do tipo<\/span><\/p>/
    )
  })
})
