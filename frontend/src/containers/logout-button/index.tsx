'use client'

import { useState } from 'react'

import { Button } from '@/components/ui/shadcn/button'
import { messageFor } from '@/domain/messages'
import { api } from '@/repositories/api'

/**
 * Finds the "Sair" button, so a page with unsent data can warn before it
 * logs out (the ticket builder, AC-3.8). The button is disabled while the
 * logout runs, and stays so until the login page loads.
 */
export const LOGOUT_SELECTOR = '[data-logout]'

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
      <Button
        data-logout=""
        disabled={pending}
        onClick={logout}
        variant="outline"
      >
        {pending ? 'Saindo…' : 'Sair'}
      </Button>
    </div>
  )
}
