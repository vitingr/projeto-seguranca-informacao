import { describe, expect, it } from 'vitest'

import {
  HISTORY_KEY,
  createStepHistory,
  decidePop,
  entryIndex,
  guardActive,
  isPlainClick,
  leavesBuilder,
  searchFor,
  stepFromSearch
} from './history'
import type { StepHistory } from './types'

const BUILDER = '/usuarios/tickets/novo'

const fakeBrowser = () => {
  const entries: { state: unknown; url: string }[] = [
    { state: null, url: '/usuarios' },
    { state: { __NA: true }, url: BUILDER }
  ]
  let at = 1
  let pathname = BUILDER
  const pops: number[] = []
  const urlOf = (url?: null | string | URL) =>
    url == null ? entries[at].url : String(url)
  const win = {
    history: {
      go: (delta?: number) => {
        pops.push(delta ?? 0)
      },
      pushState: (
        state: unknown,
        _unused: string,
        url?: null | string | URL
      ) => {
        const next = urlOf(url)
        entries.splice(at + 1)
        entries.push({ state, url: next })
        at += 1
      },
      replaceState: (
        state: unknown,
        _unused: string,
        url?: null | string | URL
      ) => {
        entries[at] = { state, url: urlOf(url) }
      }
    },
    location: {
      get pathname() {
        return pathname
      },
      get search() {
        return new URL(entries[at].url, 'https://x').search
      }
    }
  }
  let visits = 0
  const history = createStepHistory(win, () => `visit-${++visits}`)
  const shown: (null | string)[] = []
  const move = (delta: number, locked = false) => {
    at += delta
    pathname = new URL(entries[at].url, 'https://x').pathname
    shown.push(history.pop(entries[at].state, locked))
  }
  const flush = (locked = false) => {
    while (pops.length > 0) move(pops.shift() as number, locked)
  }
  return {
    entries,
    flush,
    history,
    move,
    shown,
    url: () => entries[at].url
  }
}

const reachReview = (history: StepHistory) => {
  history.start()
  history.push('detalhes')
  history.push('revisao')
}

describe('step ↔ ?etapa=', () => {
  it('maps every step, the first one to the bare URL', () => {
    expect(searchFor('tipo')).toBe('')
    expect(searchFor('detalhes')).toBe('?etapa=detalhes')
    expect(searchFor('revisao')).toBe('?etapa=revisao')
    expect(searchFor('confirmacao')).toBe('?etapa=confirmacao')
    for (const step of ['detalhes', 'revisao', 'confirmacao'] as const) {
      expect(stepFromSearch(searchFor(step))).toBe(step)
    }
  })

  it.each([
    '',
    '?etapa=',
    '?etapa=tipo',
    '?etapa=REVISAO',
    '?etapa=x',
    '?outro=1'
  ])('falls back to the first step for %j', search => {
    expect(stepFromSearch(search)).toBe('tipo')
  })
})

describe('guardActive', () => {
  it('is on only while something typed was not sent (AC-3.8)', () => {
    expect(guardActive({ dirty: true, sent: false })).toBe(true)
    expect(guardActive({ dirty: false, sent: false })).toBe(false)
    expect(guardActive({ dirty: true, sent: true })).toBe(false)
    expect(guardActive({ dirty: false, sent: true })).toBe(false)
  })
})

describe('entryIndex', () => {
  it('reads only entries of the same visit', () => {
    const state = { [HISTORY_KEY]: { index: 2, visit: 'a' }, __NA: true }
    expect(entryIndex(state, 'a')).toBe(2)
    expect(entryIndex(state, 'b')).toBeNull()
    expect(entryIndex(null, 'a')).toBeNull()
    expect(entryIndex({ __NA: true }, 'a')).toBeNull()
    expect(
      entryIndex({ [HISTORY_KEY]: { index: '1', visit: 'a' } }, 'a')
    ).toBeNull()
  })
})

