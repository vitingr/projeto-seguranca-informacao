import { z } from 'zod'

// Login only checks that both fields are filled: the backend answers 401
// for anything that does not authenticate (002 D-12), so the login and
// password rules apply only when creating users. The password is never
// trimmed or changed.
export const loginSchema = z.object({
  login: z.string().trim().min(1, 'Informe o login.'),
  senha: z.string().min(1, 'Informe a senha.')
})
