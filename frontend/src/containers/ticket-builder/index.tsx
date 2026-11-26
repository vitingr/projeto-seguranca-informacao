'use client'

import { cn } from 'cn'
import {
  ArrowRightIcon,
  ClockIcon,
  EyeIcon,
  LightbulbIcon,
  LoaderCircleIcon,
  PencilIcon,
  PlusIcon,
  SendIcon,
  XIcon
} from 'lucide-react'
import Link from 'next/link'
import { useEffect, useId, useReducer, useRef, useState } from 'react'
import { Controller, type Resolver, useForm, useWatch } from 'react-hook-form'

import { ChoiceSelect } from '@/components/choice-select'
import { FormField, FormInput, fieldIds } from '@/components/form-field'
import { InlineAlert } from '@/components/inline-alert'
import { SendProgress } from '@/components/send-progress'
import type { SendPhase } from '@/components/send-progress/types'
import { SideCard } from '@/components/side-card'
import { StepIndicator } from '@/components/step-indicator'
import { TextAreaField } from '@/components/text-area-field'
import { TicketConfirmation } from '@/components/ticket-confirmation'
import { TicketPreview } from '@/components/ticket-preview'
import { TicketReview } from '@/components/ticket-review'
import { TicketTypePicker } from '@/components/ticket-type-picker'
import { Button, buttonVariants } from '@/components/ui/shadcn/button'
import {
  FIELD_LIMITS,
  TICKET_TYPES,
  codePoints,
  ticketTitle,
  ticketTypeLabel
} from '@/domain/tickets'
import type { TicketType } from '@/domain/tickets/types'
import { api } from '@/repositories/api'
import { singleFlight } from '@/utils/single-flight'
import { zodResolver } from '@hookform/resolvers/zod'

import {
  BUILDER_MESSAGES,
  BUILDER_TEXTS,
  SECURITY_TIP,
  STEPS,
  TIPS
} from './constants'
import {
  fieldLayout,
  hasData,
  previewBadges,
  previewTitle,
  progressChips,
  reviewRows
} from './form-helpers'
import { builderReducer, initialState } from './reducer'
import { emptyValues, parseTicket, ticketSchemas } from './schemas'
import { clock, sendTicket } from './send'
import type {
  BuilderState,
  FieldSlot,
  SendTiming,
  TicketFormValues
} from './types'

