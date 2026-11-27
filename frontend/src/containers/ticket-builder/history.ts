import type {
  HistoryEntry,
  HistoryWindow,
  PopDecision,
  Step,
  StepHistory
} from './types'

export const STEP_PARAM = 'etapa'

export const HISTORY_KEY = 'ticketBuilder'

const STEPS_IN_URL = ['detalhes', 'revisao', 'confirmacao'] as const

export const searchFor = (step: Step): string =>
  step === 'tipo' ? '' : `?${STEP_PARAM}=${step}`

export const stepFromSearch = (search: string): Step => {
  const value = new URLSearchParams(search).get(STEP_PARAM)
  return STEPS_IN_URL.find(step => step === value) ?? 'tipo'
}

export const guardActive = ({
  dirty,
  sent
}: {
  dirty: boolean
  sent: boolean
}): boolean => dirty && !sent

export const entryIndex = (state: unknown, visit: string): null | number => {
  if (typeof state !== 'object' || state === null) return null
  const entry = (state as Record<string, unknown>)[HISTORY_KEY] as
    Partial<HistoryEntry> | undefined
  return entry?.visit === visit && Number.isInteger(entry.index)
    ? (entry.index as number)
    : null
}

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

export const createStepHistory = (
  win: HistoryWindow,
  newVisit: () => string
): StepHistory => {
  const path = win.location.pathname
  let visit = newVisit()
  let stack: Step[] = ['tipo']
  let cursor = 0
  const own: (() => void)[] = []

  // A URL passed to pushState/replaceState makes Next.js drop a navigation in flight.
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
      if (cursor === 0) reset('confirmacao')
      else travel(-cursor, () => reset('confirmacao'))
    },
    start: () => reset('tipo')
  }
}

export {
  isPlainClick,
  leavesPage as leavesBuilder
} from '@/utils/navigation-guard'
