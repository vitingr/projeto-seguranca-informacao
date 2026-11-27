import type {
  HistoryEntry,
  HistoryWindow,
  PopDecision,
  Step,
  StepHistory
} from './types'

// The builder's steps in the browser history (plan 5.3, AC-3.8 and AC-5.3):
// each step reached by continuing is a history entry with `?etapa=`, so
// Back moves one step and never sends. Only the step goes into the URL,
// never a typed value (S-10). The typed values live in the open page only
// (BR-7), so a step that needs them cannot be restored after a reload.

export const STEP_PARAM = 'etapa'

/** The key of the builder's data inside `history.state`. */
export const HISTORY_KEY = 'ticketBuilder'

const STEPS_IN_URL = ['detalhes', 'revisao', 'confirmacao'] as const

/** The query of a step; the first step is the bare builder URL. */
export const searchFor = (step: Step): string =>
  step === 'tipo' ? '' : `?${STEP_PARAM}=${step}`

/** The step a query names; anything else is the first step. */
export const stepFromSearch = (search: string): Step => {
  const value = new URLSearchParams(search).get(STEP_PARAM)
  return STEPS_IN_URL.find(step => step === value) ?? 'tipo'
}

/** The leave warning only matters while something typed was not sent. */
export const guardActive = ({
  dirty,
  sent
}: {
  dirty: boolean
  sent: boolean
}): boolean => dirty && !sent

/** The index of an entry written by this visit, or `null` for any other. */
export const entryIndex = (state: unknown, visit: string): null | number => {
  if (typeof state !== 'object' || state === null) return null
  const entry = (state as Record<string, unknown>)[HISTORY_KEY] as
    Partial<HistoryEntry> | undefined
  return entry?.visit === visit && Number.isInteger(entry.index)
    ? (entry.index as number)
    : null
}

/**
 * What a `popstate` does. Back to an earlier entry of this visit shows its
 * step. Moving forward is undone: a later step is only reached through its
 * button, which checks the form. While sending or after the ticket was
 * created nothing moves (AC-5.1, AC-5.3). An entry of another visit (before
 * a reload or "Criar outro ticket") keeps the current step.
 */
export const decidePop = ({
  cursor,
  index,
  locked
}: {
  cursor: number
  index: null | number
  locked: boolean
}): PopDecision => {
  if (index === null) return { kind: 'adopt' }
  if (index === cursor) return { kind: 'none' }
  if (locked || index > cursor) return { delta: cursor - index, kind: 'undo' }
  return { kind: 'apply' }
}

/**
 * The step history of one builder page. `newVisit` gives a unique id for
 * the page and for each "Criar outro ticket", so entries of an earlier
 * ticket are never taken for this one's.
 */
export const createStepHistory = (
  win: HistoryWindow,
  newVisit: () => string
): StepHistory => {
  const path = win.location.pathname
  let visit = newVisit()
  let stack: Step[] = ['tipo']
  let cursor = 0
  // `popstate`s caused by this module, with what to do once each arrives.
  const own: (() => void)[] = []

  // The URL is passed only when it changes. Next.js syncs its router with
  // every `pushState`/`replaceState` that carries a URL, as a navigation of
  // its own, and that discards a navigation still in flight: a link clicked
  // just as the page started (before `start`) would then be lost, leaving
  // the user on the builder. With no URL the browser keeps the current one
  // and the router is not involved; only the entry's state is written.
  const write = (method: 'pushState' | 'replaceState', index: number) => {
    const entry: HistoryEntry = { index, visit }
    const state = { [HISTORY_KEY]: entry }
    const url = `${path}${searchFor(stack[index])}`
    const current = `${win.location.pathname}${win.location.search}`
    if (url === current) win.history[method](state, '')
    else win.history[method](state, '', url)
  }

  const reset = (step: Step) => {
    visit = newVisit()
    stack = [step]
    cursor = 0
    write('replaceState', 0)
  }

  const travel = (delta: number, then: () => void = () => {}) => {
    own.push(then)
    win.history.go(delta)
  }

  return {
    back: step => {
      if (cursor === 0) return false
      const index = stack.lastIndexOf(step, cursor - 1)
      if (index < 0) return false
      // The `popstate` that follows shows the step (`apply`).
      win.history.go(index - cursor)
      return true
    },
    current: () => stack[cursor],
    pop: (state, locked) => {
      if (win.location.pathname !== path) return null
      const then = own.shift()
      if (then) {
        then()
        return null
      }
      const index = entryIndex(state, visit)
      const decision = decidePop({ cursor, index, locked })
      switch (decision.kind) {
        case 'none':
          return null
        case 'undo':
          travel(decision.delta)
          return null
        case 'adopt':
          write('replaceState', cursor)
          return null
        case 'apply':
          cursor = index as number
          return stack[cursor]
      }
    },
    push: step => {
      stack = [...stack.slice(0, cursor + 1), step]
      cursor += 1
      write('pushState', cursor)
    },
    restart: () => reset('tipo'),
    sent: () => {
      // Back from the confirmation leaves the builder at once, instead of
      // walking through the steps of a ticket already sent.
      if (cursor === 0) reset('confirmacao')
      else travel(-cursor, () => reset('confirmacao'))
    },
    start: () => reset('tipo')
  }
}

// The click and link rules of the leave guard now live in
// utils/navigation-guard (shared with the reply box, plan P-12); the
// builder names stay, so history.test.ts keeps testing them.
export {
  isPlainClick,
  leavesPage as leavesBuilder
} from '@/utils/navigation-guard'
