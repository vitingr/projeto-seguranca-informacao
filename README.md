# sistemas-da-informacao

Two projects in one repository:

- [`backend/`](backend/README.md): Go API (`backend_si`) with PostgreSQL,
  run through [mage](https://magefile.org) (`go tool mage`).
- [`frontend/`](frontend/README.md): Next.js app, run through pnpm.

The `Makefile` at the root runs both from here. It only delegates: every
target calls a script of `frontend/package.json` (`pnpm --dir frontend ...`)
or a mage target of `backend/` (`go -C backend tool mage ...`). Run `make`
to list the targets.

## Prerequisites

Everything the two projects need (see their READMEs), plus GNU Make:

| Tool | Used for |
|---|---|
| GNU Make 4.x | this `Makefile` |
| Go 1.27, golangci-lint 2.14.0 | backend targets |
| Node.js 24, pnpm 10.28 (`corepack enable`) | frontend targets, and `npx` for the contract and Bruno targets |
| Docker with Compose | `infra-up`, `infra-down`, `migrate`, `seed`, `e2e` |

**Windows:** run `make` from **Git Bash**. GNU Make for Windows only runs
the recipes with bash when it is started from a Git Bash shell; from
PowerShell or cmd it would use `cmd.exe`, so the `Makefile` stops with an
error there instead (it checks `MSYSTEM`).

## Getting started

```sh
make install                       # pnpm install + go mod download
cp backend/.env.example backend/.env
                                   # set POSTGRES_PASSWORD (openssl rand -hex 24)
                                   # and SEED_ADMIN_PASSWORD (15 to 128 characters)
printf 'API_BASE_URL=https://localhost:8443\n' > frontend/.env.local
make infra-up                      # certificate, PostgreSQL and the backend container
make migrate
make seed                          # first administrator: once per database
make dev                           # backend on :8443 and frontend on :3000
```

`make seed` fails when the administrator already exists, so run it once
after creating the database, not on every start. `make e2e` does not call
it.

### Frontend settings

`frontend/.env.local` holds `API_BASE_URL=https://localhost:8443`, the
backend that `make dev-backend` (`go tool mage run`) starts on the host.
Leave `API_URL` unset: the `/api` proxy then uses `API_BASE_URL` too.
`dev-frontend`, `build-frontend`, `typecheck-frontend` and `check` stop with
a message when `API_BASE_URL` is neither in the environment nor in
`frontend/.env.local`.

`make e2e` runs the Playwright flows against the docker compose stack
(`https://localhost`) instead: it sets `API_BASE_URL` and `API_URL` on the
command line, which wins over `.env.local`, and the Playwright web server
inherits them. Stop `make dev` first: outside CI, Playwright reuses a server
already listening on port 3000, which would still point at port 8443.

## Demonstração em aula

`make demo` runs only `frontend/e2e/enunciado.spec.ts`: one `describe` per
item of the assignment, in a visible Chromium, one test at a time and
slowed down (`slowMo` 400 ms). At the end the Playwright HTML report opens
in the browser; Ctrl+C stops its server.

Prerequisites on Windows: Docker Desktop (running), Git Bash, Go, Node.js,
pnpm, and the Playwright Chromium (`npx playwright install chromium` in
`frontend/`, or `make e2e-install`). `backend/.env` filled in as in
[Getting started](#getting-started).

In Git Bash, from the repository root:

```sh
make install         # once
make e2e-install     # once: Playwright Chromium
make infra-up        # certificate, PostgreSQL and the backend container
make migrate
make seed            # once per database: the first administrator
make demo            # starts the stack if needed, runs the demo, opens the report
```

Stop `make dev` first, as for `make e2e`. The tests create their own comum
users (`e2e.*` logins); the seed administrator is the only one they reuse.

| Item | What the test shows |
|---|---|
| 1. Página padrão | `/` redirects to `/login`, which has the login and password fields and the "Entrar" button |
| 2. Login do administrador | the admin lands on `/administradores` with the exact text and the "Sair" button |
| 3. Cadastro | `/cadastro` has login, password, user type ("Papel") and the button; the admin registers a comum user |
| 4. Login do usuário comum | the comum user lands on `/usuarios` with the exact text and the "Sair" button |
| 5. Sem login | `/cadastro`, `/administradores` and `/usuarios` go to `/login` |
| 6. Controle de acesso | comum gets "acesso negado" on `/administradores` and `/cadastro`, admin on `/usuarios`; direct calls to the other role's APIs answer 403 |
| 7. Injeção | `' OR '1'='1` (and `admin' --`) in the login does not get in and shows only the generic message; `<script>` in the registration is refused and never runs |
| 8. CSRF | a POST without the token answers 403 |
| 9. Sessão | `__Host-session` is HttpOnly, Secure and SameSite; its value changes at login; after logout the restricted pages go back to `/login` |
| 10. Erros | an unknown API route answers only `{error, message, request_id}` |

## Targets

| Target | Runs |
|---|---|
| `help` (default) | lists the targets |
| `install` | `install-frontend` (`pnpm install`) and `install-backend` (`go mod download`) |
| `e2e-install` | `pnpm e2e:install` (Chromium for Playwright, once) |
| `dev` | `dev-backend` and `dev-frontend` together; Ctrl+C stops both, and when one exits the other is stopped |
| `dev-frontend` | `pnpm dev` (https://localhost:3000) |
| `dev-backend` | `go tool mage run` (https://localhost:8443; needs `make infra-up` for PostgreSQL) |
| `cert` | `go tool mage cert` |
| `infra-up` | `cert`, then `go tool mage dockerUp` |
| `infra-down` | `go tool mage dockerDown` |
| `migrate`, `migrate-down` | `go tool mage migrateUp`, `migrateDown` |
| `seed` | `go tool mage seed` |
| `lint` | `lint-frontend` (`pnpm lint`) and `lint-backend` (`go tool mage lint`) |
| `typecheck-frontend` | `pnpm exec next typegen` and `pnpm exec tsc --noEmit` |
| `test` | `test-frontend` (`pnpm test`) and `test-backend` (`go tool mage test`) |
| `build` | `build-frontend` (`pnpm build`) and `build-backend` (`go tool mage build`) |
| `e2e` | `infra-up`, `migrate`, then `pnpm e2e` against `https://localhost` |
| `demo` | `infra-up`, `migrate`, then `pnpm demo` (see [Demonstração em aula](#demonstração-em-aula)) |
| `check` | `check-backend` (`go tool mage ci`) and `check-frontend` (`lint-frontend`, `typecheck-frontend`, `test-frontend`) |

Targets that only exist on one side (`go tool mage contractLint`, `bruno`,
`brunoRun`, `testIntegration`, `vuln`, `fmt`) are run from `backend/`, as
described in its README.
