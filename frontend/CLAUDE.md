@AGENTS.md

# Agent instructions (frontend)

This file applies to `frontend/`. The git repository is the parent folder,
which contains `backend/` and `frontend/`. `pnpm` commands run inside
`frontend/`; paths given to `git add` are relative to the repository root
(`frontend/...`).

Everything in [`../backend/CLAUDE.md`](../backend/CLAUDE.md) also applies
here: language rules, the spec-driven workflow, autonomy, Graphite branches
and commits, and Conventional Commits. This file only lists what is
different for the frontend.

## Specs

Frontend features are specified in `backend/specs/` (local, never
versioned), like the backend ones. Before any implementation, read
`backend/specs/constitution.md` and the feature's `spec.md` and `plan.md`.

## Checks before every commit

Instead of `mage ci`, run all of these in `frontend/`. Any failure blocks
the commit:

```sh
pnpm lint
pnpm exec next typegen   # route types used by tsc (LayoutProps, PageProps)
pnpm exec tsc --noEmit
pnpm test
```

Tasks that change a user flow (login, session, role guards, registration,
logout) also run `pnpm e2e` against the backend stack (`go tool mage
dockerUp`, `migrateUp` and `seed` in `backend/`). Leaving `pnpm e2e`
failing after a few attempts to fix it is a reason to stop and ask, like
`mage brunoRun` for the backend.

## Commit scope

Frontend commits use the scope `frontend`, whatever the area inside it:

```
feat(frontend): add the login page

Refs: specs/004-integracao-frontend
```

A change to `.github/workflows/frontend.yml` is `ci(frontend): ...`.

## Before each `gt create`

Run `git status` and confirm that none of these are being added:
`.env*`, `.next/`, `node_modules/`, `next-env.d.ts`, `test-results/`,
`playwright-report/`, `blob-report/`, `coverage/`, and anything under
`backend/certs/` or `backend/specs/`.

## Security rules

- Never disable TLS verification (`NODE_TLS_REJECT_UNAUTHORIZED`,
  `rejectUnauthorized: false`, `ignoreHTTPSErrors`). The dev server trusts
  `backend/certs/server.crt` through `--experimental-https-ca`.
- The CSRF token stays in memory. Never put it, or any credential, in
  `localStorage`, `sessionStorage`, IndexedDB, cookies, URLs or logs.
- No credential in the repository or in command-line arguments. E2E tests
  read them from the environment or `backend/.env`.

## Dependencies

- `shadcn` (a production dependency, used by the app only for
  `@import "shadcn/tailwind.css"`) pulls `ts-morph` → `@ts-morph/common` →
  `minimatch` 10.x. The Dependabot alert for GHSA-g3hv-75wg-m249 (minimatch
  `>= 7.0.0, < 11.0.1`, DoS through globparts amplification) was dismissed
  as `not_used`: the vulnerable code only runs with `optimizationLevel >= 2`,
  and `@ts-morph/common` calls `minimatch(path, pattern)` with the default
  level 1. `ts-morph` only runs in the shadcn CLI, never in the app.
- That path is fixed once `pnpm-lock.yaml` has `@ts-morph/common` 0.29.1 or
  newer (`minimatch ^11.0.1`), which needs `shadcn` on `ts-morph` 28 or
  newer. `ts-morph` 27 and `@ts-morph/common` 0.29.0 still use
  `minimatch ^10`. Nothing else is needed then.

## Source layout

```
src/
├── app/           Next routes (page.tsx, layout.tsx): thin, they compose the layers below
├── screens/       page bodies shared by routes (src/pages is reserved by the Pages Router)
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

Not created until a first file needs them: `atoms/` (Jotai global state),
`hooks/` (UI and behavior hooks, not fetching) and `queries/` (TanStack
Query hooks, keys and mutations).

Who may import whom (the `@/` folders; anything else is a violation):

| Layer | May import |
|---|---|
| `domain` | `domain` only |
| `config` | `config` |
| `utils` | `utils`, `config` |
| `lib` | `lib`, `domain`, `config` |
| `repositories` | `lib`, `domain`, `config` |
| `services` | `repositories`, `lib`, `domain`, `config`, `utils` |
| `components` | `components`, `domain` (types and pure values), `utils`, `styles` |
| `containers` | `containers`, `components`, `domain`, `utils`, and `repositories` directly while there is no `queries/` |
| `screens`, `app` | `services` (Server Components call them directly), `screens`, `containers`, `components`, `domain` |

- Code that uses server-only APIs (`server-only`, `next/headers`) lives in
  `services` or `lib`, never in components, containers or screens.
- A file that mixes types, pure rules and I/O is split across `domain`,
  `repositories` or `lib`.

## Estrutura de módulos e tipos

Todo módulo é uma pasta com esta forma:

```
api/
├── index.ts        # implementação (ou index.tsx para componentes)
├── types.ts        # todas as tipagens do módulo
└── index.test.ts   # testes, quando houver
```

Regras:
- Tipos, interfaces e props de componentes ficam sempre em `types.ts`, nunca em `index.ts(x)`.
  Isso vale também para tipos não exportados.
- `types.ts` só contém tipos: nada de constantes, funções ou schemas em runtime.
  Constantes e schemas zod ficam no `index.ts`. Quando o tipo deriva de um valor, use
  `import type` dentro do `types.ts`:
  `import type { ROLES } from '.'` → `export type Role = (typeof ROLES)[number]`
  e `import type { loginSchema } from '.'` → `export type LoginInput = z.infer<typeof loginSchema>`.
- Quem consome importa os tipos direto do `types.ts`, sempre com `import type`:
  `import type { ApiError } from '@/domain/api-error/types'`. Não reexporte tipos pelo `index.ts`.
- Se o módulo não tem nenhum tipo, não crie um `types.ts` vazio.
- Um arquivo solto (ex.: `domain/session/access.ts`) vira pasta (`domain/session/access/`)
  quando precisar de tipos.