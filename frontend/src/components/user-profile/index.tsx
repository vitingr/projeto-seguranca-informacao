import { cn } from 'cn'
import { ShieldCheckIcon, TicketPlusIcon } from 'lucide-react'
import Link from 'next/link'

import { Avatar } from '@/components/avatar'
import { DetailList, DetailRow } from '@/components/detail-list'
import { Badge } from '@/components/ui/shadcn/badge'
import { buttonVariants } from '@/components/ui/shadcn/button'
import { formatDate } from '@/domain/date-time'

import type { ProfileCardProps, UserProfileProps } from './types'

export const PROFILE_TEXTS = {
  actions: 'Ações',
  /** Exactly the text of the assignment (spec AC-0.1, AC-7.1). */
  description: 'Página de acesso restrito aos usuários comuns',
  limits: [
    'A sessão termina depois de um período sem atividade; aí é só entrar de novo.',
    'Você pode enviar até 10 tickets por minuto.'
  ],
  limitsTitle: 'Sessão e limites',
  memberSince: 'Membro desde',
  newTicket: 'Novo ticket',
  newTicketHint: 'Abra um pedido para a equipe de infraestrutura.',
  notInformed: '—',
  profile: 'Perfil',
  profileHint: 'Os dados da sua conta, como o servidor os guarda.',
  role: 'Usuário comum',
  roleLabel: 'Papel',
  title: 'Minha conta'
} as const

/**
 * A bordered card of the side column, like the builder's, with no toggle.
 * The page has one profile, so the ids are fixed (this renders on the
 * server, with no client state).
 */
const ProfileCard = ({ children, icon: Icon, id, title }: ProfileCardProps) => (
  <section
    aria-labelledby={id}
    className="border-border flex flex-col gap-3 rounded-sm border p-4"
  >
    <h2 className="flex items-center gap-2 text-sm font-medium" id={id}>
      <Icon className="text-muted-foreground size-4" aria-hidden />
      {title}
    </h2>
    {children}
  </section>
)

/**
 * The body of `/usuarios` (spec US-7, plan 5.6): the exact text of the
 * assignment and the area line, then the profile in the layout of the
 * builder (a main card next to a side column). Everything comes from the
 * backend for the session user; the user id is never shown (AC-7.3). The
 * avatar is decorative and the login is the text next to it (AC-7.2).
 */
export const UserProfile = ({
  area,
  children,
  createdAt,
  login
}: UserProfileProps) => {
  const profileId = 'profile-title'
  const since = createdAt ? formatDate(createdAt) : null
  return (
    <main className="mx-auto flex w-full max-w-7xl flex-col gap-8 px-4 py-10 sm:px-6 lg:px-8">
      <div className="flex flex-col gap-1">
        <h1 className="text-2xl font-bold sm:text-3xl">
          {PROFILE_TEXTS.title}
        </h1>
        <p>{PROFILE_TEXTS.description}</p>
        <p className="text-muted-foreground text-sm" data-testid="area">
          Área liberada pelo servidor: {area}
        </p>
      </div>

      <div className="grid gap-6 lg:grid-cols-10 lg:items-start">
        <section
          aria-labelledby={profileId}
          className="border-border flex flex-col gap-6 rounded-sm border p-4 sm:p-6 lg:col-span-7"
        >
          <div className="flex flex-col gap-1">
            <h2 className="text-base font-semibold" id={profileId}>
              {PROFILE_TEXTS.profile}
            </h2>
            <p className="text-muted-foreground text-sm">
              {PROFILE_TEXTS.profileHint}
            </p>
          </div>
          <div
            className="flex items-center gap-4"
            data-testid="profile-identity"
          >
            <Avatar login={login ?? ''} />
            <div className="flex min-w-0 flex-col items-start gap-1.5">
              <p className="text-lg font-semibold break-all">
                {login ?? PROFILE_TEXTS.notInformed}
              </p>
              <Badge variant="secondary">{PROFILE_TEXTS.role}</Badge>
            </div>
          </div>
          <DetailList>
            <DetailRow label="Login">
              {login ?? PROFILE_TEXTS.notInformed}
            </DetailRow>
            <DetailRow label={PROFILE_TEXTS.roleLabel}>
              {PROFILE_TEXTS.role}
            </DetailRow>
            <DetailRow label={PROFILE_TEXTS.memberSince}>
              {since ?? PROFILE_TEXTS.notInformed}
            </DetailRow>
          </DetailList>
        </section>

        <aside
          aria-label="Ações e limites"
          className="flex flex-col gap-4 lg:col-span-3"
        >
          <ProfileCard
            icon={TicketPlusIcon}
            id="profile-actions-title"
            title={PROFILE_TEXTS.actions}
          >
            <p className="text-muted-foreground text-sm">
              {PROFILE_TEXTS.newTicketHint}
            </p>
            <Link
              className={cn(buttonVariants(), 'w-full sm:w-fit lg:w-full')}
              href="/usuarios/tickets/novo"
            >
              {PROFILE_TEXTS.newTicket}
            </Link>
          </ProfileCard>
          <ProfileCard
            icon={ShieldCheckIcon}
            id="profile-limits-title"
            title={PROFILE_TEXTS.limitsTitle}
          >
            <ul className="text-muted-foreground flex list-disc flex-col gap-1.5 pl-4 text-sm">
              {PROFILE_TEXTS.limits.map(fact => (
                <li key={fact}>{fact}</li>
              ))}
            </ul>
          </ProfileCard>
        </aside>
      </div>
      {children}
    </main>
  )
}
