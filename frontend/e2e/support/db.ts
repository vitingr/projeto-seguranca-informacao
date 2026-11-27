import { execFileSync } from 'node:child_process'
import path from 'node:path'

import { requireEnv } from './env'

const backendDir = path.join(__dirname, '..', '..', '..', 'backend')

// Values are psql variables, quoted by psql, never concatenated into the SQL
export const setRole = (login: string, role: 'admin' | 'comum') => {
  const output = execFileSync(
    'docker',
    [
      'compose',
      'exec',
      '-T',
      'db',
      'psql',
      '-X',
      '-q',
      '-v',
      'ON_ERROR_STOP=1',
      '-U',
      requireEnv('POSTGRES_USER'),
      '-d',
      requireEnv('POSTGRES_DB'),
      '-v',
      `login=${login}`,
      '-v',
      `role=${role}`
    ],
    {
      cwd: backendDir,
      encoding: 'utf8',
      input:
        "UPDATE users SET role = :'role' WHERE login = :'login' RETURNING role;\n"
    }
  )
  if (!output.includes(role)) {
    throw new Error(`setRole: user not updated`)
  }
}
