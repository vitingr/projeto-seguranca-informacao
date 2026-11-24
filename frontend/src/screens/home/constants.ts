import {
  MessagesSquare,
  ScrollText,
  ShieldCheck,
  UserPlus,
  UserRound
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

import type { ProtectedRoute } from '@/domain/protected-routes/types'

// Presentation copy for each area card on the signed-in home.
export const AREA_DETAILS: Record<
  ProtectedRoute,
  { description: string; icon: LucideIcon; kicker: string }
> = {
  '/admin': {
    description: 'A área que o servidor libera só para administradores.',
    icon: ShieldCheck,
    kicker: 'Administração'
  },
  '/admin/auditoria': {
    description:
      'Logins, falhas, logouts, cadastros e remoções, com ator, IP e horário.',
    icon: ScrollText,
    kicker: 'Registro'
  },
  '/cadastrar-usuarios': {
    description: 'Crie contas e escolha o papel de cada pessoa.',
    icon: UserPlus,
    kicker: 'Pessoas'
  },
  '/minha-conta': {
    description: 'A sua área, liberada pelo servidor para o seu papel.',
    icon: UserRound,
    kicker: 'Conta'
  },
  '/mural': {
    description: 'Os posts de todos os usuários, com moderação.',
    icon: MessagesSquare,
    kicker: 'Comunidade'
  }
}

// Entrance stagger for lists, in order. Classes, not inline styles.
export const STAGGER = [
  'motion-safe:anim-delay-0',
  'motion-safe:anim-delay-75',
  'motion-safe:anim-delay-150',
  'motion-safe:anim-delay-225',
  'motion-safe:anim-delay-300',
  'motion-safe:anim-delay-375'
] as const

// Facts shown under the hero call to action.
export const FACTS = [
  { label: 'hash das senhas', value: 'Argon2id' },
  { label: 'sem uso encerra a sessão', value: '15 min' },
  { label: 'registrados na auditoria', value: '5 eventos' }
] as const

export const ROLES = [
  {
    items: [
      'Cadastra usuários e escolhe o papel de cada um.',
      'Lê o mural e remove posts, sempre informando o motivo.',
      'Consulta a auditoria e filtra por tipo de evento.',
      'Não publica no mural: o papel dele é moderar.'
    ],
    name: 'Administradores',
    summary: 'Cuidam das contas, da moderação e da auditoria.'
  },
  {
    items: [
      'Publica posts no mural.',
      'Edita e exclui os próprios posts, e só os próprios.',
      'Vê o motivo quando um post seu é removido.',
      'Acessa a própria conta.'
    ],
    name: 'Usuários comuns',
    summary: 'Participam do mural e cuidam da própria conta.'
  }
] as const

export const CONTROLS = [
  {
    description:
      'As senhas são guardadas como hash Argon2id, com 64 MiB de memória por cálculo. Nunca em texto.',
    tag: 'argon2id',
    title: 'Senhas com Argon2id'
  },
  {
    description:
      'A sessão fica num cookie Secure, HttpOnly e SameSite=Lax: o JavaScript da página não consegue lê-la.',
    tag: '__Host-session',
    title: 'Cookie de sessão protegido'
  },
  {
    description:
      'Por padrão, a sessão cai depois de 15 minutos sem uso e dura no máximo 8 horas.',
    tag: '15 min · 8 h',
    title: 'Sessão com prazo'
  },
  {
    description:
      'Toda ação que muda dados exige um token num cabeçalho, comparado em tempo constante. Pedidos de outra origem são recusados.',
    tag: 'X-CSRF-Token',
    title: 'Proteção contra CSRF'
  },
  {
    description:
      'Cada página e cada rota da API conferem o papel de novo. O que a interface esconde, o servidor também bloqueia.',
    tag: '403',
    title: 'Papéis checados no servidor'
  },
  {
    description:
      'Login, falha de login, logout, cadastro de usuário e remoção de post ficam registrados com ator, IP e horário.',
    tag: 'audit_events',
    title: 'Auditoria'
  }
] as const
