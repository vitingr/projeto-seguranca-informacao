import type { Role } from '@/domain/roles/types'

import type { CSRF_HEADER, ERROR_CODES } from '.'

export type ErrorCode = (typeof ERROR_CODES)[number]

export interface AreaResponse {
  area: string
}

export interface CadastroRequest {
  login: string
  papel: Role
  senha: string
}

export interface CsrfResponse {
  csrf_token: string
  header: typeof CSRF_HEADER
}

export interface ErrorResponse {
  error: string
  message: string
  request_id?: string
}

export interface LoginRequest {
  login: string
  senha: string
}

export interface UserResponse {
  login?: string
  role: Role
  user_id: number
}
