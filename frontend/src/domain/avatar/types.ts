export interface AvatarFigure {
  /** The squares of the figure, row by row (`AVATAR_GRID` × `AVATAR_GRID`). */
  cells: boolean[]
  /** One or two uppercase letters, "?" for an empty login. */
  initials: string
  /** Which tint of the theme, from 0 to `AVATAR_TINT_COUNT - 1`. */
  tint: number
}
