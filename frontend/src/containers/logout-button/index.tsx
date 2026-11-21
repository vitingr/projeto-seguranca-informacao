'use client'

import { useState } from 'react'

import { Button } from '@/components/ui/shadcn/button'
import { messageFor } from '@/domain/messages'
import { api } from '@/repositories/api'

export const LogoutButton = () => {
  const [pending, setPending] = useState<boolean>(false)
  const [error, setError] = useState<string | null>(null)

  const logout = async () => {
    setPending(true)
    setError(null)
    const result = await api.logout()
    if (!result.ok) {
      setError(messageFor(result.error, 'action'))
      setPending(false)
    }
  }

  return (
    <div className="flex items-center gap-2">
      {error ? (
        <span className="text-destructive text-sm" data-testid="logout-error">
          {error}
        </span>
      ) : null}
      <Button disabled={pending} onClick={logout} variant="outline">
        {pending ? 'Saindo…' : 'Sair'}
      </Button>
    </div>
  )
}