describe('decidePop', () => {
  it('shows an earlier entry of this visit', () => {
    expect(decidePop({ cursor: 2, index: 1, locked: false })).toEqual({
      kind: 'apply'
    })
  })

  it('undoes a forward move and any move while locked', () => {
    expect(decidePop({ cursor: 0, index: 1, locked: false })).toEqual({
      delta: -1,
      kind: 'undo'
    })
    expect(decidePop({ cursor: 2, index: 1, locked: true })).toEqual({
      delta: 1,
      kind: 'undo'
    })
  })

  it('keeps the step on a foreign entry and ignores the same entry', () => {
    expect(decidePop({ cursor: 1, index: null, locked: false })).toEqual({
      kind: 'adopt'
    })
    expect(decidePop({ cursor: 1, index: 1, locked: false })).toEqual({
      kind: 'none'
    })
  })
})

describe('createStepHistory', () => {
  it('puts each step reached in the URL, and only the step (S-10)', () => {
    const browser = fakeBrowser()
    browser.history.start()
    expect(browser.url()).toBe(BUILDER)
    browser.history.push('detalhes')
    expect(browser.url()).toBe(`${BUILDER}?etapa=detalhes`)
    browser.history.push('revisao')
    expect(browser.url()).toBe(`${BUILDER}?etapa=revisao`)
    expect(browser.entries).toHaveLength(4)
  })

  it('Back moves one step at a time, then leaves the builder', () => {
    const browser = fakeBrowser()
    reachReview(browser.history)
    browser.move(-1)
    browser.move(-1)
    expect(browser.shown).toEqual(['detalhes', 'tipo'])
    expect(browser.history.current()).toBe('tipo')
    browser.move(-1)
    expect(browser.shown.at(-1)).toBeNull()
  })

  it('"Voltar" and "Editar" go back through the history', () => {
    const browser = fakeBrowser()
    reachReview(browser.history)
    expect(browser.history.back('detalhes')).toBe(true)
    browser.flush()
    expect(browser.shown).toEqual(['detalhes'])
    expect(browser.url()).toBe(`${BUILDER}?etapa=detalhes`)
    expect(browser.history.back('tipo')).toBe(true)
    browser.flush()
    expect(browser.shown).toEqual(['detalhes', 'tipo'])
    expect(browser.history.back('tipo')).toBe(false)
  })

  it('Forward never skips to a step that was not continued again', () => {
    const browser = fakeBrowser()
    reachReview(browser.history)
    browser.move(-2)
    expect(browser.shown).toEqual(['tipo'])
    browser.move(1)
    browser.flush()
    expect(browser.shown).toEqual(['tipo', null, null])
    expect(browser.history.current()).toBe('tipo')
    expect(browser.url()).toBe(BUILDER)
    browser.history.push('detalhes')
    expect(browser.entries).toHaveLength(3)
  })

  it('nothing moves while sending (AC-5.1)', () => {
    const browser = fakeBrowser()
    reachReview(browser.history)
    browser.move(-1, true)
    browser.flush(true)
    expect(browser.shown).toEqual([null, null])
    expect(browser.url()).toBe(`${BUILDER}?etapa=revisao`)
  })

  it('after sending, the first entry is the confirmation and Back leaves (AC-5.3)', () => {
    const browser = fakeBrowser()
    reachReview(browser.history)
    browser.history.sent()
    browser.flush()
    expect(browser.url()).toBe(`${BUILDER}?etapa=confirmacao`)
    expect(browser.history.current()).toBe('confirmacao')
    browser.move(1)
    browser.move(1)
    expect(browser.shown.filter(Boolean)).toEqual([])
    expect(browser.url()).toBe(`${BUILDER}?etapa=confirmacao`)
    browser.move(-2)
    browser.move(-1)
    expect(browser.url()).toBe('/usuarios')
  })

  it('"Criar outro ticket" starts a new visit on the current entry', () => {
    const browser = fakeBrowser()
    reachReview(browser.history)
    browser.history.sent()
    browser.flush()
    browser.history.restart()
    expect(browser.url()).toBe(BUILDER)
    expect(browser.history.current()).toBe('tipo')
    browser.history.push('detalhes')
    browser.move(-1)
    expect(browser.shown.at(-1)).toBe('tipo')
  })

  it('an entry from before a reload keeps the current step', () => {
    const browser = fakeBrowser()
    reachReview(browser.history)
    const again = createStepHistory(
      {
        history: {
          go: () => {},
          pushState: () => {},
          replaceState: (state, _unused, url) => {
            browser.entries[3] = { state, url: String(url) }
          }
        },
        location: { pathname: BUILDER, search: '?etapa=revisao' }
      },
      () => 'after-reload'
    )
    again.start()
    expect(browser.entries[3].url).toBe(BUILDER)
    expect(again.pop(browser.entries[2].state, false)).toBeNull()
    expect(again.current()).toBe('tipo')
  })
})

