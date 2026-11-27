import 'server-only'

import { cookies } from 'next/headers'
import { notFound, redirect } from 'next/navigation'

import { parseApiUrl } from '@/config/api'
import { env } from '@/config/env'
import { SESSION_COOKIE, createBackendGet } from '@/lib/backend'

import { createTicketLoaders } from './loaders'

// Server reads of ticket tracking for the pages (plan P-5); the logic and
// its tests are in loaders.ts.

const backendGet = createBackendGet(
  parseApiUrl({ raw: env.apiBaseUrl }),
  (...a) => fetch(...a)
)

export const {
  loadAdminThread,
  loadAdminTicket,
  loadOwnThread,
  loadOwnTicket,
  loadOwnTickets,
  loadQueue,
  loadQueueAuthors
} = createTicketLoaders({
  cookie: async () => (await cookies()).get(SESSION_COOKIE)?.value,
  get: backendGet,
  notFound,
  redirect
})
