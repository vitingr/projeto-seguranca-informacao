# backend_si

HTTPS JSON API consumed by the frontend (in `../frontend`). It is written in
Go and stores users, server-side sessions and support tickets in PostgreSQL.

Users log in with a login and password (Argon2id hashes) and get a
server-side session in a cookie. State-changing requests need a CSRF token,
and each route declares which roles may call it (see
[Access control](#access-control)). Administrators register users through
`POST /api/cadastro`; comum users open support tickets through
`POST /api/tickets`.

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
| `testIntegration` | Runs the integration tests (`Test...Integration`) against the PostgreSQL of `dockerUp`, after `migrateUp`. `TEST_DATABASE_URL` is built from `.env`. CI runs it in the `e2e` job. |
| `fmt` / `lint` | gofumpt + goimports / golangci-lint with `.golangci.yml`. |
| `vuln` | govulncheck. |
| `ci` | `fmt`, `lint`, `vuln` and `test`, in that order. Run it before every commit. |
| `cert` | Creates `certs/server.crt` and `certs/server.key` for localhost and 127.0.0.1. |
| `dockerUp` / `dockerDown` | Starts or stops the app and PostgreSQL. |
| `migrateUp` / `migrateDown` | Applies all pending migrations / reverts the last one (goose). |
| `seed` | Creates the first administrator from `SEED_ADMIN_LOGIN` and `SEED_ADMIN_PASSWORD`. The login is lowercased and must match `^[a-z0-9._-]{3,64}$`; the password must have 15 to 128 characters. A second run fails with a conflict. |
| `contractLint` | Validates `api/openapi.json` as OpenAPI 3.1 (Redocly CLI). |
| `bruno` | Regenerates the Bruno request folders from `api/openapi.json`. |
| `brunoRun` | Runs the requests of `bruno/tests`, then `bruno/tests/access-matrix`, then `bruno/tests/tickets`, then `bruno/tests/ticket-tracking`, against the running API and writes `bruno-report.xml` (JUnit). It logs in as the seed administrator: `SEED_ADMIN_*` from the environment or `.env` reach Bruno as environment variables, so run `seed` first. Each run also registers `e2e.*` test users (see [Test users](#test-users-left-by-mage-brunorun)). |

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
- **Generated folders.** `health/`, `auth/`, `users/`, `areas/`, `tickets/` and `admin-tickets/` are
  created by `mage bruno`, one per contract tag. Never edit them by hand;
  change the contract and run `mage bruno`. CI fails when they are out of
  date.
- **Hand-written files.** `opencollection.yml` holds the checks that run on
  every response. `environments/local.yml` holds `baseUrl`. `tests/` holds
  the assertions that `mage brunoRun` runs, in `seq` order with a shared
  cookie jar: anonymous checks, rejected logins, then login as the seed
  administrator, `/api/me`, logout, and the old cookie getting 401.
  `tests/access-matrix/` runs next (`bru run tests tests/access-matrix
  tests/tickets tests/ticket-tracking`; `-r` would run subfolders first):
  the cells of the [access matrix](#access-control), anonymous, then as the
  seed administrator, who registers a comum user, then as that user; the
  ticket tracking cells follow in the same caller order.
  `tests/tickets/` runs next, as that comum user: the CSRF refusals, a 400,
  a 422, one ticket of each type, and the rate limit: a second comum user
  sends 10 creations and gets 429 on the 11th.
  `tests/ticket-tracking/` runs last, with users A and B of its own:
  "Meus tickets" with its filters and 400 cases, and B never seeing A's
  tickets.
- **Other addresses.** `mage brunoRun` targets `https://localhost`, which is
  the stack started by `mage dockerUp`. For `mage run`, set
  `BRUNO_BASE_URL=https://localhost:8443`.

## Access control

Each route declares its policy in `internal/web/routes.go` (and `x-policy`
in the contract). Anything not allowed is denied: anonymous callers of a
restricted route get 401, callers with the wrong role get 403.

| Endpoint | anonymous | comum | admin |
|---|---|---|---|
| `GET /api/administradores` | 401 | 403 | 200 |
| `GET /api/usuarios` | 401 | 200 | 403 |
| `POST /api/cadastro` | 401 | 403 | 201 |
| `GET /api/me` | 401 | 200 | 200 |
| `POST /api/logout` | 401 | 204 | 204 |
| `POST /api/tickets` | 401 | 201 | 403 |
| `GET /api/tickets` | 401 | 200 | 403 |
| `GET /api/tickets/{id}` | 401 | 200 | 403 |
| `GET /api/tickets/{id}/mensagens` | 401 | 200 | 403 |
| `POST /api/tickets/{id}/mensagens` | 401 | 201 | 403 |
| `GET /api/administradores/tickets` | 401 | 403 | 200 |
| `GET /api/administradores/tickets/autores` | 401 | 403 | 200 |
| `GET /api/administradores/tickets/{id}` | 401 | 403 | 200 |

Administrators do not reach `/api/usuarios`: it is the comum users' area.
They do not create tickets either, nor read "Meus tickets". The Go test
`TestAccessMatrix` (`internal/web`) and the Bruno folder
`tests/access-matrix` both cover the 39 cells (the `{id}` cells use a
ticket of the comum user).

Routes with a ticket id also run `security.TicketAccess`, after the role
check: a comum user only reaches their own tickets. Someone else's
ticket, a ticket that does not exist and a malformed id all get the same
404 (status, headers and body), and the attempt is logged at Warn with the
user id and the ticket id.

The role is read from the database on every request, with the session. A
role changed in the `users` table applies to the user's next request, with
no new login.

**Registration** (`POST /api/cadastro`, admin only) answers:
- 201 with `{user_id, login, role}`. The login is lowercased; the password
  is stored as an Argon2id hash.
- 400 `bad_request` when the body is not well-formed: malformed JSON, a
  `Content-Type` other than `application/json`, an unknown field, or a
  field of the wrong JSON type (`"papel": 1`). Also when the body is not
  valid UTF-8, when a key is not a documented name spelled exactly
  (`"PAPEL"`, `"Login"`), or when a key repeats, exactly or in another case
  (`{"papel":"comum",…,"Papel":"admin"}`). The login (`POST /api/login`)
  follows the same body rules, so such a body starts no session.
- 422 `unprocessable_entity` when the body is well-formed but a value is
  invalid: the login pattern, the password length (15 to 128), a `papel`
  other than exactly `admin` or `comum`, or a missing field.
- 409 `conflict` when the login is taken, also by a login that differs only
  in case.

Every registration is logged at Info as `user registered`, with the
administrator's `actor_user_id`, the new `user_id` and the `role`. The log
never has the login or the password.

**Tickets** (`POST /api/tickets`, comum only) answer:
- 201 with `{id, type, title, status, created_at, updated_at}`. The author
  is the session user, the status is always `aberto`, and the id and dates
  come from the database. The body never sets them. The title of `acesso`,
  `bug` and `deploy` tickets is built by the server (`Acesso a <sistema>`,
  `Bug em <sistema>`, `Deploy de <projeto> em <ambiente>`), cut to 120
  characters; `outro` keeps `titulo`.
- 400 `bad_request` when the body is not well-formed: malformed JSON, not
  valid UTF-8, a key that is not a documented name spelled exactly, a key of
  another type, a duplicate key (also in another case), a wrong JSON type,
  or `dados` missing or `null`.
- 422 `unprocessable_entity`, with a message that names no field, when a
  value breaks a rule: an unknown `tipo`, a missing or empty field, a
  length, a control character, a code outside its list, or a repository
  that is not an `https://` URL without credentials.
- 429 `too_many_requests` with `Retry-After` (whole seconds) after 10
  creation requests by the same user in 60 seconds. Every request that
  passes CSRF and the role check counts, valid or not (see
  [Known limitations](#known-limitations)).

Nothing is stored unless the answer is 201. The rules live in
`internal/security` (`DecodeTicket`, `ValidateTicket`), and the `tickets`
table repeats them as CHECK constraints. Every creation is logged at Info as
`ticket created` with `ticket_id`, `author_id` and `type`; refusals as
`ticket refused` with `author_id` and a fixed reason. The log never has a
field value.

**Ticket tracking** (spec 003) lets a comum user follow their tickets:
- `GET /api/tickets` ("Meus tickets", comum only) answers 200 with a page
  of the session user's tickets and nobody else's, newest update first, 20
  per page: `{items, total, page, pages, page_size}`. Items are summaries
  (`id, type, status, title, created_at, updated_at`): never the fields of
  the type nor any author data. The filters `status` and `tipo` and the
  page `pagina` are the only parameters; anything else (`usuario`
  included), a repeated or empty parameter, or a malformed query is 400. A
  page past the end is 200 with no items and the real totals.
- `GET /api/tickets/{id}` (comum only, own tickets) answers 200 with the
  summary plus `data`: every field of the type under the keys of the
  creation request, `null` for empty notes. No author data.
- `GET /api/tickets/{id}/mensagens` (comum only, own tickets) answers 200
  with the thread, oldest first: messages and status changes, each with
  its `side` (`author` or `support`) and `mine`, never an administrator's
  login or id. At most 100 entries per page; `has_more` and
  `?antes=<entry id>` read the older ones. Any other parameter is 400.
- `POST /api/tickets/{id}/mensagens` (comum only, own tickets) with
  `{"texto"}` answers 201 with the stored message. 400 for a body that is
  not well-formed (any key but `texto`), 422 for a text outside 1 to 2000
  characters after trimming or with a forbidden control character, 409 on
  a `fechado` ticket, 429 after 30 replies in 60 seconds. A reply moves
  `updated_at` and never the status.

Administrators work the tickets on their own routes (comum users get 403):
- `GET /api/administradores/tickets` (the queue) answers every user's
  tickets with the rules of "Meus tickets", plus the filter `usuario` (a
  login), `author_login` on each item and `status_counts` over the whole
  queue.
- `GET /api/administradores/tickets/autores` answers `{logins}`: the users
  with at least one ticket, in byte order, at most 500 (more is a known
  limitation). It reads no parameter: any is 400.
- `GET /api/administradores/tickets/{id}` answers any user's ticket like
  `GET /api/tickets/{id}`, plus `author_login`.

Refused queries are logged at Info as `ticket query refused` with
`user_id` and a fixed reason, never the query.

### Test users left by `mage brunoRun`

The access matrix registers a comum user through the seed administrator.
`mage brunoRun` generates its login (`e2e.` and 26 random characters) and
password with `crypto/rand` for every run and passes them to Bruno as the
`BRUNO_COMUM_LOGIN` and `BRUNO_COMUM_PASSWORD` environment variables, never
as arguments. `tests/tickets` registers a second one, the same login plus
`.rl`, for the rate limit test, and `tests/ticket-tracking` registers users
A and B, the same login plus `.a` and `.b`, and the reply rate limit user,
plus `.rlm`. So every run leaves five `e2e.*` users in the database, with
the tickets and messages they created.
`mage brunoRun` refuses a `SEED_ADMIN_LOGIN` that starts with `e2e.`, so the
cleanup below never matches the seed administrator.

To delete them, with the `POSTGRES_USER` and `POSTGRES_DB` values of
`.env`:

```sh
docker compose exec db psql -U <POSTGRES_USER> -d <POSTGRES_DB> \
  -c "DELETE FROM ticket_entries WHERE ticket_id IN (SELECT t.id FROM tickets t JOIN users u ON u.id = t.author_id WHERE u.login LIKE 'e2e.%') OR actor_id IN (SELECT id FROM users WHERE login LIKE 'e2e.%');" \
  -c "DELETE FROM tickets WHERE author_id IN (SELECT id FROM users WHERE login LIKE 'e2e.%');" \
  -c "DELETE FROM users WHERE login LIKE 'e2e.%';"
```

Their sessions are deleted with them (`ON DELETE CASCADE`). Tickets and
ticket entries (messages and status changes) are not (they are history),
so they are deleted first: the entries of their tickets and the entries
they wrote, then their tickets, then the users.

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

## Known limitations

- **The ticket creation rate limit lives in process memory.** At most 10
  ticket creation requests per user in any 60-second window
  (`security.RateLimiter`, `TicketCreateLimit` and `TicketCreateWindow`),
  answered with 429 and `Retry-After` over the limit. The counts are kept
  in the memory of the backend process: a restart resets every count, and
  with more than one backend instance each instance counts on its own, so a
  user could send 10 per instance. The stack runs a single backend
  container, so this is accepted. If the backend ever scales out, the fix is
  a shared store for the counts (PostgreSQL or Redis) behind the same
  `RateLimiter` API. The reply limit of spec 003 (`MessageCreateLimit`,
  30 per user in 60 seconds) has the same limitation.

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
| `github.com/magefile/mage` | Task runner written in Go, run as a module tool (`go tool mage`), so the backend needs no Makefile or extra binary. The root `Makefile` only calls these targets. |
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
