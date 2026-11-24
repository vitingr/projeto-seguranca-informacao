// Entrance stagger for lists, in order: classes (anim-delay-* in
// globals.css), never inline styles. Use with `motion-safe:animate-rise`.
export const STAGGER = [
  'motion-safe:anim-delay-0',
  'motion-safe:anim-delay-75',
  'motion-safe:anim-delay-150',
  'motion-safe:anim-delay-225',
  'motion-safe:anim-delay-300',
  'motion-safe:anim-delay-375'
] as const

/** The delay class for item `index`; items past the list enter together. */
export const staggerAt = (index: number) =>
  STAGGER[Math.min(index, STAGGER.length - 1)]
