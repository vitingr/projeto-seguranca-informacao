import type { Role } from '@/domain/roles/types'

export interface CurrentUser {
  /** When the user was registered (ISO 8601), for "Membro desde" (US-7). */
  createdAt?: string
  /** The login, for the profile and its avatar (US-7). */
  login?: string
  role: Role
  userId: number
}

export type Session =
  | { kind: 'anonymous' }
  | { kind: 'expired' }
  | { kind: 'user'; user: CurrentUser }

export type Access =
  | { kind: 'allowed'; user: CurrentUser }
  | { kind: 'forbidden'; user: CurrentUser }
  | { kind: 'login'; url: string }

export type AreaPath = '/api/administradores' | '/api/usuarios'

export type AreaResult =
  | { area: string; kind: 'ok' }
  | { kind: 'forbidden' }
  | { kind: 'login'; url: string }
