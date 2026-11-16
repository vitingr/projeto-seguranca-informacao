import { existsSync, readFileSync } from 'node:fs'
import path from 'node:path'
import { parseEnv } from 'node:util'

// Values come from the process environment first, then backend/.env (the
// same precedence as the backend's mage targets). Never from arguments.
const backendEnvFile = path.join(__dirname, '..', '..', '..', 'backend', '.env')

let fileValues: Record<string, string | undefined> | undefined

function fromFile(): Record<string, string | undefined> {
  if (!fileValues) {
    fileValues = existsSync(backendEnvFile)
      ? parseEnv(readFileSync(backendEnvFile, 'utf8'))
      : {}
  }
  return fileValues
}

/** Reads a required variable; the error names it but never shows a value. */
export function requireEnv(name: string): string {
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
