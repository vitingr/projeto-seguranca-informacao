import { z } from 'zod'

const envSchema = z.object({
  nodeEnv: z.union([z.literal('development'), z.literal('production')]),
  baseUrl: z.string().trim().default('https://localhost:3000'),
  apiBaseUrl: z.string().trim()
})

export const env = envSchema.parse({
  nodeEnv: process.env.NODE_ENV,
  baseUrl: process.env.NEXT_PUBLIC_BASE_URL,
  apiBaseUrl: process.env.API_BASE_URL
})
