// Types of the backend API, written by hand from backend/api/openapi.json.
// index.test.ts checks them against the contract.

import type {
  CadastroRequest,
  LoginRequest,
  PostAuthorResponse,
  PostPageResponse,
  PostRequest,
  PostResponse,
  UserResponse
} from './types'

export const ERROR_CODES = [
  'bad_request',
  'unauthorized',
  'forbidden',
  'not_found',
  'conflict',
  'unprocessable_entity',
  'internal'
] as const

export const CSRF_HEADER = 'X-CSRF-Token'

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

export const POST_REQUEST_FIELDS = [
  'title',
  'body'
] as const satisfies readonly (keyof PostRequest)[]

export const POST_RESPONSE_FIELDS = [
  'id',
  'title',
  'body',
  'author',
  'created_at',
  'updated_at',
  'mine'
] as const satisfies readonly (keyof PostResponse)[]

export const POST_AUTHOR_FIELDS = [
  'user_id',
  'login'
] as const satisfies readonly (keyof PostAuthorResponse)[]

export const POST_PAGE_FIELDS = [
  'items',
  'page',
  'page_size',
  'has_next'
] as const satisfies readonly (keyof PostPageResponse)[]