const tipsFor = ({ step, type }: BuilderState): readonly string[] =>
  step === 'confirmacao'
    ? TIPS.confirmacao
    : [
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
 * memory, in one react-hook-form form validated by the schema of the chosen
 * type (BR-7, S-2, S-10). On each step change the focus moves to the step
 * heading and the change is announced.
 */
export const TicketBuilder = () => {
  // Opt out of the React Compiler, like CadastroForm: react-hook-form keeps
  // its own subscriptions.
  'use no memo'

  const [state, dispatch] = useReducer(builderReducer, initialState)
  const headingId = useId()
  const formId = useId()
  const headingRef = useRef<HTMLHeadingElement>(null)
  const previousStep = useRef(state.step)
  const typeRef = useRef<null | TicketType>(state.type)

  useEffect(() => {
    typeRef.current = state.type
  }, [state.type])

  // Validated with the schema of the type chosen when the check runs.
  const resolver: Resolver<TicketFormValues> = (values, context, options) => {
    const type = typeRef.current
    if (type === null) return { errors: {}, values }
    const resolve = zodResolver(
      ticketSchemas[type]
    ) as unknown as Resolver<TicketFormValues>
    return resolve(values, context, options)
  }

  const form = useForm<TicketFormValues>({
    defaultValues: {},
    mode: 'onTouched',
    resolver
  })
  const values = useWatch({ control: form.control }) as TicketFormValues
  const { errors } = form.formState
  // What will be sent: the values as the schema leaves them (trimmed).
  const reviewInput =
    state.step === 'revisao' && state.type
      ? parseTicket(state.type, values)
      : null

  const done = state.step === 'confirmacao'
  // On the confirmation, every step of the indicator is done.
  const stepIndex = done
    ? STEPS.length
    : STEPS.findIndex(step => step.id === state.step)
  const stepLabel = STEPS[stepIndex]?.label ?? ''

  // AC-5.1: a second click while the request is pending sends nothing.
  const sendOnce = useRef(singleFlight(sendTicket)).current

  // The send progress (AC-5.5 to AC-5.10): the real time since the click,
  // and the names (never the values) of what is being sent.
  const [timing, setTiming] = useState<null | SendTiming>(null)
  const [now, setNow] = useState(0)
  const [chips, setChips] = useState<readonly string[]>([])

  useEffect(() => {
    if (!state.sending) return
    const timer = window.setInterval(() => setNow(clock()), 250)
    return () => window.clearInterval(timer)
  }, [state.sending])

  const send = async () => {
    if (!reviewInput || state.sending) return
    const start = clock()
    setTiming({ end: null, start })
    setNow(start)
    setChips(progressChips(reviewInput))
    // The request starts now; the progress only draws what happens.
    dispatch({ type: 'sendStarted' })
    const outcome = await sendOnce(api.createTicket, reviewInput)
    if (outcome) {
      setTiming({ end: clock(), start })
      dispatch(outcome)
    }
  }

  const restart = () => {
    form.reset({})
    setTiming(null)
    dispatch({ type: 'restart' })
  }

  const elapsedMs = timing ? (timing.end ?? now) - timing.start : 0
  const phase: null | SendPhase = state.sending
    ? 'registering'
    : state.sendError
      ? 'failed'
      : done
        ? 'done'
        : null

  // One polite announcement per outcome (AC-5.10): the error is said by
  // the inline alert, so this region keeps "Enviando ticket." meanwhile.
  const announcement =
    state.sending || state.sendError
      ? BUILDER_MESSAGES.sending
      : done && state.ticket
        ? BUILDER_MESSAGES.created(state.ticket.id)
        : `Etapa ${stepIndex + 1} de ${STEPS.length}: ${stepLabel}`

  useEffect(() => {
    if (previousStep.current !== state.step) {
      previousStep.current = state.step
      headingRef.current?.focus()
    }
  }, [state.step])

  const selectType = (ticketType: TicketType) => {
    const data = hasData(form.getValues())
    // A new type starts with its own empty fields, unless the change waits
    // for the inline confirmation (AC-2.3, AC-2.4).
    if (ticketType !== state.type && (state.type === null || !data)) {
      form.reset(emptyValues(ticketType))
    }
    dispatch({ hasData: data, ticketType, type: 'selectType' })
  }

  const confirmTypeChange = () => {
    if (state.pendingType) form.reset(emptyValues(state.pendingType))
    dispatch({ type: 'confirmTypeChange' })
  }

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

  // AC-3.2 to AC-3.6: an invalid form stays, shows each message under its
  // field and focuses the first invalid field (react-hook-form, in screen
  // order); a valid one moves on.
  const submitDetails = form.handleSubmit(() =>
    dispatch({ type: 'continueFromDetails' })
  )

  const renderField = ({ field, wide }: FieldSlot) => {
    const id = `${formId}-${field.key}`
    const error = errors[field.key]?.message
    return (
      <FormField
        className={cn(wide && 'md:col-span-2')}
        error={typeof error === 'string' ? error : undefined}
        hint={field.kind === 'url' ? field.hint : undefined}
        id={id}
        key={field.key}
        label={field.label}
        required={field.required}
      >
        {control =>
          field.kind === 'choice' ? (
            <Controller
              render={({ field: choice }) => (
                <ChoiceSelect
                  {...control}
                  labelledBy={fieldIds(id).label}
                  name={choice.name}
                  onBlur={choice.onBlur}
                  onValueChange={choice.onChange}
                  options={field.options ?? []}
                  placeholder={field.placeholder ?? ''}
                  ref={choice.ref}
                  value={choice.value ?? ''}
                />
              )}
              control={form.control}
              name={field.key}
            />
          ) : field.kind === 'long' || field.kind === 'optional' ? (
            <TextAreaField
              {...control}
              {...form.register(field.key)}
              count={codePoints(values[field.key] ?? '')}
              hint={field.hint}
              max={FIELD_LIMITS[field.kind].max}
              placeholder={field.placeholder}
            />
          ) : (
            <FormInput
              {...control}
              {...form.register(field.key)}
              autoComplete="off"
              inputMode={field.kind === 'url' ? 'url' : undefined}
              placeholder={field.placeholder}
              spellCheck={field.kind === 'url' ? false : undefined}
              type="text"
            />
          )
        }
      </FormField>
    )
  }

  const heading = (text: string) => (
    <h2
      className="text-base font-semibold outline-none"
      id={headingId}
      ref={headingRef}
      tabIndex={-1}
    >
      {text}
    </h2>
  )

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
        {done ? null : (
          <Link
            // `cn` lets the outline border win over the base `border-transparent`.
            className={cn(buttonVariants({ size: 'sm', variant: 'outline' }))}
            href="/usuarios"
          >
            <XIcon aria-hidden />
            {BUILDER_TEXTS.cancel}
          </Link>
        )}
      </div>

      <StepIndicator
        onSelect={index =>
          dispatch({ step: STEPS[index].id, type: 'goToStep' })
        }
        current={stepIndex}
        locked={done || state.sending}
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
                {heading('Qual é o tipo do pedido?')}
                <p className="text-muted-foreground text-sm">
                  Cada tipo tem um formulário com os campos que a equipe
                  precisa.
                </p>
              </div>
              <TicketTypePicker
                error={state.typeError ? BUILDER_MESSAGES.noType : undefined}
                labelledBy={headingId}
                onValueChange={selectType}
                options={TICKET_TYPES}
                value={state.type}
              />
              {state.pendingType && state.type ? (
                <div
                  className="border-border bg-muted flex flex-col gap-3 rounded-sm border p-4 sm:flex-row sm:items-center sm:justify-between"
                  data-testid="type-change"
                  role="alert"
                >
                  <p className="text-sm">
                    {BUILDER_MESSAGES.typeChange(
                      ticketTypeLabel(state.pendingType),
                      ticketTypeLabel(state.type)
                    )}
                  </p>
                  <div className="flex shrink-0 flex-col-reverse gap-2 sm:flex-row">
                    <Button
                      className="w-full sm:w-auto"
                      onClick={() => dispatch({ type: 'cancelTypeChange' })}
                      size="sm"
                      type="button"
                      variant="outline"
                    >
                      {BUILDER_TEXTS.keepType}
                    </Button>
                    <Button
                      className="w-full sm:w-auto"
                      onClick={confirmTypeChange}
                      size="sm"
                      type="button"
                    >
                      {BUILDER_TEXTS.confirmTypeChange}
                    </Button>
                  </div>
                </div>
              ) : null}
            </>
          ) : state.step === 'detalhes' && state.type ? (
            <>
              <div className="flex flex-col gap-2">
                {heading(stepLabel)}
                <p className="text-muted-foreground text-sm">
                  {BUILDER_TEXTS.detailsIntro(ticketTypeLabel(state.type))}
                </p>
                <p className="text-muted-foreground flex items-start gap-2 text-xs">
                  <ClockIcon
                    className="size-3.5 shrink-0 translate-y-px"
                    aria-hidden
                  />
                  {BUILDER_TEXTS.sessionNote}
                </p>
              </div>
              <form
                className="grid gap-5 md:grid-cols-2"
                id={formId}
                onSubmit={submitDetails}
                noValidate
              >
                {fieldLayout(state.type).map(renderField)}
              </form>
            </>
          ) : state.step === 'revisao' && state.type && reviewInput ? (
            <>
              <div className="flex flex-col gap-1">
                {heading(stepLabel)}
                <p className="text-muted-foreground text-sm">
                  {BUILDER_TEXTS.reviewIntro}
                </p>
              </div>
              <TicketReview
                rows={reviewRows(reviewInput)}
                title={ticketTitle(reviewInput)}
                type={state.type}
                typeLabel={ticketTypeLabel(state.type)}
              />
              {phase ? (
                <SendProgress
                  chips={chips}
                  elapsedMs={elapsedMs}
                  phase={phase}
                />
              ) : null}
              {state.sendError ? (
                <InlineAlert>{state.sendError}</InlineAlert>
              ) : null}
            </>
          ) : done && state.ticket ? (
            <>
              {/* The rows already on screen stay; the last step and the
                  check row enter here. */}
              <SendProgress
                chips={chips}
                elapsedMs={elapsedMs}
                enterFrom={3}
                phase="done"
              />
              <TicketConfirmation
                headingId={headingId}
                headingRef={headingRef}
                ticket={state.ticket}
              />
            </>
          ) : (
            heading(stepLabel)
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
          ) : state.step === 'detalhes' ? (
            <>
              <Button
                className="w-full sm:w-auto"
                onClick={() => dispatch({ step: 'tipo', type: 'goToStep' })}
                type="button"
                variant="outline"
              >
                Voltar
              </Button>
              <Button
                className="w-full data-[inactive=true]:opacity-50 sm:w-auto"
                data-inactive={!hasData(values)}
                form={formId}
                type="submit"
              >
                Continuar
                <ArrowRightIcon data-icon="inline-end" aria-hidden />
              </Button>
            </>
          ) : state.step === 'revisao' ? (
            <>
              {/* AC-4.2: back to the form, every value kept. */}
              <Button
                className="w-full sm:w-auto"
                disabled={state.sending}
                onClick={() => dispatch({ step: 'detalhes', type: 'goToStep' })}
                type="button"
                variant="outline"
              >
                <PencilIcon data-icon="inline-start" aria-hidden />
                Editar
              </Button>
              {/* AC-5.1: shows that it is sending; a second click does
                  nothing. `aria-disabled` keeps the focus on it. */}
              <Button
                aria-disabled={state.sending || undefined}
                className="w-full aria-disabled:opacity-70 sm:w-auto"
                onClick={send}
                type="button"
              >
                {state.sending ? (
                  <>
                    <LoaderCircleIcon
                      className="motion-safe:animate-spin"
                      data-icon="inline-start"
                      aria-hidden
                    />
                    {BUILDER_TEXTS.sending}
                  </>
                ) : (
                  <>
                    {BUILDER_TEXTS.send}
                    <SendIcon data-icon="inline-end" aria-hidden />
                  </>
                )}
              </Button>
            </>
          ) : (
            <>
              <Link
                className={cn(
                  buttonVariants({ variant: 'outline' }),
                  'w-full sm:w-auto'
                )}
                href="/usuarios"
              >
                {BUILDER_TEXTS.backToUsers}
              </Link>
              <Button
                className="w-full sm:w-auto"
                onClick={restart}
                type="button"
              >
                <PlusIcon data-icon="inline-start" aria-hidden />
                {BUILDER_TEXTS.createAnother}
              </Button>
            </>
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
          {done ? null : (
            <SideCard icon={EyeIcon} title="Prévia">
              <TicketPreview
                badges={state.type ? previewBadges(state.type, values) : []}
                title={state.type ? previewTitle(state.type, values) : null}
                type={state.type}
                typeLabel={state.type ? ticketTypeLabel(state.type) : null}
              />
            </SideCard>
          )}
        </aside>
      </div>

      <p aria-live="polite" className="sr-only" role="status">
        {announcement}
      </p>
    </main>
  )
}
