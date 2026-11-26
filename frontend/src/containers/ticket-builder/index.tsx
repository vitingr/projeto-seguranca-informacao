'use client'

import { useEffect, useId, useReducer, useRef } from 'react'

import { StepIndicator } from '@/components/step-indicator'
import { TicketTypePicker } from '@/components/ticket-type-picker'
import { Button } from '@/components/ui/shadcn/button'
import { TICKET_TYPES } from '@/domain/tickets'

import { BUILDER_MESSAGES, STEPS } from './constants'
import { builderReducer, initialState } from './reducer'

/**
 * The ticket builder (spec 001-ticket-builder): one page with the steps
 * "Tipo", "Detalhes" and "Revisão" and a confirmation. The typed data lives
 * only in this component's memory (BR-7, S-10). On each step change the
 * focus moves to the step heading and the change is announced.
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
    <main className="mx-auto flex w-full max-w-3xl flex-col gap-8 px-4 py-10">
      <div className="flex flex-col gap-2">
        <h1 className="text-2xl font-semibold">Novo ticket</h1>
        <p className="text-muted-foreground text-sm">
          Escolha o tipo de pedido, preencha os detalhes e revise antes de
          enviar para a equipe de infraestrutura.
        </p>
      </div>

      <div className="grid gap-8 md:grid-cols-4">
        <StepIndicator
          onSelect={index =>
            dispatch({ step: STEPS[index].id, type: 'goToStep' })
          }
          current={stepIndex}
          steps={STEPS}
        />

        <section
          aria-labelledby={headingId}
          className="motion-safe:animate-in motion-safe:fade-in flex flex-col gap-6 motion-safe:duration-300 md:col-span-3"
          key={state.step}
        >
          {state.step === 'tipo' ? (
            <>
              <div className="flex flex-col gap-1">
                <h2
                  className="text-lg font-semibold outline-none"
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
              <div className="flex justify-end">
                <Button
                  className="w-full data-[inactive=true]:opacity-50 sm:w-auto"
                  data-inactive={state.type === null}
                  onClick={continueFromType}
                  size="lg"
                  type="button"
                >
                  Continuar
                </Button>
              </div>
            </>
          ) : (
            <>
              <h2
                className="text-lg font-semibold outline-none"
                id={headingId}
                ref={headingRef}
                tabIndex={-1}
              >
                {stepLabel}
              </h2>
              <div className="flex">
                <Button
                  onClick={() => dispatch({ step: 'tipo', type: 'goToStep' })}
                  size="lg"
                  type="button"
                  variant="outline"
                >
                  Voltar
                </Button>
              </div>
            </>
          )}
        </section>
      </div>

      <p aria-live="polite" className="sr-only" role="status">
        {`Etapa ${stepIndex + 1} de ${STEPS.length}: ${stepLabel}`}
      </p>
    </main>
  )
}
