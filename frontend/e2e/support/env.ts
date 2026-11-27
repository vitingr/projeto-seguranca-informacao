import { existsSync, readFileSync } from 'node:fs'
import path from 'node:path'
import { parseEnv } from 'node:util'

const backendEnvFile = path.join(__dirname, '..', '..', '..', 'backend', '.env')

let fileValues: Record<string, string | undefined> | undefined

const fromFile = (): Record<string, string | undefined> => {
  if (!fileValues) {
    fileValues = existsSync(backendEnvFile)
      ? parseEnv(readFileSync(backendEnvFile, 'utf8'))
      : {}
  }
  return fileValues
}

export const requireEnv = (name: string): string => {
  const value = process.env[name] || fromFile()[name]
  if (!value) {
    throw new Error(
      `${name} is not set: export it or fill it in backend/.env (and run "go tool mage seed")`
    )
  }
  return value
}

export const seedAdmin = () => ({
  login: requireEnv('SEED_ADMIN_LOGIN'),
  password: requireEnv('SEED_ADMIN_PASSWORD')
})

export const comumUser = () => ({
  login: requireEnv('E2E_COMUM_LOGIN'),
  password: requireEnv('E2E_COMUM_PASSWORD')
})

export const generateComumCredentials = (
  random: (bytes: number) => Buffer
): {
  login: string
  password: string
} => {
  return {
    login: `e2e.${random(12).toString('hex')}`,
    password: random(24).toString('base64url')
  }
}
