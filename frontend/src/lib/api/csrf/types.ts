export interface CsrfStore {
  /** Discards the token (logout, 401). A pending fetch is discarded too. */
  clear(): void
  /** The current token, fetched once when there is none. */
  get(): Promise<string>
  /** Always fetches a new token (after login, after a 403). */
  refresh(): Promise<string>
}
