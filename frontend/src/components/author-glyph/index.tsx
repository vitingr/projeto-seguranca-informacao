import { cn } from 'cn'

import type { AuthorGlyphProps } from './types'

// FNV-1a: a stable, cheap hash, so a login always draws the same glyph on
// the server and in the browser.
const hash = (text: string) => {
  let value = 0x811c9dc5
  for (const char of text) {
    value ^= char.codePointAt(0) ?? 0
    value = Math.imul(value, 0x01000193)
  }
  return value >>> 0
}

/**
 * The 9 cells of a 3x3 grid, mirrored left to right, with at least three
 * dots on: a small mark that tells authors apart at a glance.
 */
export const glyphCells = (login: string): boolean[] => {
  let bits = hash(login) & 0b111111
  if (bits.toString(2).replaceAll('0', '').length < 3) bits |= 0b010101
  const cells: boolean[] = []
  for (let row = 0; row < 3; row++) {
    const side = (bits >> row) & 1
    const middle = (bits >> (row + 3)) & 1
    cells.push(side === 1, middle === 1, side === 1)
  }
  return cells
}

export const AuthorGlyph: React.FC<AuthorGlyphProps> = ({
  className,
  login
}) => (
  <span
    aria-hidden="true"
    className={cn('grid size-4 shrink-0 grid-cols-3 gap-px', className)}
    data-testid="author-glyph"
  >
    {glyphCells(login).map((on, index) => (
      <span
        className={cn('rounded-xs', on ? 'bg-current' : 'bg-current/12')}
        // The cells never move or change order.
        key={index}
      />
    ))}
  </span>
)
