import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'

import {
  ACCESS_LEVELS,
  ALLOWED_NEXT_STATUSES,
  ENVIRONMENTS,
  SEVERITIES,
  TICKET_FIELDS,
  TICKET_STATUS_LABELS,
  TICKET_TYPES,
  checkMessage,
  codePoints,
  messageSchema,
  ticketTitle,
  trimField
} from '.'
import type { TicketInput } from './types'
import type { TicketStatus } from './types'

// Invisible characters are built from their code points, so the source
// shows which one each test uses.
const NBSP = String.fromCodePoint(0xa0)
const NEXT_LINE = String.fromCodePoint(0x85)
const LINE_SEPARATOR = String.fromCodePoint(0x2028)
const IDEOGRAPHIC_SPACE = String.fromCodePoint(0x3000)
const BOM = String.fromCodePoint(0xfeff)

const contract = JSON.parse(
  readFileSync(
    new URL('../../../../backend/api/openapi.json', import.meta.url),
    'utf8'
  )
) as {
  components: {
    schemas: Record<
      string,
      { enum?: string[]; properties?: Record<string, unknown> }
    >
  }
}

const enumOf = (name: string) => contract.components.schemas[name]?.enum
const codes = (options: readonly { code: string }[]) =>
  options.map(option => option.code)

describe('codes match backend/api/openapi.json', () => {
  it.each([
    ['TicketType', codes(TICKET_TYPES)],
    ['AccessLevel', codes(ACCESS_LEVELS)],
    ['Severity', codes(SEVERITIES)],
    ['Environment', codes(ENVIRONMENTS)],
    ['TicketStatus', Object.keys(TICKET_STATUS_LABELS).sort()]
  ])('%s', (name, want) => {
    const got = enumOf(name) ?? []
    expect(name === 'TicketStatus' ? [...got].sort() : got).toEqual(want)
  })

  it.each([
    ['acesso', 'AccessTicketData'],
    ['bug', 'BugTicketData'],
    ['deploy', 'DeployTicketData'],
    ['outro', 'OtherTicketData']
  ] as const)('the fields of %s are the keys of %s', (type, schema) => {
    const keys = Object.keys(
      contract.components.schemas[schema]?.properties ?? {}
    )
    expect(TICKET_FIELDS[type].map(field => field.key)).toEqual(keys)
  })
})

describe('labels and help texts (spec section 4)', () => {
  it('types', () => {
    expect(TICKET_TYPES.map(option => option.label)).toEqual([
      'Solicitar acesso',
      'Suporte em bug',
      'Deploy de projeto',
      'Outro'
    ])
    for (const option of TICKET_TYPES) {
      expect(option.description).not.toBe('')
    }
  })

  it('choices', () => {
    expect(ACCESS_LEVELS.map(option => option.label)).toEqual([
      'Leitura',
      'Escrita',
      'Administrador'
    ])
    expect(ENVIRONMENTS.map(option => option.label)).toEqual([
      'Desenvolvimento',
      'Homologação',
      'Produção'
    ])
    expect(
      SEVERITIES.map(option => `${option.label}: ${option.description}`)
    ).toEqual([
      'Baixa: Incômodo, há alternativa.',
      'Média: Atrapalha o trabalho, há alternativa.',
      'Alta: Bloqueia uma tarefa, sem alternativa.',
      'Crítica: Sistema fora do ar ou dados em risco.'
    ])
  })

  it('only "Produção" and "Crítica" are risky', () => {
    const risky = [...ACCESS_LEVELS, ...SEVERITIES, ...ENVIRONMENTS]
      .filter(option => 'risky' in option && option.risky)
      .map(option => option.label)
    expect(risky).toEqual(['Crítica', 'Produção'])
  })

  it('only the deploy notes are optional', () => {
    const optional = Object.values(TICKET_FIELDS)
      .flat()
      .filter(field => !field.required)
      .map(field => field.key)
    expect(optional).toEqual(['observacoes'])
  })
})

