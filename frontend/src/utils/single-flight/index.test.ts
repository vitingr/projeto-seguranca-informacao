import { describe, expect, it, vi } from 'vitest'

import { singleFlight } from '.'

const deferred = () => {
  let resolve: (value: string) => void = () => {}
  const promise = new Promise<string>(done => {
    resolve = done
  })
  return { promise, resolve }
}

describe('singleFlight', () => {
  it('starts nothing while a call is pending', async () => {
    const gate = deferred()
    const call = vi.fn(() => gate.promise)
    const once = singleFlight(call)

    const first = once()
    const second = once()
    expect(call).toHaveBeenCalledTimes(1)
    expect(await second).toBeNull()

    gate.resolve('ok')
    expect(await first).toBe('ok')
  })

  it('allows a new call after the previous one settled, even on error', async () => {
    const call = vi
      .fn<() => Promise<string>>()
      .mockRejectedValueOnce(new Error('rede'))
      .mockResolvedValueOnce('ok')
    const once = singleFlight(call)
    await expect(once()).rejects.toThrow('rede')
    expect(await once()).toBe('ok')
    expect(call).toHaveBeenCalledTimes(2)
  })
})
