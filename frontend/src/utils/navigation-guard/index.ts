import type { PlainClick } from './types'

export const isPlainClick = (event: PlainClick): boolean =>
  event.button === 0 &&
  !event.altKey &&
  !event.ctrlKey &&
  !event.metaKey &&
  !event.shiftKey

export const leavesPage = (href: string, current: string): boolean => {
  const here = new URL(current)
  const target = new URL(href, here)
  return target.origin === here.origin && target.pathname !== here.pathname
}
