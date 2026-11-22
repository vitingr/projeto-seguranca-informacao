// Returned after a navigation to the login page: callers render nothing
// more for a page that is going away.
export const never = <T>() => new Promise<T>(() => {})
