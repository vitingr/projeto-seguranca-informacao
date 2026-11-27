// Ticket types, the fields of each type and the title rule (spec 001-ticket-
// builder, section 4 and BR-3). Codes are the values on the wire; labels and
// descriptions are what the user sees. The codes are checked against
// backend/api/openapi.json in index.test.ts, since `domain` cannot import
// the API contract in `lib`.

import { z } from 'zod'

import type {
  TicketField,
  TicketFieldKey,
  TicketInput,
  TicketStatus,
  TicketType
} from './types'

export const TICKET_TYPES = [
  {
    code: 'acesso',
    description: 'Peça acesso a um sistema, painel ou recurso da empresa.',
    label: 'Solicitar acesso'
  },
  {
    code: 'bug',
    description: 'Relate um erro em um sistema para a equipe investigar.',
    label: 'Suporte em bug'
  },
  {
    code: 'deploy',
    description: 'Peça a publicação de um projeto em um dos ambientes.',
    label: 'Deploy de projeto'
  },
  {
    code: 'outro',
    description: 'Qualquer outro pedido para a equipe de infraestrutura.',
    label: 'Outro'
  }
] as const

export const ACCESS_LEVELS = [
  {
    code: 'leitura',
    description: 'Consultar dados e telas, sem alterar nada.',
    label: 'Leitura'
  },
  {
    code: 'escrita',
    description: 'Criar e alterar dados no dia a dia.',
    label: 'Escrita'
  },
  {
    code: 'administrador',
    description: 'Gerenciar configurações e outros usuários.',
    label: 'Administrador'
  }
] as const

/** The help texts of spec section 4. "Crítica" is the risky option. */
export const SEVERITIES = [
  { code: 'baixa', description: 'Incômodo, há alternativa.', label: 'Baixa' },
  {
    code: 'media',
    description: 'Atrapalha o trabalho, há alternativa.',
    label: 'Média'
  },
  {
    code: 'alta',
    description: 'Bloqueia uma tarefa, sem alternativa.',
    label: 'Alta'
  },
  {
    code: 'critica',
    description: 'Sistema fora do ar ou dados em risco.',
    label: 'Crítica',
    risky: true
  }
] as const

/** "Produção" is the risky option. */
export const ENVIRONMENTS = [
  {
    code: 'dev',
    description: 'Ambiente de testes da equipe.',
    label: 'Desenvolvimento'
  },
  {
    code: 'homolog',
    description: 'Validação antes de ir para produção.',
    label: 'Homologação'
  },
  {
    code: 'prod',
    description: 'Ambiente usado por todas as pessoas.',
    label: 'Produção',
    risky: true
  }
] as const

export const TICKET_STATUS_LABELS: Record<TicketStatus, string> = {
  aberto: 'Aberto',
  em_andamento: 'Em andamento',
  fechado: 'Fechado',
  resolvido: 'Resolvido'
}

/**
 * The status changes an admin may choose from each status (spec 003 BR-3),
 * the same table as the backend's `AllowedNextStatuses`. "Fechado" is final.
 */
export const ALLOWED_NEXT_STATUSES = {
  aberto: ['em_andamento', 'fechado'],
  em_andamento: ['resolvido', 'fechado'],
  fechado: [],
  resolvido: ['em_andamento', 'fechado']
} as const satisfies Record<TicketStatus, readonly TicketStatus[]>

/** The longest message of a thread, in code points (spec 003 BR-6). */
export const MESSAGE_MAX = 2000

/** The field messages of a thread reply (spec 003 AC-3.5, 001 section 4). */
export const MESSAGE_PROBLEMS = {
  control: 'Remova os caracteres de controle do texto.',
  empty: 'Preencha este campo.',
  invalidText: 'O texto tem caracteres inválidos. Digite-o novamente.',
  tooLong: `Use no máximo ${MESSAGE_MAX} caracteres.`
} as const

/** Limits of spec section 4, in Unicode code points. */
export const FIELD_LIMITS = {
  long: { max: 2000, min: 10 },
  optional: { max: 2000, min: 0 },
  short: { max: 120, min: 2 },
  url: { max: 300, min: 1 }
} as const

