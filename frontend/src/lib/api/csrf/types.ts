export interface CreaateCsrfStoreProps {
  fetchFn: typeof fetch
}

export interface CsrfStore {
  clear(): void
  get(): Promise<string>
  refresh(): Promise<string>
}
