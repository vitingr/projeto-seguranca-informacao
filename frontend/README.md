# Frontend

Next.js 16 app for the backend in [`../backend`](../backend/README.md):
login, the role areas (`admin` and `comum`), user registration for
administrators, and logout. The backend is the only authority on sessions
and roles; this app talks to it through a same-origin proxy and never
handles the session cookie in JavaScript.

## Prerequisites

- Node.js 24 and pnpm 10.28 (`corepack enable` picks the version from
  `packageManager` in `package.json`).
- The backend stack running, as described in its README. From `backend/`:

  ```sh
  go tool mage cert        # backend/certs/server.crt and server.key
  go tool mage dockerUp    # API on https://localhost
  go tool mage migrateUp
  go tool mage seed        # first administrator (SEED_ADMIN_* in backend/.env)
  ```

## Getting started

```sh
pnpm install
pnpm dev                   # https://localhost:3000
```

`pnpm dev` runs `next dev` over HTTPS with the **backend certificate**
(`backend/certs/server.crt` and `server.key`), bound to `localhost` only.
The browser shows a certificate warning until you trust that certificate:
accept it once, or add it to your user's trusted roots (on Windows:
`certutil -user -addstore Root ..\backend\certs\server.crt`). It is a leaf
certificate for `localhost` and `127.0.0.1` only and cannot sign anything
else; no mkcert or local CA is installed.

Settings (copy `.env.example` to `.env.local` to change them):

| Variable | Default | Meaning |
|---|---|---|
| `API_URL` | `https://localhost` | Backend origin for the `/api` proxy; empty falls back to `API_BASE_URL`. Must be `https`, with no path. Use `https://localhost:8443` with `go tool mage run`. Server-only: never `NEXT_PUBLIC_*`. |
| `API_BASE_URL` | none (required) | Backend origin for server-side calls, and the fallback of `API_URL`. Same rules as `API_URL`. Server-only. |
| `NEXT_PUBLIC_BASE_URL` | `https://localhost:3000` | Public URL of this app (metadata). |
| `BACKEND_CERT_DIR` | `../backend/certs` | Folder with `server.crt` and `server.key`. Read by `pnpm dev` before Next starts, so set it in the shell, not in `.env.local`. |

## Scripts

| Script | What it does |
|---|---|
| `pnpm dev` | `scripts/dev.mjs`: checks the certificate files, then `next dev` over HTTPS on `localhost:3000`. |
| `pnpm lint` | ESLint and Prettier. |
| `pnpm exec next typegen` | Route types (`PageProps`, `LayoutProps`). Needed by `tsc` on a clean checkout. |
| `pnpm exec tsc --noEmit` | Type-check. |
| `pnpm test` | Vitest unit tests. |
| `pnpm e2e:install` | Installs Chromium for Playwright (once). |
| `pnpm e2e` | Playwright end-to-end tests against the running backend (starts `pnpm dev` if it is not running). |
| `pnpm build`, `pnpm start` | Production build and server. Production deployment is not set up yet. |

The checks to run before every commit are listed in
[`CLAUDE.md`](CLAUDE.md).

## How the session works

- **One origin.** The browser only talks to `https://localhost:3000`. A
  Next rewrite (`next.config.ts`) forwards `/api/*` to `API_URL`, so the
  `__Host-session` cookie (`HttpOnly`, `Secure`, `SameSite=Lax`) and the
  backend's Fetch Metadata check (`Sec-Fetch-Site: same-origin`) work as
  designed. The app defines no route of its own under `/api`.
- **Trusting the backend.** The Next server verifies the backend
  certificate: `pnpm dev` passes `--experimental-https-ca` with
  `server.crt`, and `next dev` turns it into the server process's
  `NODE_EXTRA_CA_CERTS` (it overrides any value set in the environment).
  TLS verification is never disabled; with an untrusted certificate every
  backend call fails.
- **The cookie** is handled only by the browser and the backend. The Next
  server reads it only to forward it, alone, to `GET /api/me` and the area
  endpoints, never caching the answer.
- **The CSRF token** lives in JavaScript memory only
  (`src/lib/api/csrf`): fetched before the first POST, fetched again
  after login, discarded at logout. It is never written to Web Storage,
  IndexedDB, cookies or URLs.
- **Protected pages** (`/administradores`, `/cadastro` for `admin`,
  `/usuarios` for `comum`) check the session on the server in each page
  (`src/services/session`). No cookie or a 401 → `/login?next=<page>`;
  another role → the "no permission" screen. The area pages also call
  their backend endpoint, whose 403 has the last word. A role change in
  the database applies on the next page load.
- **During use**, any 401 (except a wrong password on login) goes back to
  the login page with "Sua sessão expirou". A 403 on a POST refreshes the
  token and retries once: that is how an expired session (which the
  backend answers with 403 on a POST, since CSRF runs before the role
  check) ends on the login page.
- **Errors** are shown by the `error` code of the backend response, with
  Portuguese messages (`src/domain/messages`). A response that is
  not a backend error (backend down, proxy error) is shown as a connection
  error.

## Tests

- **Unit** (`pnpm test`, Vitest in Node): the API client and CSRF token
  rules, error messages, the server session rules, the `next` filter after
  login, form validation, the dev launcher, and a check that
  `src/lib/api/contract` matches `backend/api/openapi.json`.
- **End-to-end** (`pnpm e2e`, Playwright and Chromium) against the real
  backend: the proxy and cookie attributes, login, the role matrix of the
  pages, registration (including a taken login and an expired session),
  a role change, and logout (including the Back button).
  - Chromium trusts only the backend certificate's public key
    (`--ignore-certificate-errors-spki-list`); any other certificate error
    fails the tests.
  - Credentials come from the environment or `backend/.env`
    (`SEED_ADMIN_LOGIN`, `SEED_ADMIN_PASSWORD`; `POSTGRES_USER` and
    `POSTGRES_DB` for the role-change test, which runs `docker compose exec
    db psql` in `backend/`). Traces, screenshots and videos are off so no
    password is recorded.
  - Each run registers `e2e.*` users (one comum user for the run, plus
    those created by the registration tests). They are deleted with the
    command in the backend README,
    [Test users left by `mage brunoRun`](../backend/README.md#test-users-left-by-mage-brunorun).

## Dependencies added by the project

Each dependency added after `create-next-app` is justified here.

| Package | Kind | Why |
|---|---|---|
| `vitest` | dev | Unit tests (`pnpm test`): the API client, the session rules, form validation and the check that `src/lib/api/contract` matches `backend/api/openapi.json`. Runs in Node, without a DOM library. |
| `@playwright/test` | dev | End-to-end tests (`pnpm e2e`) in Chromium against the real backend: login, role access, registration, logout and session loss. Trusts only the backend certificate's key. Browsers are installed with `pnpm e2e:install`. |
