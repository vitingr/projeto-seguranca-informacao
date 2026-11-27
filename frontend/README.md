# Frontend

Next.js 16 app for the backend in [`../backend`](../backend/README.md):
login, the role areas (`admin` and `comum`, the latter with the user's
profile), user registration for administrators, the ticket builder for
`comum` users, ticket lists and conversations, administrator status changes,
and logout. The backend is the only authority on sessions
and roles; this app talks to it through a same-origin proxy and never
handles the session cookie in JavaScript.

## Pages

| Path | Who | What |
|---|---|---|
| `/` | everyone | Redirects: anonymous → `/login`, `admin` → `/administradores`, `comum` → `/usuarios`. |
| `/login` | anonymous | Login form; `?next=` accepts a protected path allowed for the role, including numeric ticket pages. |
| `/administradores` | `admin` | Admin area and ticket queue, status summary and filters. |
| `/cadastro` | `admin` | User registration. |
| `/usuarios` | `comum` | The user's profile (generated avatar, login, role, registration date), "Novo ticket" and "Meus tickets". |
| `/usuarios/tickets/novo` | `comum` | Ticket builder (below). |
| `/usuarios/tickets/<id>` | `comum`, author only | Ticket details and conversation. |
| `/administradores/tickets/<id>` | `admin` | Any ticket's details, conversation and status action. |

Any other path, including the old `/minha-conta`, `/admin` and
`/cadastrar-usuarios`, is Next's 404.

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

Before a commit, run `pnpm lint`, `pnpm exec next typegen`,
`pnpm exec tsc --noEmit` and `pnpm test` (or `make check-frontend` from the
repository root). Changes to a user flow (login, session, role guards,
registration, logout) also run `pnpm e2e` against the backend stack.

## Source layout

```
src/
├── app/           Next routes (page.tsx, layout.tsx): thin, they compose the layers below
├── screens/       page bodies shared by routes
├── containers/    components wired to data: client state, repository calls
├── components/    presentational: props in, no fetch, no global state; ui/shadcn = shadcn primitives
├── services/      server use cases: server-only, cookies(), redirect()
├── repositories/  backend HTTP calls and the conversion of answers into domain types
├── lib/           infrastructure: API contract, CSRF store, error reader, server-side fetch
├── domain/        types, constants and pure rules: no React, no Next, no I/O
├── config/        env and the backend origin
├── styles/        global CSS and theme tokens
├── utils/         generic helpers with no business knowledge
└── test/          helpers for unit tests only
```

Imports go one way:

| Layer | May import |
|---|---|
| `domain` | `domain` only |
| `config` | `config` |
| `utils` | `utils`, `config` |
| `lib` | `lib`, `domain`, `config` |
| `repositories` | `lib`, `domain`, `config` |
| `services` | `repositories`, `lib`, `domain`, `config`, `utils` |
| `components` | `components`, `domain` (types and pure values), `utils`, `styles` |
| `containers` | `containers`, `components`, `domain`, `utils`, `repositories` |
| `screens`, `app` | `services`, `screens`, `containers`, `components`, `domain` |

Code that uses server-only APIs (`server-only`, `next/headers`) lives in
`services` or `lib`, never in components, containers or screens.

Each module is a folder with `index.ts(x)`, `types.ts` (every type, interface
and props type of the module, and nothing else) and `index.test.ts` when it
has tests. Consumers import types straight from `types.ts` with
`import type`; `index.ts` does not re-export them.

Never put the CSRF token or any credential in `localStorage`,
`sessionStorage`, IndexedDB, cookies, URLs or logs, and never disable TLS
verification (`NODE_TLS_REJECT_UNAUTHORIZED`, `rejectUnauthorized: false`).

Layout: spacing between siblings uses `flex` or `grid` with `gap-*`, every
page section uses `mx-auto w-full max-w-7xl px-4 sm:px-6 lg:px-8`, classes are
mobile-first (screens work from 360px wide), and colors and radii come from
the tokens in `src/styles/globals.css`.

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
  server reads it only to forward it, alone, to `GET /api/me`, the area
  endpoints and ticket reads, never caching the answer.
- **The CSRF token** lives in JavaScript memory only
  (`src/lib/api/csrf`): fetched before the first write (POST or PATCH), fetched again
  after login, discarded at logout. It is never written to Web Storage,
  IndexedDB, cookies or URLs.
