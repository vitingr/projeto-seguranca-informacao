import { type Page, expect } from '@playwright/test'

export const expectNothingInBrowserStorage = async (page: Page) => {
  const storage = await page.evaluate(async () => ({
    cookie: document.cookie,
    indexedDb: (await indexedDB.databases()).map(db => db.name),
    local: Object.keys(localStorage),
    session: Object.keys(sessionStorage)
  }))
  expect(storage).toEqual({ cookie: '', indexedDb: [], local: [], session: [] })
}
