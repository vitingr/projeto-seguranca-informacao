// Types of the backend API, written by hand from backend/api/openapi.json.
// types.test.ts checks them against the contract.

export const ROLES = ['admin', 'comum'] as const
export type Role = (typeof ROLES)[number]

export const ERROR_CODES = [
  'bad_request',
  'unauthorized',
  'forbidden',
  'not_found',
  'conflict',
  'unprocessable_entity',
  'internal'
] as const
export type ErrorCode = (typeof ERROR_CODES)[number]

export const CSRF_HEADER = 'X-CSRF-Token'

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

export const LOGIN_REQUEST_FIELDS = [
  'login',
  'senha'
] as const satisfies readonly (keyof LoginRequest)[]

export const CADASTRO_REQUEST_FIELDS = [
  'login',
  'senha',
  'papel'
] as const satisfies readonly (keyof CadastroRequest)[]

export const USER_RESPONSE_FIELDS = [
  'user_id',
  'login',
  'role'
] as const satisfies readonly (keyof UserResponse)[]
