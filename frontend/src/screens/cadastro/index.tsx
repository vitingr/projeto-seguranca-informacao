import { cn } from 'cn'
import { Check, ScrollText } from 'lucide-react'

import { Eyebrow } from '@/components/eyebrow'
import { Badge } from '@/components/ui/shadcn/badge'
import { CadastroForm } from '@/containers/cadastro-form'

import { ROLE_GUIDE } from './constants'

const RISE = 'motion-safe:animate-rise'

/**
 * The body of /cadastrar-usuarios. Source order is title, form, guide, so
 * on mobile the form comes right after the title; from lg the form takes
 * the right column across both rows.
 */
export const CadastroScreen: React.FC = () => (
  <main className="mx-auto grid w-full max-w-7xl gap-10 px-4 py-10 sm:px-6 sm:py-14 lg:grid-cols-12 lg:gap-x-16 lg:gap-y-8 lg:px-8">
    <div
      className={cn(RISE, 'flex flex-col gap-4 lg:col-span-5 lg:row-start-1')}
    >
      <Eyebrow>Pessoas</Eyebrow>
      <h1 className="text-4xl font-semibold tracking-tight sm:text-5xl">
        Cadastrar usuário
      </h1>
      <p className="text-muted-foreground text-pretty">
        Crie a conta e escolha o papel. A pessoa entra com o login e a senha que
        você definir aqui.
      </p>
    </div>

    <section
      className={cn(
        RISE,
        'motion-safe:anim-delay-75',
        'bg-card flex flex-col gap-6 self-start rounded-3xl border p-6 sm:p-10 lg:col-span-7 lg:col-start-6 lg:row-span-2 lg:row-start-1'
      )}
      aria-labelledby="nova-conta"
    >
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="text-xl font-semibold tracking-tight" id="nova-conta">
          Nova conta
        </h2>
        <Badge className="h-6 px-2.5" variant="outline">
          Só administradores
        </Badge>
      </div>
      <CadastroForm />
    </section>

    <div
      className={cn(
        RISE,
        'motion-safe:anim-delay-150',
        'flex flex-col gap-8 self-start lg:col-span-5 lg:row-start-2'
      )}
    >
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-1 xl:grid-cols-2">
        {ROLE_GUIDE.map(role => (
          <section
            className="bg-card flex flex-col gap-3 rounded-2xl border p-5"
            key={role.name}
          >
            <h2 className="text-sm font-semibold">{role.name}</h2>
            <ul className="flex flex-col gap-2">
              {role.items.map(item => (
                <li
                  className="text-muted-foreground flex gap-2 text-sm"
                  key={item}
                >
                  <Check
                    aria-hidden="true"
                    className="text-brand-ink size-4 shrink-0 translate-y-0.5"
                  />
                  {item}
                </li>
              ))}
            </ul>
          </section>
        ))}
      </div>
      <p className="text-muted-foreground flex gap-3 border-t pt-6 text-sm text-pretty">
        <ScrollText
          aria-hidden="true"
          className="size-4 shrink-0 translate-y-0.5"
        />
        Cada cadastro fica na auditoria como “Usuário cadastrado”, com você como
        ator e a nova conta como alvo.
      </p>
    </div>
  </main>
)
