'use client'

import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'

import { api } from '@/shared/api/client'
import { messageFor } from '@/shared/api/messages'
import { Field, FormMessage } from '@/shared/components/ui/field'
import { Button } from '@/shared/components/ui/shadcn/button'
import {
  type CadastroInput,
  type CadastroValues,
  cadastroSchema
} from '@/shared/validation/cadastro'
import { zodResolver } from '@hookform/resolvers/zod'

const empty: CadastroInput = { login: '', papel: '', senha: '' }

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
    defaultValues: empty,
    resolver: zodResolver(cadastroSchema)
  })

  // react-hook-form recommends resetting after the submission has settled,
  // not inside the submit handler (a reset there did not clear the inputs).
  useEffect(() => {
    if (success) reset(empty)
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
      className="flex flex-col gap-4"
      onSubmit={handleSubmit(onSubmit)}
      noValidate
    >
      {success ? <FormMessage tone="success">{success}</FormMessage> : null}
      {error ? <FormMessage tone="error">{error}</FormMessage> : null}
      <Field
        autoCapitalize="none"
        autoComplete="off"
        error={errors.login?.message}
        id="login"
        label="Login"
        spellCheck={false}
        {...register('login')}
      />
      <Field
        autoComplete="new-password"
        error={errors.senha?.message}
        id="senha"
        label="Senha"
        type="password"
        {...register('senha')}
      />
      <div className="flex flex-col gap-1.5">
        <label className="text-sm font-medium" htmlFor="papel">
          Papel
        </label>
        <select
          aria-describedby={errors.papel ? 'papel-error' : undefined}
          aria-invalid={errors.papel ? true : undefined}
          className="border-input bg-background focus-visible:border-ring focus-visible:ring-ring/50 aria-invalid:border-destructive h-9 rounded-lg border px-3 text-sm outline-none focus-visible:ring-3"
          id="papel"
          {...register('papel')}
        >
          <option value="">Escolha…</option>
          <option value="comum">Usuário comum</option>
          <option value="admin">Administrador</option>
        </select>
        {errors.papel ? (
          <p className="text-destructive text-sm" id="papel-error">
            {errors.papel.message}
          </p>
        ) : null}
      </div>
      <Button disabled={isSubmitting} size="lg" type="submit">
        {isSubmitting ? 'Cadastrando…' : 'Cadastrar'}
      </Button>
    </form>
  )
}
