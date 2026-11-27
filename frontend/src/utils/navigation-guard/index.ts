import type { PlainClick } from './types'

// Pure rules of the unsent data guard (containers/leave-guard), tested in
// containers/ticket-builder/history.test.ts through their builder names.

/** A click that follows the link in this tab (no modifier, main button). */
export const isPlainClick = (event: PlainClick): boolean =>
  event.button === 0 &&
  !event.altKey &&
  !event.ctrlKey &&
  !event.metaKey &&
  !event.shiftKey

/** Whether a link leads out of the current page within this site. */
export const leavesPage = (href: string, current: string): boolean => {
  const here = new URL(current)
  const target = new URL(href, here)
  return target.origin === here.origin && target.pathname !== here.pathname
}
