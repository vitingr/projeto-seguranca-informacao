import { ClipboardCheckIcon, FileTextIcon, ListChecksIcon } from 'lucide-react'

import type { TicketType } from '@/domain/tickets/types'

import type { Step } from './types'

export const STEPS = [
  {
    description: 'Escolha do pedido',
    icon: ListChecksIcon,
    id: 'tipo',
    label: 'Tipo'
  },
  {
    description: 'Campos do tipo',
    icon: FileTextIcon,
    id: 'detalhes',
    label: 'Detalhes'
  },
  {
    description: 'Conferir e enviar',
    icon: ClipboardCheckIcon,
    id: 'revisao',
    label: 'Revisão'
  }
] as const satisfies readonly {
  description: string
  icon: typeof ListChecksIcon
  id: Step
  label: string
}[]

export const BUILDER_MESSAGES = {
  created: (id: number) => `Ticket #${id} criado.`,
  noType: 'Escolha um tipo de ticket.',
  sending: 'Enviando ticket.',
  typeChange: (next: string, current: string) =>
    `Trocar para "${next}" descarta os campos preenchidos de "${current}".`
} as const

export const BUILDER_TEXTS = {
  backToUsers: 'Voltar para usuários',
  cancel: 'Cancelar',
  confirmTypeChange: 'Trocar e descartar',
  createAnother: 'Criar outro ticket',
  detailsIntro: (type: string) =>
    `${type}: preencha os campos. Os marcados com * são obrigatórios.`,
  keepType: 'Manter o tipo atual',
  leave: 'Sair sem enviar',
  leaveBody: 'O que você preencheu neste ticket será perdido.',
  leaveTitle: 'Sair sem enviar?',
  reviewIntro:
    'Confira o pedido antes de enviar. Para corrigir algum campo, use "Editar".',
  send: 'Enviar ticket',
  sending: 'Enviando…',
  sessionNote:
    'Sua sessão expira após um período sem atividade. Envie o ticket antes de sair da página.',
  stay: 'Continuar no ticket',
  subtitle: 'Abra um pedido para a equipe de infraestrutura em três etapas.',
  title: 'Novo ticket'
} as const

export const SECURITY_TIP = 'Nunca escreva senhas ou tokens no ticket.'

export const TIPS = {
  acesso: [
    'Use o nome do sistema como a equipe o conhece.',
    'Peça o menor nível de acesso que resolve.',
    'Explique para que você precisa do acesso.'
  ],
  confirmacao: [
    'Guarde o número do ticket para falar com a equipe sobre ele.',
    'Para outro pedido, use "Criar outro ticket".'
  ],
  bug: [
    'Diga o que esperava e o que aconteceu.',
    'Liste os passos na ordem, um por linha.',
    '"Crítica" é para sistema fora do ar ou dados em risco.'
  ],
  deploy: [
    'Use o endereço https:// do repositório.',
    'Confira o ambiente antes de continuar.'
  ],
  outro: [
    'Use um título curto que resuma o pedido.',
    'Descreva o que você precisa e até quando.'
  ],
  revisao: ['Confira o título gerado.', 'Use "Editar" para corrigir um campo.'],
  tipo: [
    'Escolha o tipo mais próximo do seu pedido.',
    'Cada tipo pede só os campos de que a equipe precisa.',
    'Na dúvida, use "Outro".'
  ]
} as const satisfies Record<
  'confirmacao' | 'revisao' | 'tipo' | TicketType,
  readonly string[]
>
