import type { Role } from '@/domain/roles/types'

export interface CurrentUser {
  createdAt?: string
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
