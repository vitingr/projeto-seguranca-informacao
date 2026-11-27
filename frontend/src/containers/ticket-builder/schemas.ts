import { z } from 'zod'

import {
  ACCESS_LEVELS,
  ENVIRONMENTS,
  FIELD_LIMITS,
  SEVERITIES,
  codePoints,
  trimField
} from '@/domain/tickets'
import type { TicketInput, TicketType } from '@/domain/tickets/types'

import type {
  FieldProblem,
  TextKind,
  TicketFormValues,
  TicketValuesByType
} from './types'

export const FIELD_MESSAGES = {
  choice: 'Escolha uma opção.',
  controlMultiLine: 'Remova os caracteres de controle do texto.',
  controlSingleLine: 'Remova tabulações e outros caracteres de controle.',
  credentials: 'Não inclua usuário ou senha no endereço.',
  empty: 'Preencha este campo.',
  invalidText: 'O texto tem caracteres inválidos. Digite-o novamente.',
  lineBreak: 'Use uma única linha, sem quebras de linha.',
  notHttpsUrl:
    'Informe um endereço https:// completo, como https://github.com/org/projeto.'
} as const

export const tooShortMessage = (min: number) =>
  `Use pelo menos ${min} caracteres.`

export const tooLongMessage = (max: number) =>
  `Use no máximo ${max} caracteres.`

const LONE_SURROGATE = /\p{Surrogate}/u

const CONTROL = new RegExp('[\\u0000-\\u001f\\u007f-\\u009f]')

const LINE_BREAK = new RegExp('[\\n\\r\\u2028\\u2029]')

const HAS_SPACE = new RegExp(
  '[\\t\\n\\v\\f\\r \\u0085\\u00a0\\u1680\\u2000-\\u200a\\u2028\\u2029\\u202f\\u205f\\u3000]'
)

const checkText = (
  raw: string,
  { max, min }: { max: number; min: number },
  checkChar: (char: string) => FieldProblem | null
): FieldProblem | null => {
  if (LONE_SURROGATE.test(raw)) return 'invalidText'
  const value = trimField(raw)
  if (value === '') return min > 0 ? 'empty' : null
  for (const char of value) {
    const problem = checkChar(char)
    if (problem) return problem
  }
  const length = codePoints(value)
  if (length < min) return 'tooShort'
  if (length > max) return 'tooLong'
  return null
}

export const checkSingleLine = (
  raw: string,
  limits: { max: number; min: number }
): FieldProblem | null =>
  checkText(raw, limits, char => {
    if (LINE_BREAK.test(char)) return 'lineBreak'
    if (CONTROL.test(char)) return 'controlSingleLine'
    return null
  })

export const checkMultiLine = (
  raw: string,
  limits: { max: number; min: number }
): FieldProblem | null =>
  checkText(raw, limits, char =>
    CONTROL.test(char) && !/[\n\r\t]/.test(char) ? 'controlMultiLine' : null
  )

const INVALID_ESCAPE = /%(?![0-9a-fA-F]{2})/

