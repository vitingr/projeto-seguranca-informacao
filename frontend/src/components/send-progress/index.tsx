import { cn } from 'cn'
import { CircleCheckIcon, CircleXIcon, ListChecksIcon } from 'lucide-react'

import type { SendPhase, SendProgressProps, SendStepState } from './types'

export const SEND_STEPS = [
  'Validando campos',
  'Registrando ticket',
  'Preparando confirmação'
] as const

export const SEND_TEXTS = {
  done: 'Envio concluído',
  failed: 'Envio interrompido',
  fields: 'Campos do pedido',
  list: 'Progresso do envio',
  sending: 'Enviando ticket',
  states: {
    active: '(em andamento)',
    done: '(concluída)',
    failed: '(falhou)'
  }
} as const

const MAX_CHIPS = 5

const STATES: Record<SendPhase, readonly SendStepState[]> = {
  done: ['done', 'done', 'done'],
  failed: ['done', 'failed'],
  registering: ['done', 'active']
}

const seconds = (ms: number) =>
  (ms / 1000).toLocaleString('pt-BR', {
    maximumFractionDigits: 1,
    minimumFractionDigits: 1
  })

export const progressHeading = (phase: SendPhase, elapsedMs: number) =>
  phase === 'registering'
    ? { muted: `${Math.floor(elapsedMs / 1000)} s`, text: SEND_TEXTS.sending }
    : phase === 'done'
      ? { muted: null, text: `Enviado em ${seconds(elapsedMs)} s` }
      : { muted: `após ${seconds(elapsedMs)} s`, text: SEND_TEXTS.failed }

export const SendProgress = ({
  chips,
  elapsedMs,
  enterFrom = 0,
  phase
}: SendProgressProps) => {
  const states = STATES[phase]
  const heading = progressHeading(phase, elapsedMs)
  const shown = chips.slice(0, MAX_CHIPS)
  const hidden = chips.length - shown.length
  const enter = (index: number) =>
    index >= enterFrom && 'motion-safe:animate-enter-blur'
  const rows = 1 + states.length + (phase === 'done' ? 1 : 0)

  const line = (index: number) =>
    index < rows - 1 ? (
      <span className="bg-border min-h-2 w-px flex-1" aria-hidden />
    ) : null

  return (
    <div
      className="flex flex-col gap-3"
      data-phase={phase}
      data-testid="send-progress"
    >
      <div className="flex items-center gap-3">
        <span
          className="progress-squares grid size-4 shrink-0 grid-cols-3 place-content-center gap-px"
          aria-hidden
        >
          {Array.from({ length: 9 }, (_, index) => (
            <span
              className={cn(
                'size-1',
                phase === 'failed' ? 'bg-muted-foreground' : 'bg-brand',
                phase === 'registering' && 'motion-safe:animate-blink'
              )}
              key={index}
            />
          ))}
        </span>
        <p
          className="motion-safe:animate-enter-blur flex items-baseline gap-2 text-sm"
          key={phase}
        >
          <span
            className={cn(
              phase === 'registering'
                ? 'text-foreground motion-safe:animate-pulse'
                : phase === 'failed'
                  ? 'text-destructive'
                  : 'text-foreground font-medium'
            )}
          >
            {heading.text}
          </span>
          {heading.muted ? (
            <span className="text-muted-foreground text-xs tabular-nums">
              {heading.muted}
            </span>
          ) : null}
        </p>
      </div>

      <ol aria-label={SEND_TEXTS.list} className="flex flex-col">
        <li className={cn('flex gap-3', enter(0))}>
          <span className="flex w-4 shrink-0 flex-col items-center gap-1">
            <ListChecksIcon
              className="text-muted-foreground size-4 shrink-0"
              aria-hidden
            />
            {line(0)}
          </span>
          <div className="flex min-w-0 flex-col gap-2 pb-3">
            <span className="text-muted-foreground text-sm">
              {SEND_TEXTS.fields}
            </span>
            <ul className="flex flex-wrap gap-1.5">
              {shown.map(chip => (
                <li
                  className="border-border bg-background rounded-sm border px-2 py-0.5 text-xs"
                  key={chip}
                >
                  {chip}
                </li>
              ))}
              {hidden > 0 ? (
                <li className="bg-muted text-muted-foreground rounded-sm px-2 py-0.5 text-xs">
                  {`+${hidden}`}
                </li>
              ) : null}
            </ul>
          </div>
        </li>

        {states.map((state, index) => (
          <li
            className={cn('flex gap-3', enter(index + 1))}
            data-state={state}
            key={SEND_STEPS[index]}
          >
            <span className="flex w-4 shrink-0 flex-col items-center gap-1">
              <span className="flex h-5 items-center" aria-hidden>
                {state === 'failed' ? (
                  <CircleXIcon className="text-destructive size-4" />
                ) : (
                  <span
                    className={cn(
                      'rounded-full',
                      state === 'active'
                        ? 'bg-brand size-2 motion-safe:animate-pulse'
                        : 'bg-muted-foreground size-1.5'
                    )}
                  />
                )}
              </span>
              {line(index + 1)}
            </span>
            <span
              className={cn(
                'pb-3 text-sm',
                state === 'active'
                  ? 'text-foreground motion-safe:animate-pulse'
                  : state === 'failed'
                    ? 'text-destructive'
                    : 'text-muted-foreground'
              )}
            >
              {SEND_STEPS[index]}
              <span className="sr-only"> {SEND_TEXTS.states[state]}</span>
            </span>
          </li>
        ))}

        {phase === 'done' ? (
          <li className={cn('flex gap-3', enter(rows - 1))}>
            <span className="flex h-5 w-4 shrink-0 items-center" aria-hidden>
              <CircleCheckIcon className="fill-brand text-brand-foreground size-4" />
            </span>
            <span className="text-sm font-medium">{SEND_TEXTS.done}</span>
          </li>
        ) : null}
      </ol>
    </div>
  )
}
