# backend_si

HTTPS JSON API consumed by the frontend (in `../frontend`). It is written in
Go and stores users and server-side sessions in PostgreSQL.

Users log in with a login and password (Argon2id hashes) and get a
server-side session in a cookie. State-changing requests need a CSRF token,
and each route declares which roles may call it. User registration through
the API (`POST /api/cadastro`) is not implemented yet and fails closed.

## Prerequisites

| Tool | Version | Used for |
|---|---|---|
| Go | 1.27 (downloaded automatically by older `go` commands) | build, tests and every `mage` target (`go tool mage`) |
| golangci-lint | 2.14.0, on the `PATH` | `mage fmt`, `mage lint` |
| Docker with Compose | recent | `mage dockerUp` (app + PostgreSQL) |
| Node.js with `npx` | current LTS (24.x) | `mage contractLint`, `mage bruno`, `mage brunoRun` |

Node.js is not needed for `mage ci`. The npm packages run through `npx` at
pinned versions, so nothing is installed globally.

## Getting started

```sh
cp .env.example .env        # then set POSTGRES_PASSWORD (openssl rand -hex 24)
                            # and SEED_ADMIN_PASSWORD (15 to 128 characters)
go tool mage cert           # self-signed certificate for localhost in certs/
go tool mage dockerUp       # app on https://localhost, PostgreSQL on 127.0.0.1
go tool mage migrateUp
go tool mage seed           # creates the first administrator (once)
go tool mage brunoRun       # run the API assertions against the stack
```

`SEED_ADMIN_PASSWORD` goes into the JSON body of the Bruno login request
as is, so avoid `"` and `\` in it (`openssl rand -hex 16` works).

If port 5432 is taken on your machine, set `POSTGRES_HOST_PORT` in `.env`.

## Mage targets

Run `go tool mage -l` for the full list.

| Target | What it does |
|---|---|
| `build` | Builds `bin/backend_si`. |
| `run` | Runs the server outside Docker, reading `.env`. |
| `test` | Runs the tests with the race detector and coverage, plus the magefile tests. |
| `fmt` / `lint` | gofumpt + goimports / golangci-lint with `.golangci.yml`. |
| `vuln` | govulncheck. |
| `ci` | `fmt`, `lint`, `vuln` and `test`, in that order. Run it before every commit. |
| `cert` | Creates `certs/server.crt` and `certs/server.key` for localhost and 127.0.0.1. |
| `dockerUp` / `dockerDown` | Starts or stops the app and PostgreSQL. |
| `migrateUp` / `migrateDown` | Applies all pending migrations / reverts the last one (goose). |
| `seed` | Creates the first administrator from `SEED_ADMIN_LOGIN` and `SEED_ADMIN_PASSWORD`. The login is lowercased and must match `^[a-z0-9._-]{3,64}$`; the password must have 15 to 128 characters. A second run fails with a conflict. |
| `contractLint` | Validates `api/openapi.json` as OpenAPI 3.1 (Redocly CLI). |
| `bruno` | Regenerates the Bruno request folders from `api/openapi.json`. |
| `brunoRun` | Runs `bruno/tests` against the running API and writes `bruno-report.xml` (JUnit). It logs in as the seed administrator: `SEED_ADMIN_*` from the environment or `.env` reach Bruno as environment variables, so run `seed` first. |

## API contract and Bruno collection

Every route exists in three places that change together:
- `api/openapi.json`, the contract;
- the route table in `internal/web/routes.go`;
- the Bruno collection in `bruno/`.

The only exception is the 404 catch-all, which stays out of the contract.

- **Contract.** It describes the behavior of each route, successes and
  errors. Each operation declares its access policy in the `x-policy`
  extension: `"public"` or a list of roles. A Go test in `internal/web`
  fails when the contract and the code disagree. It compares routes,
  policies, the CSRF header, error responses and field names.
- **Generated folders.** `health/`, `auth/`, `users/` and `areas/` are
  created by `mage bruno`, one per contract tag. Never edit them by hand;
  change the contract and run `mage bruno`. CI fails when they are out of
  date.
- **Hand-written files.** `opencollection.yml` holds the checks that run on
  every response. `environments/local.yml` holds `baseUrl`. `tests/` holds
  the assertions that `mage brunoRun` runs, in `seq` order with a shared
  cookie jar: anonymous checks, rejected logins, then login as the seed
  administrator, `/api/me`, logout, and the old cookie getting 401.
- **Other addresses.** `mage brunoRun` targets `https://localhost`, which is
  the stack started by `mage dockerUp`. For `mage run`, set
  `BRUNO_BASE_URL=https://localhost:8443`.

