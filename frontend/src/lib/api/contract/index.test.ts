import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'

import { AUDIT_EVENT_TYPES } from '@/domain/audit'
import {
  POST_BODY_MAX,
  POST_PAGE_MAX,
  POST_TITLE_MAX,
  REMOVAL_REASON_MAX
} from '@/domain/post'
import { ROLES } from '@/domain/roles'

import {
  AUDIT_ACTOR_FIELDS,
  AUDIT_EVENT_FIELDS,
  AUDIT_PAGE_FIELDS,
  AUDIT_TARGET_FIELDS,
  CADASTRO_REQUEST_FIELDS,
  CSRF_HEADER,
  ERROR_CODES,
  LOGIN_REQUEST_FIELDS,
  POST_AUTHOR_FIELDS,
  POST_PAGE_FIELDS,
  POST_REMOVAL_FIELDS,
  POST_REQUEST_FIELDS,
  POST_RESPONSE_FIELDS,
  REMOVAL_REQUEST_FIELDS,
  USER_RESPONSE_FIELDS
} from '.'

interface Schema {
  const?: string
  description?: string
  enum?: string[]
  maximum?: number
  maxLength?: number
  properties?: Record<string, Schema>
  required?: string[]
}

const contract = JSON.parse(
  readFileSync(
    new URL('../../../../../backend/api/openapi.json', import.meta.url),
    'utf8'
  )
) as {
  components: {
    parameters: Record<string, { schema: Schema }>
    schemas: Record<string, Schema>
  }
}

const schema = (name: string): Schema => {
  const s = contract.components.schemas[name]
  if (!s) throw new Error(`schema ${name} not found in the contract`)
  return s
}

const fields = (name: string) => Object.keys(schema(name).properties ?? {})

describe('API types match backend/api/openapi.json', () => {
  it('roles', () => {
    expect([...ROLES]).toEqual(schema('Role').enum)
  })

  it('error codes listed in ErrorResponse.error', () => {
    const description = schema('ErrorResponse').properties?.error?.description
    const listed = [...(description ?? '').matchAll(/`([a-z_]+)`/g)].map(
      m => m[1]
    )
    expect([...ERROR_CODES]).toEqual(listed)
  })

  it('ErrorResponse required fields', () => {
    expect(schema('ErrorResponse').required).toEqual(['error', 'message'])
    expect(fields('ErrorResponse')).toEqual(['error', 'message', 'request_id'])
  })

  it('CSRF header', () => {
    expect(schema('CsrfResponse').properties?.header?.const).toBe(CSRF_HEADER)
    expect(fields('CsrfResponse')).toEqual(['csrf_token', 'header'])
  })

  it.each([
    ['LoginRequest', LOGIN_REQUEST_FIELDS],
    ['CadastroRequest', CADASTRO_REQUEST_FIELDS],
    ['UserResponse', USER_RESPONSE_FIELDS],
    ['PostRequest', POST_REQUEST_FIELDS],
    ['Post', POST_RESPONSE_FIELDS],
    ['PostAuthor', POST_AUTHOR_FIELDS],
    ['PostPage', POST_PAGE_FIELDS],
    ['PostRemoval', POST_REMOVAL_FIELDS],
    ['RemovalRequest', REMOVAL_REQUEST_FIELDS],
    ['AuditActor', AUDIT_ACTOR_FIELDS],
    ['AuditTarget', AUDIT_TARGET_FIELDS],
    ['AuditEvent', AUDIT_EVENT_FIELDS],
    ['AuditPage', AUDIT_PAGE_FIELDS]
  ] as const)('%s fields', (name, expected) => {
    expect(fields(name)).toEqual([...expected])
  })

  it('AreaResponse fields', () => {
    expect(fields('AreaResponse')).toEqual(['area'])
  })

  it('audit events and the removal limit (006 D-2, D-4)', () => {
    expect([...AUDIT_EVENT_TYPES]).toEqual(schema('AuditEventType').enum)
    expect(schema('RemovalRequest').properties?.reason?.maxLength).toBe(
      REMOVAL_REASON_MAX
    )
  })

  it('post limits (005 D-4, D-5)', () => {
    const request = schema('PostRequest').properties
    expect(request?.title?.maxLength).toBe(POST_TITLE_MAX)
    expect(request?.body?.maxLength).toBe(POST_BODY_MAX)
    expect(contract.components.parameters.Page?.schema.maximum).toBe(
      POST_PAGE_MAX
    )
  })
})
