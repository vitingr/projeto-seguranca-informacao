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
