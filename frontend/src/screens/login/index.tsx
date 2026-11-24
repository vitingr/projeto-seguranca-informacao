import { cn } from 'cn'
import Link from 'next/link'

import { Brand } from '@/components/brand'
import { Eyebrow } from '@/components/eyebrow'
import { LoginForm } from '@/containers/login-form'

import { LOGIN_STEPS, STEP_DELAYS } from './constants'
import type { LoginScreenProps } from './types'

const RISE = 'motion-safe:animate-rise'

/** The login page: the form, and what the server does with it. */
export const LoginScreen: React.FC<LoginScreenProps> = ({ expired, next }) => (
  <div className="grid min-h-dvh lg:grid-cols-2">
    <div className="flex flex-col">
      <header className="flex h-16 items-center px-4 sm:px-6 lg:px-8">
        <Link
          className="focus-visible:ring-ring/50 rounded-lg outline-none focus-visible:ring-3"
          href="/"
        >
          <Brand />
        </Link>
      </header>
      <main className="flex flex-1 items-center justify-center px-4 py-10 sm:px-6 lg:px-8">
        <div className="flex w-full max-w-sm flex-col gap-8">
          <div className={cn(RISE, 'flex flex-col gap-3')}>
            <Eyebrow>Acesso restrito</Eyebrow>
            <h1 className="text-4xl font-semibold tracking-tight">Entrar</h1>
            <p className="text-muted-foreground text-pretty">
              Use o login e a senha que um administrador criou para você.
            </p>
          </div>
          <div className={cn(RISE, 'motion-safe:anim-delay-75')}>
            <LoginForm expired={expired} next={next} />
          </div>
          <p
            className={cn(
              RISE,
              'motion-safe:anim-delay-150',
              'text-muted-foreground border-t pt-6 text-sm text-pretty'
            )}
          >
            Não tem conta? Não há cadastro aberto: peça a um administrador.
          </p>
        </div>
      </main>
    </div>

    <aside className="p-3 lg:p-4">
      <section className="bg-primary text-primary-foreground flex h-full flex-col justify-between gap-12 rounded-3xl p-8 sm:p-12">
        <div className={cn(RISE, 'flex flex-col gap-4')}>
          <span aria-hidden="true" className="bg-brand h-1 w-10 rounded-full" />
          <h2 className="max-w-md text-2xl font-semibold tracking-tight text-balance sm:text-3xl">
            O que acontece quando você entra
          </h2>
        </div>
        <ol className="flex flex-col">
          {LOGIN_STEPS.map((step, index) => (
            <li
              className={cn(
                RISE,
                STEP_DELAYS[index],
                'border-primary-foreground/15 flex gap-5 border-t py-5'
              )}
              key={step.tag}
            >
              <span className="text-brand w-6 shrink-0 text-sm font-semibold">
                {String(index + 1).padStart(2, '0')}
              </span>
              <span className="flex flex-col gap-2">
                <span className="text-primary-foreground/85 text-sm text-pretty">
                  {step.text}
                </span>
                <code className="text-primary-foreground/60 font-mono text-xs">
                  {step.tag}
                </code>
              </span>
            </li>
          ))}
        </ol>
        <p className="text-primary-foreground/60 text-xs text-pretty">
          Por padrão, a sessão termina depois de 15 minutos sem uso e dura no
          máximo 8 horas.
        </p>
      </section>
    </aside>
  </div>
)