/** The title limit of BR-3, the same as the server and the database. */
export const TITLE_MAX = 120

/**
 * The form of each type, in the request key order of `dados` (the screen
 * may group short fields first). `hint` is help text (under the long
 * fields, or a format rule); `placeholder` is a short example inside the
 * empty control.
 */
export const TICKET_FIELDS = {
  acesso: [
    {
      hint: 'Exemplo: Grafana, VPN, banco de dados de relatórios.',
      key: 'sistema',
      kind: 'short',
      label: 'Sistema ou recurso',
      placeholder: 'Ex.: Grafana, VPN',
      required: true
    },
    {
      key: 'nivel_acesso',
      kind: 'choice',
      label: 'Nível de acesso',
      options: ACCESS_LEVELS,
      placeholder: 'Escolha o nível',
      required: true
    },
    {
      hint: 'Conte para que você precisa do acesso e por quanto tempo.',
      key: 'justificativa',
      kind: 'long',
      label: 'Justificativa',
      placeholder: 'Ex.: Preciso consultar os painéis de uso durante o mês.',
      required: true
    }
  ],
  bug: [
    {
      hint: 'Exemplo: Portal do aluno, API de pagamentos.',
      key: 'sistema',
      kind: 'short',
      label: 'Sistema afetado',
      placeholder: 'Ex.: Portal do aluno',
      required: true
    },
    {
      hint: 'O que acontece e o que você esperava que acontecesse.',
      key: 'descricao',
      kind: 'long',
      label: 'Descrição',
      placeholder: 'Ex.: Ao salvar o formulário, a página mostra erro 500.',
      required: true
    },
    {
      hint: 'Um passo por linha, do início até o erro aparecer.',
      key: 'passos',
      kind: 'long',
      label: 'Passos para reproduzir',
      placeholder: '1. Abra o portal\n2. Clique em Salvar',
      required: true
    },
    {
      key: 'severidade',
      kind: 'choice',
      label: 'Severidade',
      options: SEVERITIES,
      placeholder: 'Escolha a severidade',
      required: true
    }
  ],
  deploy: [
    {
      hint: 'Exemplo: api-de-pagamentos.',
      key: 'projeto',
      kind: 'short',
      label: 'Nome do projeto',
      placeholder: 'Ex.: api-de-pagamentos',
      required: true
    },
    {
      hint: 'Endereço https:// do repositório, como https://github.com/org/projeto.',
      key: 'repositorio',
      kind: 'url',
      label: 'Repositório',
      placeholder: 'https://github.com/org/projeto',
      required: true
    },
    {
      key: 'ambiente',
      kind: 'choice',
      label: 'Ambiente',
      options: ENVIRONMENTS,
      placeholder: 'Escolha o ambiente',
      required: true
    },
    {
      hint: 'Janela de horário, dependências ou cuidados para a publicação.',
      key: 'observacoes',
      kind: 'optional',
      label: 'Observações',
      placeholder: 'Ex.: Publicar depois das 18h.',
      required: false
    }
  ],
  outro: [
    {
      hint: 'Um resumo curto do pedido.',
      key: 'titulo',
      kind: 'short',
      label: 'Título',
      placeholder: 'Ex.: Liberar uma porta no firewall',
      required: true
    },
    {
      hint: 'Explique o que você precisa com o máximo de detalhes.',
      key: 'descricao',
      kind: 'long',
      label: 'Descrição',
      placeholder:
        'Ex.: Preciso da porta 8443 aberta para o servidor de testes.',
      required: true
    }
  ]
} as const satisfies { [Type in TicketType]: readonly TicketField[] }

/** The label of a ticket type code. */
export const ticketTypeLabel = (type: TicketType): string =>
  TICKET_TYPES.find(option => option.code === type)?.label ?? type

/** The label of a fixed-choice code of a field, or the code itself. */
export const choiceLabel = (field: TicketField, code: string): string =>
  field.options?.find(option => option.code === code)?.label ?? code

