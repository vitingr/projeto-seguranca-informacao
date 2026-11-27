import type { AvatarFigure } from './types'

export const AVATAR_TINT_COUNT = 8

export const AVATAR_GRID = 5

export const normalizeLogin = (login: string): string =>
  login.trim().toLowerCase()

export const fnv1a = (text: string): number => {
  let hash = 0x811c9dc5
  for (const byte of new TextEncoder().encode(text)) {
    hash ^= byte
    hash = Math.imul(hash, 0x01000193)
  }
  return hash >>> 0
}

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

const figureCells = (hash: number): boolean[] => {
  const half = Math.ceil(AVATAR_GRID / 2)
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

export const avatarFor = (login: string): AvatarFigure => {
  const hash = fnv1a(normalizeLogin(login))
  return {
    cells: figureCells(hash),
    initials: avatarInitials(login),
    tint: hash % AVATAR_TINT_COUNT
  }
}
