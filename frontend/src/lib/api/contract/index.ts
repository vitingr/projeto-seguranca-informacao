// Types of the backend API, written by hand from backend/api/openapi.json.
// index.test.ts checks them against the contract.

import type {
  AuditActorResponse,
  AuditEventResponse,
  AuditPageResponse,
  AuditTargetResponse,
  CadastroRequest,
  LoginRequest,
  PostAuthorResponse,
  PostPageResponse,
  PostRemovalResponse,
  PostRequest,
  PostResponse,
  RemovalRequest,
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
  'mine',
  'removal'
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

export const POST_REMOVAL_FIELDS = [
  'reason',
  'removed_at'
] as const satisfies readonly (keyof PostRemovalResponse)[]

export const REMOVAL_REQUEST_FIELDS = [
  'reason'
] as const satisfies readonly (keyof RemovalRequest)[]

export const AUDIT_ACTOR_FIELDS = [
  'user_id',
  'login'
] as const satisfies readonly (keyof AuditActorResponse)[]

export const AUDIT_TARGET_FIELDS = [
  'type',
  'id',
  'login'
] as const satisfies readonly (keyof AuditTargetResponse)[]

export const AUDIT_EVENT_FIELDS = [
  'id',
  'event',
  'actor',
  'target',
  'ip',
  'created_at'
] as const satisfies readonly (keyof AuditEventResponse)[]

export const AUDIT_PAGE_FIELDS = [
  'items',
  'page',
  'page_size',
  'has_next'
] as const satisfies readonly (keyof AuditPageResponse)[]
