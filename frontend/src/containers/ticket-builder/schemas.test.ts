import { describe, expect, it } from 'vitest'

import {
  FIELD_MESSAGES,
  checkHttpsUrl,
  checkMultiLine,
  checkSingleLine,
  emptyValues,
  parseTicket,
  ticketSchemas
} from './schemas'
import type { FieldProblem, TicketFormValues } from './types'

// The boundary vectors of backend/internal/security/tickets_test.go (task
// T3.1: TestSingleLine, TestMultiLine, TestMultiLineOptional, TestHTTPSURL;
// T3.2: TestValidateTicketAccepts/Refuses), so the form accepts and refuses
// what the backend does (S-2). Text that is not valid UTF-8 cannot exist in
// a JavaScript string; its counterpart is a lone surrogate.

const rep = (value: string, n: number) => value.repeat(n)
// Invisible characters are built from their code points, so the source
// shows which one each test uses.
const BAD_TEXT = String.fromCharCode(0xd800)
const NBSP = String.fromCodePoint(0xa0)
const LINE_SEPARATOR = String.fromCodePoint(0x2028)
const PARAGRAPH_SEPARATOR = String.fromCodePoint(0x2029)

type Vector = [name: string, raw: string, problem: FieldProblem | null]

describe('checkSingleLine (2–120)', () => {
  it.each<Vector>([
    ['plain', 'Grafana', null],
    ['trimmed', '  \t Grafana \n ', null],
    ['inner spaces kept', 'Portal  do aluno', null],
    ['html kept as typed', '<b>x</b>', null],
    ['empty', '', 'empty'],
    ['spaces only', ' \t\n\r ', 'empty'],
    ['min - 1', 'a', 'tooShort'],
    ['min', 'ab', null],
    ['min after trim - 1', '  a  ', 'tooShort'],
    ['max', rep('a', 120), null],
    ['max + 1', rep('a', 121), 'tooLong'],
    ['multi-byte at the max', `${rep('é', 119)}😀`, null],
    ['multi-byte past the max', `${rep('é', 120)}😀`, 'tooLong'],
    ['line feed', 'Gra\nfana', 'lineBreak'],
    ['carriage return', 'Gra\rfana', 'lineBreak'],
    ['line separator', `Gra${LINE_SEPARATOR}fana`, 'lineBreak'],
    ['paragraph separator', `Gra${PARAGRAPH_SEPARATOR}fana`, 'lineBreak'],
    ['tab', 'Gra\tfana', 'controlSingleLine'],
    ['NUL', 'Gra\u0000fana', 'controlSingleLine'],
    ['escape', 'Gra\u001bfana', 'controlSingleLine'],
    ['DEL', 'Gra\u007ffana', 'controlSingleLine'],
    ['C1 control', 'Gra\u0085fana', 'controlSingleLine'],
    ['invalid text', `Gra${BAD_TEXT}fana`, 'invalidText'],
    ['invalid text at the end', `Grafana${BAD_TEXT}`, 'invalidText']
  ])('%s', (_name, raw, problem) => {
    expect(checkSingleLine(raw, { max: 120, min: 2 })).toBe(problem)
  })
})

describe('checkMultiLine (10–2000)', () => {
  it.each<Vector>([
    ['plain', 'Preciso ver os painéis.', null],
    ['line breaks and tabs kept', '1. Abrir\r\n2. Entrar\n\tdepois', null],
    ['trimmed', '\n\n  Preciso ver os painéis.  \n', null],
    ['markdown kept as typed', '**negrito** e _x_', null],
    ['empty', '', 'empty'],
    ['spaces and breaks only', ' \n\t\r\n ', 'empty'],
    ['min - 1', rep('a', 9), 'tooShort'],
    ['min', rep('a', 10), null],
    ['max', rep('a', 2000), null],
    ['max + 1', rep('a', 2001), 'tooLong'],
    ['multi-byte at the max', `${rep('ç', 1999)}😀`, null],
    ['multi-byte past the max', `${rep('ç', 2000)}😀`, 'tooLong'],
    ['line separator allowed', `linha um${LINE_SEPARATOR}linha dois`, null],
    ['NUL', 'Preciso ver\u0000 os painéis', 'controlMultiLine'],
    ['escape', 'Preciso ver\u001b[31m os painéis', 'controlMultiLine'],
    ['vertical tab', 'Preciso ver\v os painéis', 'controlMultiLine'],
    ['form feed', 'Preciso ver\f os painéis', 'controlMultiLine'],
    ['DEL', 'Preciso ver\u007f os painéis', 'controlMultiLine'],
    ['C1 control', 'Preciso ver\u0085 os painéis', 'controlMultiLine'],
    ['invalid text', `Preciso ver ${BAD_TEXT} os painéis`, 'invalidText']
  ])('%s', (_name, raw, problem) => {
    expect(checkMultiLine(raw, { max: 2000, min: 10 })).toBe(problem)
  })
})