describe('trimField and codePoints', () => {
  it.each([
    ['  Grafana \t', 'Grafana'],
    ['\n\r\v\f Grafana', 'Grafana'],
    [`${NBSP}Grafana${IDEOGRAPHIC_SPACE}`, 'Grafana'],
    [`${NEXT_LINE}Grafana${LINE_SEPARATOR}`, 'Grafana'],
    // U+FEFF is not whitespace for the backend (Go's unicode.IsSpace).
    [`${BOM}Grafana`, `${BOM}Grafana`],
    ['Portal  do aluno', 'Portal  do aluno']
  ])('trimField(%j) → %j', (raw, want) => {
    expect(trimField(raw)).toBe(want)
  })

  it('counts code points', () => {
    expect(codePoints('é😀a')).toBe(3)
    expect(codePoints('')).toBe(0)
  })
})

// The BR-3 vectors of backend/internal/service/tickets_test.go
// (titleVectors), so both sides build the same title.
const rep = (value: string, n: number) => value.repeat(n)

const access = (sistema: string): TicketInput => ({
  dados: {
    justificativa: 'Preciso ver os painéis.',
    nivel_acesso: 'leitura',
    sistema
  },
  tipo: 'acesso'
})
const bug = (sistema: string): TicketInput => ({
  dados: {
    descricao: 'A página não carrega.',
    passos: '1. Abrir\n2. Entrar',
    severidade: 'alta',
    sistema
  },
  tipo: 'bug'
})
const deploy = (
  projeto: string,
  ambiente: 'dev' | 'homolog' | 'prod'
): TicketInput => ({
  dados: { ambiente, projeto, repositorio: 'https://git.example/api' },
  tipo: 'deploy'
})
const other = (titulo: string): TicketInput => ({
  dados: { descricao: 'Como peço uma VPN?', titulo },
  tipo: 'outro'
})

const titleVectors: [string, TicketInput, string][] = [
  ['acesso', access('Grafana'), 'Acesso a Grafana'],
  ['acesso, system trimmed', access('  Grafana \t'), 'Acesso a Grafana'],
  ['bug', bug('Portal do aluno'), 'Bug em Portal do aluno'],
  ['deploy to dev', deploy('api', 'dev'), 'Deploy de api em Desenvolvimento'],
  [
    'deploy to homolog',
    deploy('api', 'homolog'),
    'Deploy de api em Homologação'
  ],
  ['deploy to prod', deploy('api', 'prod'), 'Deploy de api em Produção'],
  [
    'outro keeps the typed title',
    other('  Dúvida sobre a VPN '),
    'Dúvida sobre a VPN'
  ],
  [
    'outro with HTML is kept as typed',
    other('<b>Ajuda</b> & **x**'),
    '<b>Ajuda</b> & **x**'
  ],
  ['exactly 120 is not cut', bug(rep('é', 113)), `Bug em ${rep('é', 113)}`],
  ['acesso cut to 120', access(rep('é', 120)), `Acesso a ${rep('é', 111)}`],
  [
    'emoji at position 120 is kept whole',
    access(`${rep('a', 110)}😀zz`),
    `Acesso a ${rep('a', 110)}😀`
  ],
  [
    'emoji at position 121 is dropped whole',
    access(`${rep('a', 111)}😀`),
    `Acesso a ${rep('a', 111)}`
  ],
  [
    'deploy cut inside the environment label',
    deploy(rep('p', 105), 'prod'),
    `Deploy de ${rep('p', 105)} em P`
  ],
  [
    'space at the cut is removed',
    access(`${rep('a', 110)} bcd`),
    `Acesso a ${rep('a', 110)}`
  ],
  [
    'several spaces before the cut are removed',
    bug(`${rep('b', 108)}     xyz`),
    `Bug em ${rep('b', 108)}`
  ],
  [
    'no-break space at the cut is removed',
    access(`${rep('a', 110)}${NBSP}yy`),
    `Acesso a ${rep('a', 110)}`
  ],
  [
    'deploy cut right after "em " drops the space',
    deploy(rep('p', 106), 'prod'),
    `Deploy de ${rep('p', 106)} em`
  ],
  [
    'inner spaces are kept',
    other('Dúvida   sobre  a VPN'),
    'Dúvida   sobre  a VPN'
  ],
  [
    'deploy with a 120 project keeps only the project',
    deploy(rep('p', 120), 'dev'),
    `Deploy de ${rep('p', 110)}`
  ]
]

