import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it, vi } from 'vitest'

import { LeaveGuardBar, installLeaveGuard } from '.'

vi.mock('next/navigation', () => ({ useRouter: () => ({ push: () => {} }) }))

const fakeWindow = () => {
  const listeners = new Map<string, EventListener>()
  return {
    addEventListener: vi.fn((type: string, listener: EventListener) => {
      listeners.set(type, listener)
    }),
    listeners,
    removeEventListener: vi.fn((type: string) => {
      listeners.delete(type)
    })
  }
}

const options = (logoutRunning = false) => ({
  current: () => 'https://localhost:3000/usuarios/tickets/7',
  logoutAllowed: { current: false },
  logoutRunning: () => logoutRunning,
  onRequest: vi.fn()
})

describe('installLeaveGuard', () => {
  it('listens for beforeunload and for clicks in the capture phase', () => {
    const win = fakeWindow()
    installLeaveGuard(win as never, options())
    expect([...win.listeners.keys()]).toEqual(['beforeunload', 'click'])
    expect(win.addEventListener).toHaveBeenCalledWith(
      'click',
      expect.any(Function),
      true
    )
  })

  it('removes both listeners on cleanup (unmount, or the text was sent)', () => {
    const win = fakeWindow()
    const cleanup = installLeaveGuard(win as never, options())
    cleanup()
    expect(win.listeners.size).toBe(0)
    expect(win.removeEventListener).toHaveBeenCalledWith(
      'click',
      expect.any(Function),
      true
    )
  })

  it('asks the browser to warn on unload, unless a confirmed logout runs', () => {
    for (const [running, warned] of [
      [false, true],
      [true, false]
    ]) {
      const win = fakeWindow()
      installLeaveGuard(win as never, options(running))
      const event = { preventDefault: vi.fn() }
      win.listeners.get('beforeunload')?.(event as never)
      expect(event.preventDefault).toHaveBeenCalledTimes(warned ? 1 : 0)
    }
  })
})

describe('LeaveGuardBar', () => {
  const props = {
    body: 'A mensagem será perdida.',
    leaveLabel: 'Sair sem enviar',
    onLeave: () => {},
    onStay: () => {},
    stayLabel: 'Continuar',
    stayRef: { current: null },
    title: 'Sair sem enviar?'
  }

  it('shows the inline alert with both actions only while open', () => {
    const html = renderToStaticMarkup(<LeaveGuardBar {...props} open />)
    expect(html).toContain('role="alert"')
    expect(html).toContain(
      '<span class="font-medium">Sair sem enviar?</span> A mensagem será perdida.'
    )
    expect(html).toContain('>Continuar</button>')
    expect(html).toContain('>Sair sem enviar</button>')
    expect(
      renderToStaticMarkup(<LeaveGuardBar {...props} open={false} />)
    ).toBe('')
  })
})
