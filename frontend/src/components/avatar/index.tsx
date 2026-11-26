import { cn } from 'cn'

import { AVATAR_GRID, avatarFor } from '@/domain/avatar'

import type { AvatarProps } from './types'

/** One class pair per tint of `domain/avatar`, from the theme tokens. */
export const AVATAR_TINT_CLASSES = [
  'bg-avatar-1 text-avatar-1-foreground',
  'bg-avatar-2 text-avatar-2-foreground',
  'bg-avatar-3 text-avatar-3-foreground',
  'bg-avatar-4 text-avatar-4-foreground',
  'bg-avatar-5 text-avatar-5-foreground',
  'bg-avatar-6 text-avatar-6-foreground',
  'bg-avatar-7 text-avatar-7-foreground',
  'bg-avatar-8 text-avatar-8-foreground'
] as const

/**
 * The generated avatar (spec AC-7.2, plan 5.6): the initials of the login
 * on a round tile of one of the theme's tints, over a faint mirrored grid
 * of squares. Inline SVG and text only: no image, no link and no request,
 * so it fits the CSP. Decorative: the login is the text next to it.
 */
export const Avatar = ({ className, login }: AvatarProps) => {
  const { cells, initials, tint } = avatarFor(login)
  return (
    <span
      className={cn(
        'relative inline-flex size-16 shrink-0 items-center justify-center overflow-hidden rounded-full select-none',
        AVATAR_TINT_CLASSES[tint],
        className
      )}
      data-avatar-tint={tint}
      aria-hidden
    >
      <svg
        className="absolute inset-0 size-full opacity-20"
        fill="currentColor"
        focusable="false"
        shapeRendering="crispEdges"
        // The grid fills the tile; the circle crops its corners.
        viewBox={`0 0 ${AVATAR_GRID} ${AVATAR_GRID}`}
      >
        {cells.map((filled, index) =>
          filled ? (
            <rect
              height="1"
              key={index}
              width="1"
              x={index % AVATAR_GRID}
              y={Math.floor(index / AVATAR_GRID)}
            />
          ) : null
        )}
      </svg>
      <span className="relative text-xl font-semibold">{initials}</span>
    </span>
  )
}
