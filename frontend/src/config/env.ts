import { z } from 'zod'

const envSchema = z.object({
  nodeEnv: z.union([z.literal('development'), z.literal('production')]),
  baseUrl: z.string().trim().default('http://localhost:3000')
})

export const env = envSchema.parse({
  nodeEnv: process.env.NODE_ENV,
  baseUrl: process.env.NEXT_PUBLIC_BASE_URL
})