- **Protected pages** (`/administradores`, `/cadastro` for `admin`,
  `/usuarios` and `/usuarios/tickets/novo` for `comum`, plus each role's
  dynamic ticket pages) check the session
  on the server in each page
  (`src/services/session`). No cookie or a 401 → `/login?next=<page>`;
  another role → the "no permission" screen. The area pages also call
  their backend endpoint, whose 403 has the last word. A role change in
  the database applies on the next page load.
- **During use**, any 401 (except a wrong password on login) goes back to
  the login page with "Sua sessão expirou". A 403 on a write refreshes the
  token and retries once: that is how an expired session (which the
  backend answers with 403 on a POST, since CSRF runs before the role
  check) ends on the login page.
- **Errors** are shown by the `error` code of the backend response, with
  Portuguese messages (`src/domain/messages`). A response that is
  not a backend error (backend down, proxy error) is shown as a connection
  error.

## Ticket builder

`/usuarios/tickets/novo` (`src/containers/ticket-builder`) creates a ticket
with `POST /api/tickets` in three steps and a confirmation: **Tipo** (one of
"Solicitar acesso", "Suporte em bug", "Deploy de projeto", "Outro"),
**Detalhes** (the form of that type), **Revisão**, then the confirmation
with the ticket number and the status "Aberto".

- **Validation.** The zod schemas of each type (`schemas.ts`) apply the
  same limits and formats as the backend (`internal/security/tickets.go`):
  trimmed values, lengths in code points, no control characters in
  single-line fields, `https://` repository URLs with a host and no
  credentials. An invalid form sends nothing. The backend still validates
  everything; its 400 and 422 are shown as a generic message.
- **Plain text.** Values are rendered as React text only, never as HTML;
  ESLint forbids `dangerouslySetInnerHTML` (`react/no-danger` is an error).
- **No draft anywhere.** Typed values live only in React memory: never in
  Web Storage, IndexedDB, cookies or the URL. The URL carries only the
  step (`?etapa=detalhes|revisao|confirmacao`), so the browser's Back and
  Forward move one step and never re-send. A reload or an expired session
  loses the data, which the form says.
- **Leave guard.** With unsent data, closing or reloading the tab asks the
  browser's `beforeunload` prompt, and an internal link or "Sair" shows an
  inline "Sair sem enviar?" bar instead of leaving.
