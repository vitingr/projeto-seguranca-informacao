'use client'

import { CircleAlert } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'

import { Button } from '@/components/ui/shadcn/button'
import { Field, FormMessage } from '@/components/ui/shadcn/field'
import {
  NativeSelect,
  NativeSelectOption
} from '@/components/ui/shadcn/native-select'
import { messageFor } from '@/domain/messages'
import { api } from '@/repositories/api'
import { zodResolver } from '@hookform/resolvers/zod'

import { REGISTER_FORM_DEFAULT_VALUES } from './constants'
import { cadastroSchema } from './schemas'
import type { CadastroInput, CadastroValues } from './types'

export const CadastroForm = () => {
  // Opt out of the React Compiler: with it, react-hook-form's reset() did
  // not clear the inputs after a successful registration (E2E, task 8).
  'use no memo'

  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)
  const {
    formState: { errors, isSubmitting },
    handleSubmit,
    register,
    reset,
    setError: setFieldError
  } = useForm<CadastroInput, unknown, CadastroValues>({
    defaultValues: REGISTER_FORM_DEFAULT_VALUES,
    resolver: zodResolver(cadastroSchema)
  })

  // react-hook-form recommends resetting after the submission has settled,
  // not inside the submit handler (a reset there did not clear the inputs).
  useEffect(() => {
    if (success) reset(REGISTER_FORM_DEFAULT_VALUES)
  }, [success, reset])

  const onSubmit = async (values: CadastroValues) => {
    setError(null)
    setSuccess(null)
    const result = await api.register(values)
    if (result.ok) {
      // The backend answers with the normalized (lowercase) login.
      setSuccess(
        `Usuário ${result.data.login ?? values.login} cadastrado como ${result.data.role}.`
      )
      return
    }
    const message = messageFor(result.error, 'cadastro')
    if (result.error.code === 'conflict') {
      setFieldError('login', { message })
    } else {
      setError(message)
    }
  }

  return (
    <form
      className="flex flex-col gap-5"
      onSubmit={handleSubmit(onSubmit)}
      noValidate
    >
      {success ? <FormMessage tone="success">{success}</FormMessage> : null}
      {error ? <FormMessage tone="error">{error}</FormMessage> : null}
      <Field
        autoCapitalize="none"
        autoComplete="off"
        error={errors.login?.message}
        hint="3 a 64 caracteres: letras minúsculas, números, ponto, hífen ou sublinhado."
        id="login"
        label="Login"
        spellCheck={false}
        {...register('login')}
      />
      <Field
        autoComplete="new-password"
        error={errors.senha?.message}
        hint="De 15 a 128 caracteres. Uma frase longa é mais fácil de lembrar."
        id="senha"
        label="Senha"
        type="password"
        {...register('senha')}
      />
      <div className="flex flex-col gap-2">
        <label className="text-sm font-medium" htmlFor="papel">
          Papel
        </label>
        <NativeSelect
          aria-describedby={errors.papel ? 'papel-error' : 'papel-hint'}
          aria-invalid={errors.papel ? true : undefined}
          className="w-full"
          id="papel"
          size="lg"
          {...register('papel')}
        >
          <NativeSelectOption value="">Escolha…</NativeSelectOption>
          <NativeSelectOption value="comum">Usuário comum</NativeSelectOption>
          <NativeSelectOption value="admin">Administrador</NativeSelectOption>
        </NativeSelect>
        {errors.papel ? (
          <p
            className="text-destructive motion-safe:animate-in motion-safe:fade-in motion-safe:slide-in-from-top-1 flex items-start gap-1.5 text-sm"
            id="papel-error"
          >
            <CircleAlert
              aria-hidden="true"
              className="size-4 shrink-0 translate-y-0.5"
            />
            {errors.papel.message}
          </p>
        ) : (
          <p
            className="text-muted-foreground text-sm text-pretty"
            id="papel-hint"
          >
            Define as páginas que a pessoa abre e as ações que o servidor
            aceita.
          </p>
        )}
      </div>
      <Button
        className="w-full"
        disabled={isSubmitting}
        size="xl"
        type="submit"
      >
        {isSubmitting ? 'Cadastrando…' : 'Cadastrar'}
      </Button>
    </form>
  )
}
