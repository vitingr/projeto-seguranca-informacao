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
| `check` | `check-backend` (`go tool mage ci`) and `check-frontend` (`lint-frontend`, `typecheck-frontend`, `test-frontend`) |

Targets that only exist on one side (`go tool mage contractLint`, `bruno`,
`brunoRun`, `testIntegration`, `vuln`, `fmt`) are run from `backend/`, as
described in its README.
