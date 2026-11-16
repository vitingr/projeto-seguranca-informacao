import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'

import {
  CADASTRO_REQUEST_FIELDS,
  CSRF_HEADER,
  ERROR_CODES,
  LOGIN_REQUEST_FIELDS,
  ROLES,
  USER_RESPONSE_FIELDS
} from './types'

interface Schema {
  const?: string
  description?: string
  enum?: string[]
  properties?: Record<string, Schema>
  required?: string[]
}

const contract = JSON.parse(
  readFileSync(
    new URL('../../../../backend/api/openapi.json', import.meta.url),
    'utf8'
  )
) as { components: { schemas: Record<string, Schema> } }

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
    ['UserResponse', USER_RESPONSE_FIELDS]
  ] as const)('%s fields', (name, expected) => {
    expect(fields(name)).toEqual([...expected])
  })

  it('AreaResponse fields', () => {
    expect(fields('AreaResponse')).toEqual(['area'])
  })
})
