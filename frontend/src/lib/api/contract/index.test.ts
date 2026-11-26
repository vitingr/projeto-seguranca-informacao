import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'

import { ROLES } from '@/domain/roles'

import {
  ACCESS_LEVELS,
  CADASTRO_REQUEST_FIELDS,
  CREATE_TICKET_REQUEST_FIELDS,
  CSRF_HEADER,
  ENVIRONMENTS,
  ERROR_CODES,
  LOGIN_REQUEST_FIELDS,
  SEVERITIES,
  TICKET_DATA_FIELDS,
  TICKET_RESPONSE_FIELDS,
  TICKET_STATUSES,
  TICKET_TYPES,
  USER_RESPONSE_FIELDS
} from '.'

interface Schema {
  $ref?: string
  additionalProperties?: boolean
  const?: string
  description?: string
  discriminator?: { mapping: Record<string, string>; propertyName: string }
  enum?: string[]
  properties?: Record<string, Schema>
  required?: string[]
}

interface Response {
  $ref?: string
  content?: Record<string, { example?: unknown; schema: Schema }>
  headers?: Record<string, { schema: Schema & { maximum?: number } }>
}

const contract = JSON.parse(
  readFileSync(
    new URL('../../../../../backend/api/openapi.json', import.meta.url),
    'utf8'
  )
) as {
  components: {
    responses: Record<string, Response>
    schemas: Record<string, Schema>
  }
  paths: Record<
    string,
    Record<
      string,
      {
        requestBody: { content: Record<string, { schema: Schema }> }
        responses: Record<string, Response>
      }
    >
  >
}

const schema = (name: string): Schema => {
  const s = contract.components.schemas[name]
  if (!s) throw new Error(`schema ${name} not found in the contract`)
  return s
}

/** The schema name a `$ref` points to. */
const refName = (ref: string | undefined) =>
  (ref ?? '').replace(/^#\/components\/(schemas|responses)\//, '')

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

describe('ticket types match backend/api/openapi.json', () => {
  it.each([
    ['TicketType', TICKET_TYPES],
    ['TicketStatus', TICKET_STATUSES],
    ['AccessLevel', ACCESS_LEVELS],
    ['Severity', SEVERITIES],
    ['Environment', ENVIRONMENTS]
  ] as const)('%s enum', (name, expected) => {
    expect(schema(name).enum).toEqual([...expected])
  })

  it('CreateTicketRequest is discriminated by tipo, one schema per type', () => {
    const { discriminator } = schema('CreateTicketRequest')
    expect(discriminator?.propertyName).toBe('tipo')
    expect(Object.keys(discriminator?.mapping ?? {})).toEqual([...TICKET_TYPES])
  })

  it.each(TICKET_TYPES)('request of type %s', type => {
    const mapping = schema('CreateTicketRequest').discriminator?.mapping ?? {}
    const request = schema(refName(mapping[type]))
    expect(Object.keys(request.properties ?? {})).toEqual([
      ...CREATE_TICKET_REQUEST_FIELDS
    ])
    expect(request.required).toEqual([...CREATE_TICKET_REQUEST_FIELDS])
    expect(request.additionalProperties).toBe(false)
    expect(request.properties?.tipo?.const).toBe(type)

    const data = schema(refName(request.properties?.dados?.$ref))
    const keys: readonly string[] = TICKET_DATA_FIELDS[type]
    expect(Object.keys(data.properties ?? {})).toEqual([...keys])
    // Every key is required except the deploy notes.
    expect(data.required).toEqual(keys.filter(key => key !== 'observacoes'))
    expect(data.additionalProperties).toBe(false)
  })

  it.each([
    ['AccessTicketData', 'nivel_acesso', 'AccessLevel'],
    ['BugTicketData', 'severidade', 'Severity'],
    ['DeployTicketData', 'ambiente', 'Environment']
  ])('%s.%s uses %s', (data, field, enumName) => {
    expect(refName(schema(data).properties?.[field]?.$ref)).toBe(enumName)
  })

  it('TicketResponse fields, all required', () => {
    expect(fields('TicketResponse')).toEqual([...TICKET_RESPONSE_FIELDS])
    expect(schema('TicketResponse').required).toEqual([
      ...TICKET_RESPONSE_FIELDS
    ])
  })

  it('POST /api/tickets: request, 201 and 429', () => {
    const operation = contract.paths['/api/tickets']?.post
    expect(operation).toBeDefined()
    const json = 'application/json'
    expect(refName(operation.requestBody.content[json]?.schema.$ref)).toBe(
      'CreateTicketRequest'
    )
    expect(
      refName(operation.responses['201']?.content?.[json]?.schema.$ref)
    ).toBe('TicketResponse')
    expect(refName(operation.responses['429']?.$ref)).toBe('TooManyRequests')
  })

  it('429: too_many_requests with Retry-After', () => {
    expect(ERROR_CODES).toContain('too_many_requests')
    const response = contract.components.responses.TooManyRequests
    expect(response?.headers?.['Retry-After']?.schema.maximum).toBe(60)
    expect(response?.content?.['application/json']?.example).toMatchObject({
      error: 'too_many_requests',
      message:
        'Muitas solicitações em pouco tempo. Aguarde um minuto e tente novamente.'
    })
  })
})
