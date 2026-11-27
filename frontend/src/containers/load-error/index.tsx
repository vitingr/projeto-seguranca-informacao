'use client'

import { CircleAlertIcon } from 'lucide-react'
import { useRouter } from 'next/navigation'
import { useTransition } from 'react'

import { Alert, AlertDescription } from '@/components/ui/shadcn/alert'
import { Button } from '@/components/ui/shadcn/button'

import type { LoadErrorProps } from './types'

export const LOAD_ERROR_TEXTS = {
  retry: 'Tentar novamente',
  retrying: 'Tentando…'
} as const

export const LoadError = ({ message }: LoadErrorProps) => {
  const router = useRouter()
  const [pending, startTransition] = useTransition()
  return (
    <Alert
      className="border-destructive/40 flex flex-col gap-3 px-4 py-3"
      data-testid="load-error"
      variant="destructive"
    >
      <div className="flex items-start gap-2">
        <CircleAlertIcon className="size-4 shrink-0" aria-hidden />
        <AlertDescription className="text-destructive">
          {message}
        </AlertDescription>
      </div>
      <Button
        onClick={() => {
          if (pending) return
          startTransition(() => {
            router.refresh()
          })
        }}
        aria-disabled={pending || undefined}
        className="w-fit"
        type="button"
        variant="outline"
      >
        {pending ? LOAD_ERROR_TEXTS.retrying : LOAD_ERROR_TEXTS.retry}
      </Button>
    </Alert>
  )
}
