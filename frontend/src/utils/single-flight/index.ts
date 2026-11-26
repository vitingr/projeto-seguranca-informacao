/**
 * Wraps an async call so that only one runs at a time: a call made while
 * another is pending returns `null` at once and starts nothing. Used to
 * keep a double click from sending twice (spec 001 AC-5.1).
 */
export const singleFlight = <Args extends unknown[], Result>(
  call: (...args: Args) => Promise<Result>
): ((...args: Args) => Promise<null | Result>) => {
  let pending = false
  return async (...args) => {
    if (pending) return null
    pending = true
    try {
      return await call(...args)
    } finally {
      pending = false
    }
  }
}
