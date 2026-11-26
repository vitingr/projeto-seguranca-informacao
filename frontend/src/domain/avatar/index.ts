import type { AvatarFigure } from './types'

// The avatar of the profile (spec AC-7.2, plan 5.6): made in the browser
// from the login alone, always the same for the same login. Pure: no
// network, no image, no randomness.

/** How many tints the theme has (`--avatar-1` to `--avatar-8`). */
export const AVATAR_TINT_COUNT = 8

/** The figure is a GRID × GRID grid of squares, mirrored left to right. */
export const AVATAR_GRID = 5

/** The login as the backend stores it: trimmed and lowercase. */
export const normalizeLogin = (login: string): string =>
  login.trim().toLowerCase()

/** FNV-1a, 32 bits, over the UTF-8 bytes of the text. */
export const fnv1a = (text: string): number => {
  let hash = 0x811c9dc5
  for (const byte of new TextEncoder().encode(text)) {
    hash ^= byte
    hash = Math.imul(hash, 0x01000193)
  }
  return hash >>> 0
}

/**
 * The first letter of the first two parts of the login, split on ".", "_"
 * and "-", in uppercase: "maria.silva" → "MS", "joao" → "J".
 */
export const avatarInitials = (login: string): string => {
  const parts = normalizeLogin(login)
    .split(/[._-]+/)
    .filter(part => part !== '')
  const initials = parts
    .slice(0, 2)
    .map(part => Array.from(part)[0])
    .join('')
    .toUpperCase()
  return initials === '' ? '?' : initials
}

/**
 * The cells of the figure, row by row. The left half and the middle column
 * come from the hash and the right half mirrors them, so the figure is
 * symmetric like a mark, never a texture. It is never empty.
 */
const figureCells = (hash: number): boolean[] => {
  const half = Math.ceil(AVATAR_GRID / 2)
  // The low bits pick the tint; the figure uses the ones above them.
  const bits = hash >>> 8
  const cells: boolean[] = []
  for (let row = 0; row < AVATAR_GRID; row += 1) {
    for (let column = 0; column < AVATAR_GRID; column += 1) {
      const source = column < half ? column : AVATAR_GRID - 1 - column
      cells.push(((bits >>> (row * half + source)) & 1) === 1)
    }
  }
  if (!cells.includes(true)) cells[Math.floor(cells.length / 2)] = true
  return cells
}

/** The initials, the tint and the figure of a login. */
export const avatarFor = (login: string): AvatarFigure => {
  const hash = fnv1a(normalizeLogin(login))
  return {
    cells: figureCells(hash),
    initials: avatarInitials(login),
    tint: hash % AVATAR_TINT_COUNT
  }
}