describe('createStepHistory passes the URL only when it changes', () => {
  const nextLikeWindow = (search: string) => {
    const calls: {
      method: 'pushState' | 'replaceState'
      state: unknown
      url: unknown
    }[] = []
    const router = { pending: '/usuarios' as null | string }
    const record =
      (method: 'pushState' | 'replaceState') =>
      (state: unknown, _unused: string, url?: null | string | URL) => {
        calls.push({ method, state, url })
        if (url != null) router.pending = null
      }
    const win = {
      history: {
        go: () => {},
        pushState: record('pushState'),
        replaceState: record('replaceState')
      },
      location: { pathname: BUILDER, search }
    }
    return { calls, router, win }
  }

  it('on the bare builder URL, start writes only the state, so a link clicked meanwhile still leaves', () => {
    const { calls, router, win } = nextLikeWindow('')
    createStepHistory(win, () => 'visit').start()
    expect(calls).toEqual([
      {
        method: 'replaceState',
        state: { [HISTORY_KEY]: { index: 0, visit: 'visit' } },
        url: undefined
      }
    ])
    expect(router.pending).toBe('/usuarios')
  })

  it('a step that changes the URL still passes it', () => {
    const { calls, win } = nextLikeWindow('')
    const history = createStepHistory(win, () => 'visit')
    history.start()
    history.push('detalhes')
    expect(calls.at(-1)).toEqual({
      method: 'pushState',
      state: { [HISTORY_KEY]: { index: 1, visit: 'visit' } },
      url: `${BUILDER}?etapa=detalhes`
    })
  })

  it('a reload on a later step still goes back to the bare URL (BR-7)', () => {
    const { calls, win } = nextLikeWindow('?etapa=revisao')
    createStepHistory(win, () => 'visit').start()
    expect(calls).toEqual([
      {
        method: 'replaceState',
        state: { [HISTORY_KEY]: { index: 0, visit: 'visit' } },
        url: BUILDER
      }
    ])
  })
})

describe('isPlainClick', () => {
  const click = {
    altKey: false,
    button: 0,
    ctrlKey: false,
    metaKey: false,
    shiftKey: false
  }

  it('is a main-button click with no modifier', () => {
    expect(isPlainClick(click)).toBe(true)
    expect(isPlainClick({ ...click, button: 1 })).toBe(false)
    expect(isPlainClick({ ...click, ctrlKey: true })).toBe(false)
    expect(isPlainClick({ ...click, metaKey: true })).toBe(false)
    expect(isPlainClick({ ...click, shiftKey: true })).toBe(false)
    expect(isPlainClick({ ...click, altKey: true })).toBe(false)
  })
})

describe('leavesBuilder', () => {
  const here = `https://localhost:3000${BUILDER}?etapa=detalhes`

  it('is true for another page of this site', () => {
    expect(leavesBuilder('/usuarios', here)).toBe(true)
    expect(leavesBuilder('https://localhost:3000/', here)).toBe(true)
  })

  it('is false for the builder itself and for other sites', () => {
    expect(leavesBuilder(`${BUILDER}?etapa=revisao`, here)).toBe(false)
    expect(leavesBuilder('#topo', here)).toBe(false)
    expect(leavesBuilder('https://example.com/usuarios', here)).toBe(false)
  })
})
