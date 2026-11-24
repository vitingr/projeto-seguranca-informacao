import { cn } from 'cn'
import { AppWindow, CircleAlert, CircleCheck, ScrollText } from 'lucide-react'

import { Badge } from '@/components/ui/shadcn/badge'

import type { DemoAuditRowProps, DemoPaneProps } from './types'

// A looping, CSS-only demo of what the system does: what users see on the
// left, what the audit log records on the right. The steps enter in order
// (keyframes demo-1..7 in globals.css). With reduced motion, every step is
// shown at once, still. Illustrative data only: nothing is fetched.

const DemoPane: React.FC<DemoPaneProps> = ({ children, icon, title }) => (
  <div className="bg-card flex flex-col gap-3 rounded-2xl border p-3 sm:p-4">
    <div className="text-muted-foreground flex items-center gap-2 text-xs font-medium">
      {icon}
      {title}
    </div>
    {children}
  </div>
)

const DemoAuditRow: React.FC<DemoAuditRowProps> = ({
  className,
  detail,
  event,
  time,
  variant,
  who
}) => (
  <li
    className={cn(
      'bg-card flex items-center gap-3 rounded-xl border px-3 py-2.5',
      className
    )}
  >
    <Badge className="h-6 px-2.5" variant={variant}>
      {event}
    </Badge>
    <span className="flex min-w-0 flex-1 flex-col">
      <span className="truncate text-xs font-medium">{who}</span>
      <span className="text-muted-foreground truncate font-mono text-xs">
        {detail}
      </span>
    </span>
    <span className="text-muted-foreground font-mono text-xs">{time}</span>
  </li>
)

export const SecurityDemo: React.FC = () => (
  <figure
    aria-label="Demonstração: uma sessão do login à moderação, e o que fica registrado na auditoria"
    className="bg-muted flex flex-col gap-2 rounded-3xl border p-2 sm:gap-3 sm:p-3"
  >
    <figcaption className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1 px-2 pt-1">
      <span className="text-sm font-semibold tracking-tight">
        Uma sessão, do login à moderação
      </span>
      <span className="text-muted-foreground text-xs">Dados de exemplo</span>
    </figcaption>

    <div className="grid gap-2 sm:grid-cols-2 sm:gap-3">
      <DemoPane
        icon={<AppWindow aria-hidden="true" className="size-4" />}
        title="O que o usuário vê"
      >
        <ol className="flex flex-col gap-2">
          <li className="bg-background motion-safe:animate-demo-1 flex flex-col gap-2 rounded-xl border p-3">
            <span className="flex items-center justify-between text-xs">
              <span className="font-medium">Tela de login</span>
              <span className="text-muted-foreground font-mono">14:02:07</span>
            </span>
            <span className="flex gap-2 text-xs">
              <span className="bg-card flex h-8 min-w-0 flex-1 items-center rounded-lg border px-2.5">
                maria.lima
              </span>
              <span className="bg-card text-muted-foreground flex h-8 w-20 items-center rounded-lg border px-2.5 tracking-widest">
                ••••••
              </span>
            </span>
            <span className="text-destructive flex items-center gap-1.5 text-xs font-medium">
              <CircleAlert aria-hidden="true" className="size-3.5" />
              Login ou senha incorretos.
            </span>
          </li>

          <li className="bg-background motion-safe:animate-demo-3 flex flex-col gap-1.5 rounded-xl border p-3">
            <span className="flex items-center justify-between text-xs">
              <span className="font-medium">Tela de login</span>
              <span className="text-muted-foreground font-mono">14:02:31</span>
            </span>
            <span className="text-success flex items-center gap-1.5 text-xs font-medium">
              <CircleCheck aria-hidden="true" className="size-3.5" />
              Sessão criada
            </span>
            <span className="text-muted-foreground font-mono text-xs">
              __Host-session · HttpOnly
            </span>
          </li>

          <li className="bg-background motion-safe:animate-demo-5 flex flex-col gap-2.5 rounded-xl border p-3">
            <span className="flex items-center gap-2.5">
              <span
                aria-hidden="true"
                className="bg-brand-soft text-brand-ink flex size-7 items-center justify-center rounded-full text-xs font-semibold"
              >
                ML
              </span>
              <span className="flex min-w-0 flex-1 flex-col">
                <span className="text-xs font-medium">maria.lima · Mural</span>
                <span className="text-muted-foreground font-mono text-xs">
                  14:05:48
                </span>
              </span>
            </span>
            <span className="motion-safe:animate-demo-dim flex flex-col gap-0.5">
              <span className="text-sm font-semibold tracking-tight">
                Troca de turno na sexta
              </span>
              <span className="text-muted-foreground text-xs">
                Alguém cobre o meu turno das 14h? Me chama no (61) 9xxxx-4521.
              </span>
            </span>
            <span className="bg-warning-soft text-warning motion-safe:animate-demo-6 rounded-lg px-2.5 py-2 text-xs">
              <span className="font-semibold">Removido pela moderação:</span>{' '}
              telefone pessoal exposto.
            </span>
          </li>
        </ol>
      </DemoPane>

      <DemoPane
        icon={<ScrollText aria-hidden="true" className="size-4" />}
        title="O que a auditoria registra"
      >
        <ol className="flex flex-col gap-2">
          <DemoAuditRow
            className="motion-safe:animate-demo-2"
            detail="10.0.4.17"
            event="Falha de login"
            time="14:02:07"
            variant="destructive"
            who="maria.lima"
          />
          <DemoAuditRow
            className="motion-safe:animate-demo-4"
            detail="10.0.4.17"
            event="Login"
            time="14:02:31"
            variant="success"
            who="maria.lima"
          />
          <DemoAuditRow
            className="motion-safe:animate-demo-7"
            detail="Post #42"
            event="Post removido"
            time="14:07:12"
            variant="warning"
            who="admin"
          />
        </ol>
        <p className="text-muted-foreground border-t border-dashed pt-3 text-xs text-pretty">
          Publicar no mural não gera registro. A auditoria guarda cinco eventos:
          login, falha de login, logout, usuário cadastrado e post removido.
        </p>
      </DemoPane>
    </div>

    <div aria-hidden="true" className="px-2 pb-1 motion-reduce:hidden">
      <div className="bg-border h-0.5 overflow-hidden rounded-full">
        <div className="bg-brand motion-safe:animate-demo-progress h-full origin-left" />
      </div>
    </div>
  </figure>
)