describe('ticketTitle (BR-3)', () => {
  it.each(titleVectors)('%s', (_name, input, want) => {
    const title = ticketTitle(input)
    expect(title).toBe(want)
    expect(codePoints(title)).toBeLessThanOrEqual(120)
    expect(title.isWellFormed()).toBe(true)
  })
})

// Copied from the Go table of backend/internal/security/ticket_status_test.go
// (spec 003 BR-3): every from × to pair and whether it is allowed.
const transitionVectors: [TicketStatus, TicketStatus, boolean][] = [
  ['aberto', 'aberto', false],
  ['aberto', 'em_andamento', true],
  ['aberto', 'resolvido', false],
  ['aberto', 'fechado', true],
  ['em_andamento', 'aberto', false],
  ['em_andamento', 'em_andamento', false],
  ['em_andamento', 'resolvido', true],
  ['em_andamento', 'fechado', true],
  ['resolvido', 'aberto', false],
  ['resolvido', 'em_andamento', true],
  ['resolvido', 'resolvido', false],
  ['resolvido', 'fechado', true],
  ['fechado', 'aberto', false],
  ['fechado', 'em_andamento', false],
  ['fechado', 'resolvido', false],
  ['fechado', 'fechado', false]
]

describe('ALLOWED_NEXT_STATUSES (BR-3)', () => {
  it.each(transitionVectors)('%s → %s: %s', (from, to, allowed) => {
    const next: readonly TicketStatus[] = ALLOWED_NEXT_STATUSES[from]
    expect(next.includes(to)).toBe(allowed)
  })

  it('is the table of the contract', () => {
    const request = contract.components.schemas.ChangeStatusRequest as {
      'x-allowed-transitions'?: Record<string, string[]>
    }
    const documented = request['x-allowed-transitions'] ?? {}
    for (const [from, next] of Object.entries(ALLOWED_NEXT_STATUSES)) {
      expect([...next].sort()).toEqual([...(documented[from] ?? [])].sort())
    }
  })
})

describe('checkMessage and messageSchema (spec 003 BR-6)', () => {
  it.each([
    ['one character', 'a'],
    ['2000 characters', 'a'.repeat(2000)],
    ['2000 multi-byte characters', '😀'.repeat(2000)],
    ['line breaks and tabs', 'linha 1\n\tlinha 2\r\n'],
    ['a line separator inside', `a${LINE_SEPARATOR}b`]
  ])('accepts %s', (_name, text) => {
    expect(checkMessage(text)).toBeNull()
  })

  it.each([
    ['empty', '', 'Preencha este campo.'],
    ['spaces only', ` \n\t${NBSP}`, 'Preencha este campo.'],
    ['2001 characters', 'a'.repeat(2001), 'Use no máximo 2000 caracteres.'],
    ['a NUL', 'a\u0000b', 'Remova os caracteres de controle do texto.']
  ])('refuses %s', (_name, text, message) => {
    expect(checkMessage(text)).toBe(message)
    const result = messageSchema.safeParse(text)
    expect(result.success).toBe(false)
    expect(result.error?.issues[0]?.message).toBe(message)
  })

  it('trims what it accepts, as the backend stores it', () => {
    expect(messageSchema.parse('  olá\n')).toBe('olá')
    expect(codePoints(`  ${'a'.repeat(2000)}  `)).toBe(2004)
    expect(checkMessage(`  ${'a'.repeat(2000)}  `)).toBeNull()
  })
})
