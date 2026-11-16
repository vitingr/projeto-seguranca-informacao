import { fileURLToPath } from 'node:url'
import { defineConfig } from 'vitest/config'

const fromRoot = (path: string) => fileURLToPath(new URL(path, import.meta.url))

export default defineConfig({
  resolve: {
    alias: {
      '@': fromRoot('./src'),
      // Next aliases `server-only` itself; outside Next it must resolve to
      // the same empty module so server code can be unit-tested.
      'server-only': fromRoot(
        './node_modules/next/dist/compiled/server-only/empty.js'
      )
    }
  },
  test: {
    environment: 'node',
    include: ['src/**/*.test.ts', 'scripts/**/*.test.mjs']
  }
})
