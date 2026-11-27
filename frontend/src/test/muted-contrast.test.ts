import { describe, expect, it } from 'vitest'

import { contrastRatio, themeToken } from './contrast'

describe('secondary text contrast (spec 003 section 7)', () => {
  it.each(['background', 'muted'])('meets AA on %s', background => {
    expect(
      contrastRatio(themeToken('muted-foreground'), themeToken(background))
    ).toBeGreaterThanOrEqual(4.5)
  })
})
