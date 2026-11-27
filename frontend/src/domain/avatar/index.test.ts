import { describe, expect, it } from 'vitest'

import {
  AVATAR_GRID,
  AVATAR_TINT_COUNT,
  avatarFor,
  avatarInitials,
  fnv1a,
  normalizeLogin
} from '.'

describe('fnv1a', () => {
  it('matches the published FNV-1a 32-bit vectors', () => {
    expect(fnv1a('')).toBe(0x811c9dc5)
    expect(fnv1a('a')).toBe(0xe40c292c)
    expect(fnv1a('foobar')).toBe(0xbf9cf968)
  })
})

describe('avatarInitials', () => {
  it.each([
    ['maria.silva', 'MS'],
    ['joao', 'J'],
    ['ana_paula', 'AP'],
    ['jose-carlos.souza', 'JC'],
    ['a.b.c', 'AB'],
    ['.x', 'X'],
    ['  Maria.Silva  ', 'MS'],
    ['2fa.admin', '2A'],
    ['', '?'],
    ['..', '?']
  ])('%j → %j', (login, initials) => {
    expect(avatarInitials(login)).toBe(initials)
  })
})

describe('avatarFor', () => {
  it('always gives the same initials, tint and figure for the same login', () => {
    const first = avatarFor('maria.silva')
    expect(avatarFor('maria.silva')).toEqual(first)
    expect(avatarFor(' MARIA.SILVA ')).toEqual(first)
  })

  it('has a mirrored, never empty grid of the planned size', () => {
    for (const login of ['maria.silva', 'joao', 'admin', 'x', 'e2e.comum']) {
      const { cells } = avatarFor(login)
      expect(cells).toHaveLength(AVATAR_GRID * AVATAR_GRID)
      expect(cells).toContain(true)
      for (let row = 0; row < AVATAR_GRID; row += 1) {
        const line = cells.slice(row * AVATAR_GRID, (row + 1) * AVATAR_GRID)
        expect(line).toEqual([...line].reverse())
      }
    }
  })

  it('spreads different logins over every tint', () => {
    const counts = Array.from({ length: AVATAR_TINT_COUNT }, () => 0)
    for (let index = 0; index < 400; index += 1) {
      counts[avatarFor(`user${index}.test`).tint] += 1
    }
    for (const count of counts) {
      expect(count).toBeGreaterThan(25)
      expect(count).toBeLessThan(80)
    }
  })

  it('gives different figures to different logins', () => {
    const figures = new Set(
      ['maria.silva', 'joao', 'ana.paula', 'carlos', 'admin', 'e2e.comum'].map(
        login => avatarFor(login).cells.join('')
      )
    )
    expect(figures.size).toBe(6)
  })
})

describe('normalizeLogin', () => {
  it('trims and lowercases', () => {
    expect(normalizeLogin('  Maria.Silva ')).toBe('maria.silva')
  })
})
