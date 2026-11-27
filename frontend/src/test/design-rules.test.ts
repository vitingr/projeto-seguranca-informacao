import { readFileSync, readdirSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const RULES = [
  {
    name: 'shadow class',
    pattern:
      /(?<![\w-])(?:drop-|inset-|text-)?shadow-(?!none(?![\w-]))[\w[\]()/.,%-]+/g
  },
  {
    name: 'box-shadow',
    pattern: /box-shadow\s*:(?!\s*none\s*[;}!])[^;}\n]*|boxShadow/g
  },
  {
    name: 'gradient',
    pattern:
      /\bbg-(?:gradient|linear|radial|conic)\b|(?:repeating-)?(?:linear|radial|conic)-gradient/g
  },
  {
    name: 'tracking class',
    pattern: /(?<![\w-])tracking-[\w[\]()/.,%-]+/g
  },
  {
    name: 'radius other than sm',
    pattern:
      /(?<![\w-])rounded(?:-(?:tl|tr|br|bl|ss|se|es|ee|t|r|b|l|s|e))?(?:-(?!(?:sm|none|full)(?![\w-]))[\w[\]()/.,%-]+)?(?![\w-])/g
  }
]

const ALLOWED_RADIUS = /^rounded(?:-[a-z]{1,2})?-(?:sm|none|full)$/

const violations = (source: string) =>
  RULES.flatMap(({ name, pattern }) =>
    [...source.matchAll(pattern)]
      .map(match => match[0])
      .filter(found => !ALLOWED_RADIUS.test(found))
      .map(found => `${name}: ${found}`)
  )

describe('violations', () => {
  it.each([
    'shadow-md',
    'hover:shadow-lg',
    'drop-shadow-md',
    'inset-shadow-sm',
    'shadow-[0_1px_2px_black]',
    'box-shadow: 0 1px 2px black;',
    '[box-shadow:0_1px_black]',
    'style={{ boxShadow: x }}',
    'bg-gradient-to-r',
    'bg-linear-to-r',
    'bg-radial',
    'background: linear-gradient(red, blue);',
    'tracking-tight',
    'md:tracking-wide',
    'rounded-lg',
    'rounded-md',
    'rounded',
    'rounded-t-lg',
    'rounded-[min(var(--radius-md),10px)]',
    'in-data-[slot=button-group]:rounded-lg'
  ])('refuses %s', source => {
    expect(violations(`<div className="${source}" />`)).not.toEqual([])
  })

  it.each([
    'rounded-sm',
    'rounded-full',
    'rounded-none',
    'rounded-t-sm',
    'focus-visible:rounded-sm',
    'shadow-none',
    'hover:shadow-none',
    'box-shadow: none;',
    '--shadow-md: 0 0 transparent;',
    '--radius-sm: calc(var(--radius) * 0.6);',
    'focus-visible:ring-3 outline-none',
    'border px-3 text-sm'
  ])('allows %s', source => {
    expect(violations(`<div className="${source}" />`)).toEqual([])
  })
})

describe('source files', () => {
  const srcDir = fileURLToPath(new URL('..', import.meta.url))
  const self = fileURLToPath(import.meta.url)
  const files = readdirSync(srcDir, { recursive: true, withFileTypes: true })
    .filter(entry => entry.isFile() && /\.(?:ts|tsx|css)$/.test(entry.name))
    .map(entry => path.join(entry.parentPath, entry.name))
    .filter(file => path.resolve(file) !== path.resolve(self))

  it('finds the source files', () => {
    expect(files.some(file => file.endsWith('globals.css'))).toBe(true)
    expect(files.some(file => file.endsWith('layout.tsx'))).toBe(true)
  })

  it('follow the design rules', () => {
    const found = files.flatMap(file =>
      violations(readFileSync(file, 'utf8')).map(
        violation => `${path.relative(srcDir, file)}: ${violation}`
      )
    )
    expect(found).toEqual([])
  })
})

describe('pointer cursor', () => {
  const css = readFileSync(
    fileURLToPath(new URL('../styles/globals.css', import.meta.url)),
    'utf8'
  )
    .replace(/\/\*[\s\S]*?\*\//g, '')
    .replaceAll('"', "'")

  const layerBodies = (name: string) => {
    const bodies: string[] = []
    let start = css.indexOf(`@layer ${name} {`)
    while (start >= 0) {
      const open = css.indexOf('{', start)
      let depth = 0
      let end = open
      while (end < css.length) {
        if (css[end] === '{') depth += 1
        if (css[end] === '}') depth -= 1
        if (depth === 0) break
        end += 1
      }
      bodies.push(css.slice(open + 1, end))
      start = css.indexOf(`@layer ${name} {`, end)
    }
    return bodies
  }

  const pointerSelectors = (source: string) =>
    [...source.matchAll(/([^{};]+)\{[^{}]*cursor\s*:\s*pointer/g)].flatMap(
      match => match[1].split(',').map(selector => selector.trim())
    )

  const base = layerBodies('base').flatMap(pointerSelectors)

  it('is set only in the base layer, without !important', () => {
    expect(base.length).toBeGreaterThan(0)
    expect(pointerSelectors(css)).toEqual(base)
    expect(css).not.toMatch(/cursor\s*:[^;}]*!important/)
  })

  it('applies to buttons and to role="button"', () => {
    const starts = (prefix: string) =>
      base.some(selector => selector.startsWith(prefix))
    expect(starts('button:')).toBe(true)
    expect(starts("[role='button']:")).toBe(true)
  })

  it('leaves out disabled and aria-disabled buttons', () => {
    for (const selector of base) {
      expect(selector).toContain(":not([aria-disabled='true'])")
      if (selector.startsWith('button')) {
        expect(selector).toContain(':not(:disabled)')
      }
    }
  })
})
