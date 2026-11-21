import { env } from '../env'

export const parseApiUrl = ({ raw }: { raw: string | undefined }): string => {
  const value = raw?.trim() || env.apiBaseUrl
  let url: URL

  try {
    url = new URL(value)
  } catch {
    throw new Error('API_URL must be an absolute https URL')
  }

  if (url.protocol !== 'https:') {
    throw new Error('API_URL must use https')
  }

  if (url.username || url.password) {
    throw new Error('API_URL must not contain credentials')
  }

  if (url.pathname !== '/' || url.search || url.hash) {
    throw new Error(
      'API_URL must be an origin, with no path, query or fragment'
    )
  }

  return url.origin
}
