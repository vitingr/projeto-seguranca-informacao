import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'

import { ENVIRONMENTS, SEVERITIES } from '@/domain/tickets'
import { Select } from '@base-ui/react/select'

import { ChoiceOptionRow, ChoiceSelect } from '.'

// Static markup only (Node, no DOM): the open menu is checked in
// e2e/tickets.spec.ts. A row needs the Select and Item context to render.
const row = (
  option: (typeof SEVERITIES)[number] | (typeof ENVIRONMENTS)[number]
) =>
  renderToStaticMarkup(
    <Select.Root>
      <Select.Item value={option.code}>
        <ChoiceOptionRow option={option} />
      </Select.Item>
    </Select.Root>
  )

const trigger = (value: string) =>
  renderToStaticMarkup(
    <ChoiceSelect
      id="sev"
      labelledBy="sev-label"
      name="severidade"
      onBlur={() => {}}
      onValueChange={() => {}}
      options={SEVERITIES}
      placeholder="Escolha a severidade"
      value={value}
    />
  )

describe('ChoiceOptionRow', () => {
  it('shows the label and a muted description', () => {
    const html = row(SEVERITIES[0])
    expect(html).toContain('>Baixa<')
    expect(html).toMatch(
      /class="text-muted-foreground[^"]*">Incômodo, há alternativa\.</
    )
  })

  it('puts only risky options in discreet red ("Crítica", "Produção")', () => {
    for (const option of [...SEVERITIES, ...ENVIRONMENTS]) {
      const red = /class="text-destructive"[^>]*>[^<]*</.test(row(option))
      expect(red).toBe(
        option.label === 'Crítica' || option.label === 'Produção'
      )
    }
  })
})

describe('ChoiceSelect', () => {
  it('is a combobox named by the visible label', () => {
    const html = trigger('')
    expect(html).toMatch(/role="combobox"/)
    expect(html).toContain('aria-labelledby="sev-label"')
    expect(html).toContain('Escolha a severidade')
  })

  it('shows the chosen label, in red when it is risky', () => {
    expect(trigger('alta')).toContain('>Alta<')
    expect(trigger('critica')).toMatch(
      /role="combobox"[^>]*class="[^"]*text-destructive/
    )
    expect(trigger('alta')).not.toMatch(
      /role="combobox"[^>]*class="[^"]*text-destructive/
    )
  })
})
