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
