import { describe, expect, it } from 'vitest'

import { glyphCells } from '.'

describe('glyphCells', () => {
  it('always draws the same glyph for a login', () => {
    expect(glyphCells('maria')).toEqual(glyphCells('maria'))
  })

  it('is mirrored left to right', () => {
    for (const login of ['maria', 'joao.silva', 'e2e.abc', 'x']) {
      const cells = glyphCells(login)
      expect(cells).toHaveLength(9)
      for (let row = 0; row < 3; row++) {
        expect(cells[row * 3]).toBe(cells[row * 3 + 2])
      }
    }
  })

  it('never draws fewer than three dots', () => {
    for (let i = 0; i < 200; i++) {
      const on = glyphCells(`user${i}`).filter(Boolean).length
      expect(on).toBeGreaterThanOrEqual(3)
    }
  })

  it('tells most logins apart', () => {
    const glyphs = new Set(
      Array.from({ length: 40 }, (_, i) =>
        glyphCells(`user${i}`).map(Number).join('')
      )
    )
    expect(glyphs.size).toBeGreaterThan(15)
  })
})