describe('checkMultiLine, optional (0–2000)', () => {
  it.each<Vector>([
    ['empty is allowed', '', null],
    ['spaces only become empty', '  \n ', null],
    ['one character', 'x', null],
    ['max', rep('a', 2000), null],
    ['max + 1', rep('a', 2001), 'tooLong'],
    ['NUL still refused', 'a\u0000b', 'controlMultiLine']
  ])('%s', (_name, raw, problem) => {
    expect(checkMultiLine(raw, { max: 2000, min: 0 })).toBe(problem)
  })
})

describe('checkHttpsUrl (up to 300)', () => {
  const base = 'https://git.example.com/'
  it.each<Vector>([
    ['repository', 'https://github.com/org/repo', null],
    [
      'with .git, query and fragment',
      'https://git.example.com/a/b.git?ref=main#readme',
      null
    ],
    ['with port', 'https://git.example.com:8443/a', null],
    ['ip host', 'https://10.0.0.1/repo', null],
    ['ipv6 host', 'https://[::1]:8443/repo', null],
    ['upper case scheme', 'HTTPS://git.example.com/a', null],
    ['trimmed', '  https://github.com/org/repo \n', null],
    ['max', base + rep('a', 300 - base.length), null],
    ['max + 1', base + rep('a', 301 - base.length), 'tooLong'],
    ['multi-byte at the max', base + rep('é', 300 - base.length), null],
    ['empty', '', 'empty'],
    ['spaces only', '   ', 'empty'],
    ['http', 'http://github.com/org/repo', 'notHttpsUrl'],
    ['javascript', 'javascript:alert(1)', 'notHttpsUrl'],
    ['data', 'data:text/html,<script>alert(1)</script>', 'notHttpsUrl'],
    ['ssh scp form', 'git@github.com:org/repo.git', 'notHttpsUrl'],
    ['ssh url', 'ssh://git@github.com/org/repo.git', 'notHttpsUrl'],
    ['no scheme', 'github.com/org/repo', 'notHttpsUrl'],
    ['scheme relative', '//github.com/org/repo', 'notHttpsUrl'],
    ['no host', 'https://', 'notHttpsUrl'],
    ['no host with path', 'https:///org/repo', 'notHttpsUrl'],
    ['opaque', 'https:github.com/org/repo', 'notHttpsUrl'],
    ['one slash', 'https:/github.com/org/repo', 'notHttpsUrl'],
    ['port only', 'https://:443/repo', 'notHttpsUrl'],
    [
      'user and password',
      'https://user:pass@github.com/org/repo',
      'credentials'
    ],
    ['user only', 'https://token@github.com/org/repo', 'credentials'],
    ['inner space', 'https://github.com/org/my repo', 'notHttpsUrl'],
    [
      'no-break space inside',
      `https://github.com/org/my${NBSP}repo`,
      'notHttpsUrl'
    ],
    ['bad port', 'https://github.com:abc/org', 'notHttpsUrl'],
    ['bad host character', 'https://git{hub}.com/org', 'notHttpsUrl'],
    ['bad escape in the path', 'https://github.com/org/%zz', 'notHttpsUrl'],
    ['line feed', 'https://github.com/org\n/repo', 'lineBreak'],
    ['NUL', 'https://github.com/\u0000repo', 'controlSingleLine'],
    ['invalid text', `https://github.com/${BAD_TEXT}repo`, 'invalidText']
  ])('%s', (_name, raw, problem) => {
    expect(checkHttpsUrl(raw)).toBe(problem)
  })
})

/** The message under each field, or `{}` when the form is valid. */
const fieldErrors = (
  type: keyof typeof ticketSchemas,
  values: TicketFormValues
) => {
  const result = ticketSchemas[type].safeParse(values)
  if (result.success) return {}
  return Object.fromEntries(
    result.error.issues.map(issue => [String(issue.path[0]), issue.message])
  )
}

const valid = {
  acesso: {
    justificativa: '\nPreciso ver os painéis.\n',
    nivel_acesso: 'leitura',
    sistema: ' Grafana '
  },
  bug: {
    descricao: 'A página não carrega.',
    passos: '1. Abrir\n2. Entrar',
    severidade: 'critica',
    sistema: 'Portal'
  },
  deploy: {
    ambiente: 'prod',
    observacoes: '',
    projeto: 'api',
    repositorio: ' https://git.example.com/org/api '
  },
  outro: { descricao: 'Como peço uma VPN?', titulo: '  Dúvida ' }
} satisfies Record<keyof typeof ticketSchemas, TicketFormValues>

