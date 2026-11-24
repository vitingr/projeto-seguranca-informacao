import { cn } from 'cn'
import Image from 'next/image'
import Link from 'next/link'

import { Brand } from '@/components/brand'
import { LoginForm } from '@/containers/login-form'

import { RISE } from './constants'
import type { LoginScreenProps } from './types'

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
              'text-muted-foreground border-t pt-6 text-center text-sm text-pretty'
            )}
          >
            Não tem conta? Não há cadastro aberto: peça a um administrador.
          </p>
        </div>
      </main>
    </div>

    <aside className="">
      <figure className="h-full w-full">
        <Image
          alt="Login Background Image"
          fetchPriority="high"
          height={1920}
          loading="eager"
          src="/login-background.png"
          width={1920}
        />
      </figure>
    </aside>
  </div>
)
