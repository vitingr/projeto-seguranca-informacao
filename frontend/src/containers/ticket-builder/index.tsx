'use client'

import { cn } from 'cn'
import { ArrowRightIcon, EyeIcon, LightbulbIcon, XIcon } from 'lucide-react'
import Link from 'next/link'
import { useEffect, useId, useReducer, useRef } from 'react'

import { SideCard } from '@/components/side-card'
import { StepIndicator } from '@/components/step-indicator'
import { TicketPreview } from '@/components/ticket-preview'
import { TicketTypePicker } from '@/components/ticket-type-picker'
import { Button, buttonVariants } from '@/components/ui/shadcn/button'
import { TICKET_TYPES, ticketTypeLabel } from '@/domain/tickets'

import {
  BUILDER_MESSAGES,
  BUILDER_TEXTS,
  SECURITY_TIP,
  STEPS,
  TIPS
} from './constants'
import { builderReducer, initialState } from './reducer'
import type { BuilderState } from './types'

const tipsFor = ({ step, type }: BuilderState): readonly string[] => [
  ...(step === 'detalhes' && type
    ? TIPS[type]
    : step === 'revisao'
      ? TIPS.revisao
      : TIPS.tipo),
  SECURITY_TIP
]

/**
 * The ticket builder (spec 001-ticket-builder): one page with the steps
 * "Tipo", "Detalhes" and "Revisão" and a confirmation, laid out after
 * `design/references/tela-criacao-tickets.png` (plan 5.4): title, step
 * indicator, the step in a bordered card, "Dicas" and "Prévia" on the side,
 * and the actions under them. The typed data lives only in this component's
 * memory (BR-7, S-10). On each step change the focus moves to the step
 * heading and the change is announced.
 */
export const TicketBuilder = () => {
  const [state, dispatch] = useReducer(builderReducer, initialState)
  const headingId = useId()
  const headingRef = useRef<HTMLHeadingElement>(null)
  const previousStep = useRef(state.step)

  const stepIndex = STEPS.findIndex(step => step.id === state.step)
  const stepLabel = STEPS[stepIndex]?.label ?? ''

  useEffect(() => {
    if (previousStep.current !== state.step) {
      previousStep.current = state.step
      headingRef.current?.focus()
    }
  }, [state.step])

  const continueFromType = () => {
    dispatch({ type: 'continueFromType' })
    if (state.type === null) {
      // AC-3.5: the focus goes to the invalid choice.
      document
        .querySelector<HTMLElement>(
          `[aria-labelledby="${headingId}"] [role="radio"]`
        )
        ?.focus()
    }
  }

  return (
    <main className="mx-auto flex w-full max-w-7xl flex-col gap-8 px-4 py-10 sm:px-6 lg:px-8">
      <div className="flex items-start justify-between gap-4">
        <div className="flex flex-col gap-1">
          <h1 className="text-2xl font-bold sm:text-3xl">
            {BUILDER_TEXTS.title}
          </h1>
          <p className="text-muted-foreground text-sm">
            {BUILDER_TEXTS.subtitle}
          </p>
        </div>
        <Link
          // `cn` lets the outline border win over the base `border-transparent`.
          className={cn(buttonVariants({ size: 'sm', variant: 'outline' }))}
          href="/usuarios"
        >
          <XIcon aria-hidden />
          {BUILDER_TEXTS.cancel}
        </Link>
      </div>

      <StepIndicator
        onSelect={index =>
          dispatch({ step: STEPS[index].id, type: 'goToStep' })
        }
        current={stepIndex}
        steps={STEPS}
      />

      <div className="grid gap-6 lg:grid-cols-10 lg:items-start">
        <section
          aria-labelledby={headingId}
          className="border-border motion-safe:animate-in motion-safe:fade-in flex flex-col gap-6 rounded-sm border p-4 motion-safe:duration-300 sm:p-6 lg:col-span-7 lg:row-start-1"
          key={state.step}
        >
          {state.step === 'tipo' ? (
            <>
              <div className="flex flex-col gap-1">
                <h2
                  className="text-base font-semibold outline-none"
                  id={headingId}
                  ref={headingRef}
                  tabIndex={-1}
                >
                  Qual é o tipo do pedido?
                </h2>
                <p className="text-muted-foreground text-sm">
                  Cada tipo tem um formulário com os campos que a equipe
                  precisa.
                </p>
              </div>
              <TicketTypePicker
                onValueChange={ticketType =>
                  dispatch({ ticketType, type: 'selectType' })
                }
                error={state.typeError ? BUILDER_MESSAGES.noType : undefined}
                labelledBy={headingId}
                options={TICKET_TYPES}
                value={state.type}
              />
            </>
          ) : (
            <h2
              className="text-base font-semibold outline-none"
              id={headingId}
              ref={headingRef}
              tabIndex={-1}
            >
              {stepLabel}
            </h2>
          )}
        </section>

        <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end lg:col-span-10 lg:row-start-2">
          {state.step === 'tipo' ? (
            <Button
              className="w-full data-[inactive=true]:opacity-50 sm:w-auto"
              data-inactive={state.type === null}
              onClick={continueFromType}
              type="button"
            >
              Continuar
              <ArrowRightIcon data-icon="inline-end" aria-hidden />
            </Button>
          ) : (
            <Button
              className="w-full sm:w-auto"
              onClick={() => dispatch({ step: 'tipo', type: 'goToStep' })}
              type="button"
              variant="outline"
            >
              Voltar
            </Button>
          )}
        </div>

        <aside
          aria-label="Ajuda e prévia"
          className="flex flex-col gap-4 lg:col-span-3 lg:col-start-8 lg:row-start-1"
        >
          <SideCard icon={LightbulbIcon} title="Dicas">
            <ul className="text-muted-foreground flex list-disc flex-col gap-1.5 pl-4 text-sm">
              {tipsFor(state).map(tip => (
                <li key={tip}>{tip}</li>
              ))}
            </ul>
          </SideCard>
          <SideCard icon={EyeIcon} title="Prévia">
            <TicketPreview
              title={null}
              type={state.type}
              typeLabel={state.type ? ticketTypeLabel(state.type) : null}
            />
          </SideCard>
        </aside>
      </div>

      <p aria-live="polite" className="sr-only" role="status">
        {`Etapa ${stepIndex + 1} de ${STEPS.length}: ${stepLabel}`}
      </p>
    </main>
  )
}