describe('ticket schemas accept (T3.2)', () => {
  it.each([
    [
      'acesso',
      valid.acesso,
      {
        dados: {
          justificativa: 'Preciso ver os painéis.',
          nivel_acesso: 'leitura',
          sistema: 'Grafana'
        },
        tipo: 'acesso'
      }
    ],
    [
      'bug',
      valid.bug,
      {
        dados: {
          descricao: 'A página não carrega.',
          passos: '1. Abrir\n2. Entrar',
          severidade: 'critica',
          sistema: 'Portal'
        },
        tipo: 'bug'
      }
    ],
    [
      'deploy without notes',
      valid.deploy,
      {
        dados: {
          ambiente: 'prod',
          projeto: 'api',
          repositorio: 'https://git.example.com/org/api'
        },
        tipo: 'deploy'
      }
    ],
    [
      'deploy with blank notes',
      { ...valid.deploy, observacoes: ' \n ' },
      {
        dados: {
          ambiente: 'prod',
          projeto: 'api',
          repositorio: 'https://git.example.com/org/api'
        },
        tipo: 'deploy'
      }
    ],
    [
      'deploy with notes',
      { ...valid.deploy, observacoes: ' Depois das 18h. ' },
      {
        dados: {
          ambiente: 'prod',
          observacoes: 'Depois das 18h.',
          projeto: 'api',
          repositorio: 'https://git.example.com/org/api'
        },
        tipo: 'deploy'
      }
    ],
    [
      'outro',
      valid.outro,
      {
        dados: { descricao: 'Como peço uma VPN?', titulo: 'Dúvida' },
        tipo: 'outro'
      }
    ]
  ] as const)('%s', (name, values, want) => {
    const type = name.split(' ')[0] as keyof typeof ticketSchemas
    expect(parseTicket(type, values)).toEqual(want)
  })
})

