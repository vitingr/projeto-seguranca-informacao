// The backend's post rules (005 D-4, D-5), checked before sending so the
// user gets a field message. The backend stays the authority.

export const POST_TITLE_MAX = 120
export const POST_BODY_MAX = 2000
export const POST_PAGE_MAX = 10000

/** Length as the backend counts it: code points after trimming. */
export const postLength = (value: string) => [...value.trim()].length

// Unicode control characters (Cc), like Go's unicode.IsControl.
const CONTROL = /\p{Cc}/u
const CONTROL_EXCEPT_LINE_BREAKS_AND_TAB = /(?![\n\r\t])\p{Cc}/u

export const titleHasControl = (value: string) => CONTROL.test(value)

export const bodyHasControl = (value: string) =>
  CONTROL_EXCEPT_LINE_BREAKS_AND_TAB.test(value)

/**
 * The wall page in the URL (`?pagina=`): an integer from 1 to
 * POST_PAGE_MAX. Anything else, or no value, is page 1.
 */
export const parsePageParam = (raw: string | string[] | undefined): number => {
  if (typeof raw !== 'string' || !/^[1-9]\d{0,4}$/.test(raw)) return 1
  const page = Number(raw)
  return page <= POST_PAGE_MAX ? page : 1
}
