'use client'

import { useState } from 'react'

import { api } from '@/shared/api/client'
import { messageFor } from '@/shared/api/messages'
import { Button } from '@/shared/components/ui/shadcn/button'

/**
 * Logs out through the API client: POST /api/logout with the token, then
 * the token is discarded and the page is replaced by /login (a full load,
 * so nothing of the session stays in memory).
 */
export const LogoutButton = () => {
  const [pending, setPending] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const logout = async () => {
    setPending(true)
    setError(null)
    const result = await api.logout()
    // Only reached on failure: success navigates away.
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