- **Send.** One request per click (a second click is ignored); the send
  progress ("Validando campos", "Registrando ticket", "Preparando
  confirmação") only marks what really happened and never delays the
  answer. Errors (422, 429, 403, 500, network) keep the data and allow
  sending again; a 429 is not retried by the client. With reduced motion
  the same states appear without animation.
- **Titles.** `ticketTitle` (`src/domain/tickets`) builds the title the
  server will store (BR-3), from the same test vectors as the Go service.

## Ticket tracking

"Meus tickets" and the administrator queue show 20 tickets per page,
ordered by the latest update and then by number (descending). Status and
type filters combine; the queue adds an author filter and a status summary
over the whole queue. Only users with tickets appear in the author filter,
up to the backend's limit of 500 logins. Long logins in the queue are
truncated, with the full login in `title`. Tables become rows below 768 px.
The exact area texts and the link to `/cadastro` are preserved.

- **Addresses.** `status`, `tipo`, `usuario` (queue only) and `pagina`
  preserve list filters and paging. Unknown values become "Todos" before
  calling the backend. Ticket pages preserve `aba=conversa`; a validated
  `de` address restores the list filters on return. The builder's
  confirmation links to the stored ticket with "Ver ticket".
- **Details and conversation.** Both roles use `screens/ticket-page`:
  `ticket-header`, `detail-list`, `ticket-tabs`, `thread`, `thread-message`
  and `thread-event`. Details preserve every stored field and line break;
  HTML, Markdown and URLs remain plain text. Own messages appear as "Você";
  administrators appear as "Suporte" to the author, without their logins
  in the backend response. The admin view names the author and other admins.
- **Reads and refresh.** `services/tickets` loads through the server-only
  backend client; `repositories/tickets` validates each answer and refuses
  unexpected fields or shapes. Lists and ticket pages load on navigation,
  reload, after an accepted write and with "Atualizar". No polling,
  WebSocket, automatic status update or notification is used.
- **Replies.** `containers/ticket-conversation` owns the text and send
  state; `components/reply-box` renders the growing textarea, counter,
  square send button and separate gray visibility tray. Enter adds a
  line; Ctrl+Enter (Command+Enter on macOS) sends. `domain/tickets` mirrors
  the backend's 1–2000 code-point message rule. A single-flight guard
  prevents duplicate simultaneous sends. Success clears the text,
  refreshes the page and announces "Mensagem enviada."; errors keep the
  text, except session expiry redirects to login and loses in-memory data.
- **Leaving.** The shared `useLeaveGuard` hook lives in
  `containers/leave-guard`, with the presentational inline bar. It is used
  by the builder and the conversation; generic click rules live in
  `utils/navigation-guard`. Closing/reloading uses `beforeunload`, while
  links and "Sair" show the inline confirmation. Nothing is saved to
  browser storage or URLs.
- **Status.** `containers/status-action`, only on the admin page, offers
  the transitions of `ALLOWED_NEXT_STATUSES`. Closing asks for inline
  confirmation with an outlined red-text button. A reply never changes
  status. A stale change (409) refreshes the page and asks the admin to
  check the new state. "Fechado" is final: both sides keep reading, with
  no reply box or status menu.
- **Limits and errors.** Replies and status changes each allow 30 requests
  per user in 60 seconds, enforced by separate backend limiters. A 429 is
  not retried automatically. Own-ticket access denied, missing ticket and
  invalid id share the not-found screen; wrong-role pages show the
  forbidden screen. Backend checks remain authoritative.
- **Known limits.** The page shows only the latest 100 thread entries.
  Older-entry loading was deferred: the backend cursor API and client
  parser exist, but there is no UI control yet. Offset pages can repeat or
  miss a row when another ticket moves to the top between page loads.
  There is no idempotency key for a retry after a network failure.

The tracking Vitest suite covers contracts and parsers, route guards,
queries, message/transition rules, components, contrast and write errors.
Playwright adds the list, queue, details, conversation, replies, leave
guard and status flows in `meus-tickets.spec.ts`, `fila.spec.ts`,
`ticket-page.spec.ts`, `ticket-conversa.spec.ts`, `ticket-resposta.spec.ts`,
`ticket-admin.spec.ts` and `ticket-status.spec.ts`. Test users are removed
with the backend README cleanup, which deletes entries before tickets
and users. Bruno runs the tracking collection after the existing ticket
and access-matrix collections.

## Design rules

- **Look.** Poppins, light theme only, thin borders, one blue accent
  (`--brand`) used only by the send progress and the active step.
- **Hard rules**, checked by `src/test/design-rules.test.ts` over every
  `src/**/*.{ts,tsx,css}` file (comments included): radius `rounded-sm`
  only (plus `rounded-full` and `rounded-none`), no `shadow-*` or
  `box-shadow`, no gradients, no `tracking-*` classes. The shadow tokens in
  `globals.css` are empty shadows.
- **Components.** shadcn (`src/components/ui/shadcn`) and the existing
  Base UI primitives, adapted to the rules above. Ticket tracking adds
  no visual library; affectUI is not used.
- **Avatars** are generated in the browser from the login (initials and a
  figure in an inline SVG, `src/domain/avatar`): no image, no request to
  another origin.

## Tests

- **Unit** (`pnpm test`, Vitest in Node): the API client and CSRF token
  rules, error messages, the server session rules, the `next` filter after
  login and the root redirect, form validation (including the ticket
  schemas with the backend's boundary values), the ticket titles, the
  builder's reducer, history and send rules, the avatar (with an AA
  contrast check of every tint), components rendered to static markup
  with `react-dom/server` (no DOM library), the design guard, the dev
  launcher, and a check that `src/lib/api/contract` matches
  `backend/api/openapi.json`.
- **End-to-end** (`pnpm e2e`, Playwright and Chromium) against the real
  backend: the proxy and cookie attributes, login, the role matrix of the
  pages, the root redirect and the old paths (404), the header on small
  screens, registration (including a taken login and an expired session),
  a role change, logout (including the Back button), the profile on
  `/usuarios`, and the ticket builder (each type to the confirmation,
  field errors with no request, HTML shown as text, Back and reload,
  the leave guard, each send error simulated with `page.route`, reduced
  motion, nothing left in browser storage).
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