## Sessions and security settings

| Setting | Default | Meaning |
|---|---|---|
| `SESSION_IDLE_TIMEOUT` | `15m` (1m to 24h) | A session without requests for this long ends. Expired sessions are also purged on this interval. |
| `SESSION_ABSOLUTE_TIMEOUT` | `8h` (1m to 168h, at least the idle timeout) | A session ends this long after login, even with continuous activity. |

Fixed in code:
- Every request has a 10s deadline, below the 15s `WriteTimeout`, so a
  request waiting for the database or for a password hash ends with a JSON
  500 instead of hanging.
- At most `GOMAXPROCS` Argon2id hashes (64 MiB each) run at once.
- Login answers the same 401 for an unknown user and a wrong password,
  after the same Argon2id work.
- Clients must call `GET /api/csrf` again after login, because the CSRF
  token changes with the session.

## Go version

The module requires Go 1.27: `go 1.27.0` with `toolchain go1.27.4` in
`go.mod`. With an older Go installed, the `go` command downloads that
toolchain automatically (`GOTOOLCHAIN=auto`, the default). Keep three places
aligned:
- `go.mod`;
- the `golang:1.27` build image in the `Dockerfile`;
- `GO_VERSION` (`1.27.x`) in `.github/workflows/backend.yml`, at the
  repository root.

Only the two newest Go releases get security fixes, so `mage vuln` starts
reporting standard library vulnerabilities when a release falls out of
support. Bump the toolchain, then the major version, when that happens.

The tools run with `go run` (goose v3.29.0 and govulncheck v1.9.0) are
pinned in `magefiles/magefile.go`. Upgrade them on purpose, checking that
each one supports the Go version in `go.mod`.

## Dependencies

Every dependency is listed here with the reason it is needed.

**Go modules**

| Module | Why |
|---|---|
| `github.com/jackc/pgx/v5` | PostgreSQL driver, used through `database/sql` (`pgx/v5/stdlib`). It is the most complete pure-Go PostgreSQL driver and maps unique-violation errors through `pgconn`. |
| `github.com/magefile/mage` | Task runner written in Go, run as a module tool (`go tool mage`), so there is no Makefile or extra binary. |
| `golang.org/x/crypto` | Only `argon2` (`argon2.IDKey`) for password hashing; the standard library has no Argon2id. Pinned to v0.58.0, the newest release at least two weeks old. `govulncheck` lists GO-2026-5932 for this module, but it affects only `x/crypto/openpgp`, which we never import, so `mage vuln` passes. It brings `golang.org/x/sys` as an indirect dependency. |
| `golang.org/x/text` | Only `unicode/norm`: passwords are normalized to NFKC before they are validated and hashed, so visually equivalent inputs give the same hash. It was already in the module graph through `pgx`; the standard library has no Unicode normalization. |

**Tools run with `go run`** (versions pinned in `magefiles/magefile.go`)

| Tool | Why |
|---|---|
| `github.com/pressly/goose/v3` | Runs the SQL migrations in `migrations/`. It is built without the drivers we do not use. |
| `golang.org/x/vuln/cmd/govulncheck` | Reports known vulnerabilities that the code actually reaches. |

**npm packages run with `npx`** (exact versions pinned in `magefiles/`)

| Package | Why |
|---|---|
| `@redocly/cli@2.72.1` | Validates the contract as OpenAPI 3.1 (`mage contractLint`). `redocly.yaml` documents the rules we turn off. |
| `@usebruno/cli@4.3.0` | Generates the Bruno collection from the contract and runs the assertions (`mage bruno`, `mage brunoRun`). The collections are plain YAML files that live in the repository. |

**Container images and CI**

- `golang:1.27` builds the binary.
- `gcr.io/distroless/static-debian13:nonroot` runs it, without a shell or
  root.
- `postgres:18-alpine` is the database.
- GitHub Actions are pinned by commit SHA in
  `.github/workflows/backend.yml`.
