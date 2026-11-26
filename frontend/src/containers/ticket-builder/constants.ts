import { ClipboardCheckIcon, FileTextIcon, ListChecksIcon } from 'lucide-react'

import type { TicketType } from '@/domain/tickets/types'

import type { Step } from './types'

/** The steps in order, as the step indicator shows them. */
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
  noType: 'Escolha um tipo de ticket.'
} as const

export const BUILDER_TEXTS = {
  cancel: 'Cancelar',
  subtitle: 'Abra um pedido para a equipe de infraestrutura em três etapas.',
  title: 'Novo ticket'
} as const

/** Shown on every step and type: tickets are read by people. */
export const SECURITY_TIP = 'Nunca escreva senhas ou tokens no ticket.'

/** The "Dicas" card (plan 5.4): short and true, per step and type. */
export const TIPS = {
  acesso: [
    'Use o nome do sistema como a equipe o conhece.',
    'Peça o menor nível de acesso que resolve.',
    'Explique para que você precisa do acesso.'
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
} as const satisfies Record<'revisao' | 'tipo' | TicketType, readonly string[]>
