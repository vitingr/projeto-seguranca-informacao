import { cn } from 'cn'

import type { StepIndicatorProps } from './types'

const number = (index: number) => String(index + 1).padStart(2, '0')

/**
 * A numbered list of the steps ("01 Tipo · 02 Detalhes · 03 Revisão") as its
 * own navigation element, never a label above the page title. The active
 * step is dark with a short dash in front; steps already done are buttons
 * that go back to them; later steps cannot be skipped to. A row below `md`,
 * a column from `md` up.
 */
export const StepIndicator = ({
  current,
  onSelect,
  steps
}: StepIndicatorProps) => (
  <nav aria-label="Etapas do ticket">
    <ol className="border-border flex flex-wrap gap-x-5 gap-y-2 border-b pb-3 md:flex-col md:gap-3 md:border-b-0 md:pb-0">
      {steps.map((step, index) => {
        const done = index < current
        const active = index === current
        const content = (
          <>
            <span
              className={cn(
                'bg-foreground h-px w-3 shrink-0',
                active ? 'block' : 'hidden'
              )}
              aria-hidden
            />
            <span className="text-xs tabular-nums">{number(index)}</span>
            <span>{step.label}</span>
            {done ? <span className="sr-only">(concluída)</span> : null}
          </>
        )
        return (
          <li
            data-state={active ? 'active' : done ? 'done' : 'todo'}
            key={step.id}
          >
            {done ? (
              <button
                className="text-muted-foreground hover:text-foreground focus-visible:ring-ring/50 flex items-center gap-2 rounded-sm text-sm underline-offset-4 outline-none hover:underline focus-visible:ring-3"
                onClick={() => onSelect(index)}
                type="button"
              >
                {content}
              </button>
            ) : (
              <span
                className={cn(
                  'flex items-center gap-2 text-sm',
                  active
                    ? 'text-foreground font-medium'
                    : 'text-muted-foreground'
                )}
                aria-current={active ? 'step' : undefined}
              >
                {content}
              </span>
            )}
          </li>
        )
      })}
    </ol>
  </nav>
)