/** The fields of a type, as a plain list (for loops over any type). */
export const fieldsOf = (type: TicketType): readonly TicketField[] =>
  TICKET_FIELDS[type]

/** The field of a type with this key. */
export const fieldOf = (
  type: TicketType,
  key: TicketFieldKey
): TicketField | undefined => fieldsOf(type).find(field => field.key === key)

/**
 * Whitespace as the backend trims it (Go's unicode.IsSpace). It differs from
 * JavaScript's `trim()`: U+0085 is whitespace here and U+FEFF is not.
 */
const SPACE =
  '\\t\\n\\v\\f\\r \\u0085\\u00a0\\u1680\\u2000-\\u200a\\u2028\\u2029\\u202f\\u205f\\u3000'

const LEADING_SPACE = new RegExp(`^[${SPACE}]+`)
const TRAILING_SPACE = new RegExp(`[${SPACE}]+$`)

export const trimEndField = (value: string): string =>
  value.replace(TRAILING_SPACE, '')

/** Trims a field value the way the backend does before validating. */
export const trimField = (value: string): string =>
  trimEndField(value.replace(LEADING_SPACE, ''))

/** Length in Unicode code points, as the backend counts it. */
export const codePoints = (value: string): number => [...value].length

/** A lone surrogate: the JavaScript form of text that is not valid UTF-8. */
const LONE_SURROGATE = /\p{Surrogate}/u

/** Go's unicode.IsControl except line feed, carriage return and tab. */
const FORBIDDEN_CONTROL = new RegExp(
  '[\\u0000-\\u0008\\u000b\\u000c\\u000e-\\u001f\\u007f-\\u009f]'
)

/**
 * The message rule of the backend's `ValidateMessage` (spec 003 BR-6, the
 * multi-line rule of 001): valid text, trimmed, 1 to 2000 code points, and
 * no control character but line feed, carriage return and tab. `null` when
 * the text is valid, otherwise the message to show under the field.
 */
export const checkMessage = (raw: string): null | string => {
  if (LONE_SURROGATE.test(raw)) return MESSAGE_PROBLEMS.invalidText
  const value = trimField(raw)
  if (value === '') return MESSAGE_PROBLEMS.empty
  if (FORBIDDEN_CONTROL.test(value)) return MESSAGE_PROBLEMS.control
  if (codePoints(value) > MESSAGE_MAX) return MESSAGE_PROBLEMS.tooLong
  return null
}

/** The reply text as zod checks it before sending; the result is trimmed. */
export const messageSchema = z
  .string()
  .superRefine((raw, ctx) => {
    const problem = checkMessage(raw)
    if (problem) ctx.addIssue({ code: 'custom', message: problem })
  })
  .transform(trimField)

/** The first `max` code points of `value`, never splitting a character. */
const cutCodePoints = (value: string, max: number): string =>
  [...value].slice(0, max).join('')

/**
 * The title the server gives a ticket (BR-3): "Outro" keeps the typed title;
 * the other types get "Acesso a <sistema>", "Bug em <sistema afetado>" or
 * "Deploy de <projeto> em <ambiente>" (environment label), cut to 120 code
 * points, with whitespace left at the end by the cut removed. Field values
 * are trimmed first, as the server does.
 */
export const ticketTitle = (ticket: TicketInput): string => {
  const title = (() => {
    switch (ticket.tipo) {
      case 'acesso':
        return `Acesso a ${trimField(ticket.dados.sistema)}`
      case 'bug':
        return `Bug em ${trimField(ticket.dados.sistema)}`
      case 'deploy': {
        const environment =
          ENVIRONMENTS.find(option => option.code === ticket.dados.ambiente)
            ?.label ?? ticket.dados.ambiente
        return `Deploy de ${trimField(ticket.dados.projeto)} em ${environment}`
      }
      case 'outro':
        return trimField(ticket.dados.titulo)
    }
  })()
  return trimEndField(cutCodePoints(title, TITLE_MAX))
}
