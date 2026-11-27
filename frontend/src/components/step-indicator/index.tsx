import { cn } from 'cn'
import { CheckIcon, ChevronRightIcon } from 'lucide-react'

import type { StepIndicatorProps } from './types'

export const StepIndicator = ({
  current,
  locked = false,
  onSelect,
  steps
}: StepIndicatorProps) => {
  const active = steps[current]

  return (
    <nav aria-label="Etapas do ticket" className="flex flex-col gap-3">
      <ol className="flex items-center gap-2">
        {steps.map((step, index) => {
          const done = index < current
          const isActive = index === current
          const Icon = done ? CheckIcon : step.icon
          const content = (
            <>
              <span
                className={cn(
                  'flex size-9 shrink-0 items-center justify-center rounded-full',
                  isActive
                    ? 'bg-brand text-brand-foreground'
                    : done
                      ? 'bg-muted text-foreground'
                      : 'bg-muted text-muted-foreground'
                )}
                aria-hidden
              >
                <Icon className="size-4" />
              </span>
              <span className="sr-only flex-col gap-0.5 text-left md:not-sr-only md:flex">
                <span
                  className={cn(
                    'text-sm',
                    isActive || done
                      ? 'text-foreground font-medium'
                      : 'text-muted-foreground'
                  )}
                >
                  {step.label}
                </span>
                <span className="text-muted-foreground text-xs">
                  {step.description}
                </span>
                {done ? <span className="sr-only">(concluída)</span> : null}
              </span>
            </>
          )
          return (
            <li
              className={cn(
                'flex items-center gap-2',
                index < steps.length - 1 && 'flex-1'
              )}
              data-state={isActive ? 'active' : done ? 'done' : 'todo'}
              key={step.id}
            >
              {done && !locked ? (
                <button
                  className="hover:bg-muted focus-visible:ring-ring/50 flex items-center gap-3 rounded-sm text-left outline-none focus-visible:ring-3 md:pr-2"
                  onClick={() => onSelect(index)}
                  type="button"
                >
                  {content}
                </button>
              ) : (
                <span
                  aria-current={isActive ? 'step' : undefined}
                  className="flex items-center gap-3"
                >
                  {content}
                </span>
              )}
              {index < steps.length - 1 ? (
                <span className="flex flex-1 justify-center" aria-hidden>
                  <ChevronRightIcon className="text-muted-foreground size-4" />
                </span>
              ) : null}
            </li>
          )
        })}
      </ol>
      {active ? (
        <p className="flex flex-col gap-0.5 md:hidden" aria-hidden>
          <span className="text-sm font-medium">{active.label}</span>
          <span className="text-muted-foreground text-xs">
            {active.description}
          </span>
        </p>
      ) : null}
    </nav>
  )
}