const INVALID_HOST_CHAR = /[^A-Za-z0-9\-_.~!$&'()*+,;=:[\]<>"%]/

const checkHost = (hostPort: string): FieldProblem | null => {
  let host = hostPort
  let port = ''
  if (hostPort.startsWith('[')) {
    const end = hostPort.indexOf(']')
    if (end < 0) return 'notHttpsUrl'
    host = hostPort.slice(1, end)
    const rest = hostPort.slice(end + 1)
    if (rest !== '' && !rest.startsWith(':')) return 'notHttpsUrl'
    port = rest.slice(1)
  } else {
    const colon = hostPort.lastIndexOf(':')
    if (colon >= 0) {
      host = hostPort.slice(0, colon)
      port = hostPort.slice(colon + 1)
    }
  }
  if (!/^\d*$/.test(port)) return 'notHttpsUrl'
  if (host === '') return 'notHttpsUrl'
  const ascii = [...host].filter(char => char.charCodeAt(0) < 0x80).join('')
  if (INVALID_HOST_CHAR.test(ascii)) return 'notHttpsUrl'
  if (INVALID_ESCAPE.test(host)) return 'notHttpsUrl'
  return null
}

// Follows Go's url.Parse, not the browser's URL parser, which accepts forms the backend refuses.
export const checkHttpsUrl = (raw: string): FieldProblem | null => {
  const problem = checkSingleLine(raw, FIELD_LIMITS.url)
  if (problem) return problem
  const value = trimField(raw)
  if (HAS_SPACE.test(value)) return 'notHttpsUrl'

  const scheme = /^([A-Za-z][A-Za-z0-9+.-]*):/.exec(value)
  if (!scheme || scheme[1].toLowerCase() !== 'https') return 'notHttpsUrl'

  const hash = value.indexOf('#')
  const fragment = hash >= 0 ? value.slice(hash + 1) : ''
  const beforeFragment = hash >= 0 ? value.slice(0, hash) : value
  const rest = beforeFragment.slice(scheme[0].length).split('?')[0]
  if (!rest.startsWith('//')) return 'notHttpsUrl'

  const afterSlashes = rest.slice(2)
  const slash = afterSlashes.indexOf('/')
  const authority = slash >= 0 ? afterSlashes.slice(0, slash) : afterSlashes
  const path = slash >= 0 ? afterSlashes.slice(slash) : ''
  if (INVALID_ESCAPE.test(path) || INVALID_ESCAPE.test(fragment)) {
    return 'notHttpsUrl'
  }

  const at = authority.lastIndexOf('@')
  const hostProblem = checkHost(authority.slice(at + 1))
  if (hostProblem) return hostProblem
  return at >= 0 ? 'credentials' : null
}

export const messageForProblem = (
  problem: FieldProblem,
  { max, min }: { max: number; min: number }
): string => {
  switch (problem) {
    case 'tooShort':
      return tooShortMessage(min)
    case 'tooLong':
      return tooLongMessage(max)
    default:
      return FIELD_MESSAGES[problem]
  }
}

const textField = (kind: TextKind) => {
  const limits = FIELD_LIMITS[kind]
  const check = (raw: string) => {
    switch (kind) {
      case 'url':
        return checkHttpsUrl(raw)
      case 'short':
        return checkSingleLine(raw, limits)
      default:
        return checkMultiLine(raw, limits)
    }
  }
  return z
    .string()
    .superRefine((raw, ctx) => {
      const problem = check(raw)
      if (problem) {
        ctx.addIssue({
          code: 'custom',
          message: messageForProblem(problem, limits)
        })
      }
    })
    .transform(trimField)
}

const choiceField = <const Code extends string>(
  options: readonly { code: Code }[]
) => {
  const codes: readonly string[] = options.map(option => option.code)
  return z
    .string({ error: FIELD_MESSAGES.choice })
    .refine(value => codes.includes(value), { message: FIELD_MESSAGES.choice })
    .transform(value => value as Code)
}

export const ticketSchemas = {
  acesso: z.object({
    justificativa: textField('long'),
    nivel_acesso: choiceField(ACCESS_LEVELS),
    sistema: textField('short')
  }),
  bug: z.object({
    descricao: textField('long'),
    passos: textField('long'),
    severidade: choiceField(SEVERITIES),
    sistema: textField('short')
  }),
  deploy: z.object({
    ambiente: choiceField(ENVIRONMENTS),
    observacoes: textField('optional'),
    projeto: textField('short'),
    repositorio: textField('url')
  }),
  outro: z.object({
    descricao: textField('long'),
    titulo: textField('short')
  })
}

export const emptyValues = (type: TicketType): TicketFormValues =>
  Object.fromEntries(
    Object.keys(ticketSchemas[type].shape).map(key => [key, ''])
  )

export const toTicketInput = <Type extends TicketType>(
  type: Type,
  values: TicketValuesByType[Type]
): TicketInput => {
  if (type === 'deploy') {
    const { observacoes, ...rest } = values as TicketValuesByType['deploy']
    return {
      dados: observacoes === '' ? rest : { ...rest, observacoes },
      tipo: 'deploy'
    }
  }
  return { dados: values, tipo: type } as TicketInput
}

export const parseTicket = (
  type: TicketType,
  values: TicketFormValues
): TicketInput | null => {
  const result = ticketSchemas[type].safeParse(values)
  return result.success
    ? toTicketInput(type, result.data as TicketValuesByType[typeof type])
    : null
}
