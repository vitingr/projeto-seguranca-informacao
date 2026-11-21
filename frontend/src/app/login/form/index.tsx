'use client'

import { useRouter } from 'next/navigation'
import { useState } from 'react'
import { useForm } from 'react-hook-form'

import { MESSAGES, messageFor } from '@/domain/messages'
import { safeNext } from '@/domain/navigation'
import { api } from '@/repositories/api'
import { Button } from '@/shared/components/ui/shadcn/button'
import { Field, FormMessage } from '@/shared/components/ui/shadcn/field'
import { zodResolver } from '@hookform/resolvers/zod'

import { LOGIN_FORM_DEFAULT_VALUES } from './constants'
import type { LoginValues } from './schemas'
import { loginSchema } from './schemas'

export const LoginForm = ({
  expired,
  next
}: {
  expired: boolean
  next?: string
}) => {
  const router = useRouter()
  const [error, setError] = useState<string | null>(null)
  const [navigating, setNavigating] = useState(false)
  const {
    formState: { errors, isSubmitting },
    handleSubmit,
    register,
    resetField,
    setFocus
  } = useForm<LoginValues>({
    defaultValues: LOGIN_FORM_DEFAULT_VALUES,
    resolver: zodResolver(loginSchema)
  })

  const onSubmit = async (values: LoginValues) => {
    setError(null)
    const result = await api.login(values)
    if (result.ok) {
      // Client navigation keeps the new CSRF token in memory; the target is
      // one of the user's own routes (safeNext), never a URL from the query.
      setNavigating(true)
      router.replace(safeNext(next, result.data.role))
      return
    }
    setError(messageFor(result.error, 'login'))
    if (result.error.code === 'unauthorized') {
      resetField('senha')
      setFocus('senha')
    }
  }

  const pending = isSubmitting || navigating

  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={handleSubmit(onSubmit)}
      noValidate
    >
      {expired && !error ? (
        <FormMessage tone="error">{MESSAGES.sessionExpired}</FormMessage>
      ) : null}
      {error ? <FormMessage tone="error">{error}</FormMessage> : null}
      <Field
        autoCapitalize="none"
        autoComplete="username"
        error={errors.login?.message}
        id="login"
        label="Login"
        spellCheck={false}
        {...register('login')}
      />
      <Field
        autoComplete="current-password"
        error={errors.senha?.message}
        id="senha"
        label="Senha"
        type="password"
        {...register('senha')}
      />
      <Button disabled={pending} size="lg" type="submit">
        {pending ? 'Entrando…' : 'Entrar'}
      </Button>
    </form>
  )
}
