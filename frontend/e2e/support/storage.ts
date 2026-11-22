import { type Page, expect } from '@playwright/test'

/**
 * The CSRF token and the session cookie are never visible to page storage
 * (spec D-2, D-3): no Web Storage entries, no IndexedDB database and no
 * cookie readable by JavaScript.
 */
export const expectNothingInBrowserStorage = async (page: Page) => {
  const storage = await page.evaluate(async () => ({
    cookie: document.cookie,
    indexedDb: (await indexedDB.databases()).map(db => db.name),
    local: Object.keys(localStorage),
    session: Object.keys(sessionStorage)
  }))
  expect(storage).toEqual({ cookie: '', indexedDb: [], local: [], session: [] })
}