describe('ticket schemas refuse (T3.2)', () => {
  const empty = FIELD_MESSAGES.empty
  it.each<
    [keyof typeof ticketSchemas, TicketFormValues, Record<string, string>]
  >([
    ['acesso', { ...valid.acesso, sistema: '' }, { sistema: empty }],
    ['acesso', { ...valid.acesso, sistema: '   ' }, { sistema: empty }],
    [
      'acesso',
      { ...valid.acesso, sistema: 'é' },
      { sistema: 'Use pelo menos 2 caracteres.' }
    ],
    [
      'acesso',
      { ...valid.acesso, sistema: rep('a', 121) },
      { sistema: 'Use no máximo 120 caracteres.' }
    ],
    [
      'acesso',
      { ...valid.acesso, sistema: 'Gra\nfana' },
      { sistema: FIELD_MESSAGES.lineBreak }
    ],
    [
      'acesso',
      { ...valid.acesso, nivel_acesso: '' },
      { nivel_acesso: FIELD_MESSAGES.choice }
    ],
    [
      'acesso',
      { ...valid.acesso, nivel_acesso: 'Leitura' },
      { nivel_acesso: FIELD_MESSAGES.choice }
    ],
    [
      'acesso',
      { ...valid.acesso, nivel_acesso: ' leitura' },
      { nivel_acesso: FIELD_MESSAGES.choice }
    ],
    [
      'acesso',
      { ...valid.acesso, nivel_acesso: 'root' },
      { nivel_acesso: FIELD_MESSAGES.choice }
    ],
    [
      'acesso',
      { ...valid.acesso, justificativa: '' },
      { justificativa: empty }
    ],
    [
      'acesso',
      { ...valid.acesso, justificativa: rep('é', 9) },
      { justificativa: 'Use pelo menos 10 caracteres.' }
    ],
    [
      'acesso',
      { ...valid.acesso, justificativa: rep('a', 2001) },
      { justificativa: 'Use no máximo 2000 caracteres.' }
    ],
    [
      'acesso',
      { ...valid.acesso, justificativa: 'Preciso\u0000 ver tudo' },
      { justificativa: FIELD_MESSAGES.controlMultiLine }
    ],
    ['bug', { ...valid.bug, sistema: '' }, { sistema: empty }],
    [
      'bug',
      { ...valid.bug, descricao: 'curta' },
      { descricao: 'Use pelo menos 10 caracteres.' }
    ],
    [
      'bug',
      { ...valid.bug, descricao: `Não ${BAD_TEXT} carrega nada` },
      { descricao: FIELD_MESSAGES.invalidText }
    ],
    ['bug', { ...valid.bug, passos: ' \n ' }, { passos: empty }],
    [
      'bug',
      { ...valid.bug, passos: rep('ç', 2001) },
      { passos: 'Use no máximo 2000 caracteres.' }
    ],
    [
      'bug',
      { ...valid.bug, passos: '1. Abrir\u001b[2J agora' },
      { passos: FIELD_MESSAGES.controlMultiLine }
    ],
    [
      'bug',
      { ...valid.bug, severidade: '' },
      { severidade: FIELD_MESSAGES.choice }
    ],
    [
      'bug',
      { ...valid.bug, severidade: 'média' },
      { severidade: FIELD_MESSAGES.choice }
    ],
    [
      'bug',
      { ...valid.bug, severidade: 'Alta' },
      { severidade: FIELD_MESSAGES.choice }
    ],
    ['deploy', { ...valid.deploy, projeto: '' }, { projeto: empty }],
    [
      'deploy',
      { ...valid.deploy, projeto: 'api\tnova' },
      { projeto: FIELD_MESSAGES.controlSingleLine }
    ],
    [
      'deploy',
      { ...valid.deploy, projeto: rep('a', 121) },
      { projeto: 'Use no máximo 120 caracteres.' }
    ],
    ['deploy', { ...valid.deploy, repositorio: '' }, { repositorio: empty }],
    [
      'deploy',
      { ...valid.deploy, repositorio: 'http://git.example.com/x' },
      { repositorio: FIELD_MESSAGES.notHttpsUrl }
    ],
    [
      'deploy',
      { ...valid.deploy, repositorio: "javascript:alert('x')" },
      { repositorio: FIELD_MESSAGES.notHttpsUrl }
    ],
    [
      'deploy',
      { ...valid.deploy, repositorio: 'git@github.com:org/x.git' },
      { repositorio: FIELD_MESSAGES.notHttpsUrl }
    ],
    [
      'deploy',
      { ...valid.deploy, repositorio: 'https:///x' },
      { repositorio: FIELD_MESSAGES.notHttpsUrl }
    ],
    [
      'deploy',
      { ...valid.deploy, repositorio: 'https://user:x@git.example.com/x' },
      { repositorio: FIELD_MESSAGES.credentials }
    ],
    [
      'deploy',
      {
        ...valid.deploy,
        repositorio: `https://git.example.com/${rep('a', 277)}`
      },
      { repositorio: 'Use no máximo 300 caracteres.' }
    ],
    [
      'deploy',
      { ...valid.deploy, ambiente: '' },
      { ambiente: FIELD_MESSAGES.choice }
    ],
    [
      'deploy',
      { ...valid.deploy, ambiente: 'Produção' },
      { ambiente: FIELD_MESSAGES.choice }
    ],
    [
      'deploy',
      { ...valid.deploy, observacoes: rep('a', 2001) },
      { observacoes: 'Use no máximo 2000 caracteres.' }
    ],
    [
      'deploy',
      { ...valid.deploy, observacoes: 'Depois\u0000 das 18h' },
      { observacoes: FIELD_MESSAGES.controlMultiLine }
    ],
    ['outro', { ...valid.outro, titulo: '' }, { titulo: empty }],
    [
      'outro',
      { ...valid.outro, titulo: ' a ' },
      { titulo: 'Use pelo menos 2 caracteres.' }
    ],
    [
      'outro',
      { ...valid.outro, titulo: rep('é', 121) },
      { titulo: 'Use no máximo 120 caracteres.' }
    ],
    [
      'outro',
      { ...valid.outro, titulo: 'Dúvida\r\nnova' },
      { titulo: FIELD_MESSAGES.lineBreak }
    ],
    ['outro', { ...valid.outro, descricao: '' }, { descricao: empty }],
    [
      'outro',
      { ...valid.outro, descricao: rep('a', 2001) },
      { descricao: 'Use no máximo 2000 caracteres.' }
    ]
  ])('%s: %j', (type, values, want) => {
    expect(fieldErrors(type, values)).toEqual(want)
    expect(parseTicket(type, values)).toBeNull()
  })

  it('an empty form shows the message of every required field', () => {
    expect(fieldErrors('bug', emptyValues('bug'))).toEqual({
      descricao: empty,
      passos: empty,
      severidade: FIELD_MESSAGES.choice,
      sistema: empty
    })
  })
})

describe('emptyValues', () => {
  it('has every field of the type, empty', () => {
    expect(emptyValues('deploy')).toEqual({
      ambiente: '',
      observacoes: '',
      projeto: '',
      repositorio: ''
    })
  })
})
