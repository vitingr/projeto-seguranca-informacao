import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'

import { SEND_STEPS, SEND_TEXTS, SendProgress, progressHeading } from '.'

const render = (props: Partial<Parameters<typeof SendProgress>[0]> = {}) =>
  renderToStaticMarkup(
    <SendProgress
      chips={['Outro', 'Título', 'Descrição']}
      elapsedMs={420}
      phase="registering"
      {...props}
    />
  )

const steps = (html: string) =>
  [...html.matchAll(/<li[^>]*data-state="(\w+)"[^>]*>.*?<\/li>/g)].map(
    match => ({ html: match[0], state: match[1] })
  )

describe('SendProgress', () => {
  it('names only steps that really happen (AC-5.5)', () => {
    expect(SEND_STEPS).toEqual([
      'Validando campos',
      'Registrando ticket',
      'Preparando confirmação'
    ])
    const all = Object.values(SEND_TEXTS).flatMap(value =>
      typeof value === 'string' ? [value] : Object.values(value)
    )
    for (const text of [...SEND_STEPS, ...all]) {
      expect(text).not.toMatch(
        /malware|v[ií]rus|criptograf|seguran|antiv|amea|verificando/i
      )
    }
  })

  it('while pending: fields checked, "Registrando ticket" in progress, no confirmation step', () => {
    const list = steps(render())
    expect(list.map(step => step.state)).toEqual(['done', 'active'])
    expect(list[0].html).toContain('Validando campos')
    expect(list[1].html).toContain('Registrando ticket')
    expect(render()).not.toContain('Preparando confirmação')
    expect(render()).not.toContain(SEND_TEXTS.done)
  })

  it('on an error: "Registrando ticket" failed and nothing after it (AC-5.7)', () => {
    const html = render({ phase: 'failed' })
    expect(steps(html).map(step => step.state)).toEqual(['done', 'failed'])
    expect(html).toContain(SEND_TEXTS.failed)
    expect(html).not.toContain('Preparando confirmação')
    expect(html).not.toContain(SEND_TEXTS.done)
  })

  it('only after the ticket was created: every step done and the check row (AC-5.8)', () => {
    const html = render({ phase: 'done' })
    expect(steps(html).map(step => step.state)).toEqual([
      'done',
      'done',
      'done'
    ])
    expect(html).toContain('Preparando confirmação')
    expect(html).toContain(SEND_TEXTS.done)
  })

  it('says each state in words and hides the decorations (AC-5.10)', () => {
    const html = render()
    expect(html).toContain(`<ol aria-label="${SEND_TEXTS.list}"`)
    expect(html).toContain(
      `<span class="sr-only"> ${SEND_TEXTS.states.done}</span>`
    )
    expect(html).toContain(
      `<span class="sr-only"> ${SEND_TEXTS.states.active}</span>`
    )
    expect(html).toMatch(/class="progress-squares[^"]*" aria-hidden="true"/)
    expect(render({ phase: 'failed' })).toContain(
      `<span class="sr-only"> ${SEND_TEXTS.states.failed}</span>`
    )
  })

  it('shows field names as chips, at most five and then "+N"', () => {
    const html = render({
      chips: ['Suporte em bug', 'A', 'B', 'C', 'D', 'E', 'F']
    })
    expect(html).toContain('>Suporte em bug</li>')
    expect(html).toContain('>D</li>')
    expect(html).not.toContain('>E</li>')
    expect(html).toContain('>+2</li>')
  })

  it('moves only under motion-safe, with no gradient (AC-5.9)', () => {
    for (const phase of ['registering', 'failed', 'done'] as const) {
      const html = render({ phase })
      const animated = [
        ...html.matchAll(/(?:^|\s|")([\w:-]*animate-[\w-]+)/g)
      ].map(match => match[1])
      expect(animated.length).toBeGreaterThan(0)
      for (const name of animated) {
        expect(name).toMatch(/^motion-safe:animate-/)
      }
      // Gradient classes are also refused by src/test/design-rules.test.ts.
      expect(html).not.toMatch(/gradient|shimmer/)
    }
  })

  it('blinks the squares only while the request is pending', () => {
    expect(render()).toContain('motion-safe:animate-blink')
    expect(render({ phase: 'done' })).not.toContain('animate-blink')
    expect(render({ phase: 'failed' })).not.toContain('animate-blink')
  })

  it('lets only the rows from `enterFrom` enter on the confirmation', () => {
    const html = render({ enterFrom: 3, phase: 'done' })
    const list = steps(html)
    expect(list[0].html).not.toContain('animate-enter-blur')
    expect(list[1].html).not.toContain('animate-enter-blur')
    expect(list[2].html).toContain('motion-safe:animate-enter-blur')
  })
})

describe('progressHeading', () => {
  it('counts whole seconds while pending and shows the real total after', () => {
    expect(progressHeading('registering', 1999)).toEqual({
      muted: '1 s',
      text: 'Enviando ticket'
    })
    expect(progressHeading('done', 420)).toEqual({
      muted: null,
      text: 'Enviado em 0,4 s'
    })
    expect(progressHeading('failed', 2350)).toEqual({
      muted: 'após 2,4 s',
      text: 'Envio interrompido'
    })
  })
})
