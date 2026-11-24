# Agent instructions (repository root)

The repository has two projects, each with its own rules:

- [`backend/AGENTS.md`](backend/AGENTS.md): language, spec-driven workflow,
  autonomy, Graphite branches and commits, Conventional Commits. These rules
  apply to the whole repository.
- [`frontend/AGENTS.md`](frontend/AGENTS.md): what is different for the
  frontend (checks, commit scope, security rules, source layout).

## Running things from the root

The root `Makefile` only delegates to `pnpm --dir frontend ...` and
`go -C backend tool mage ...`; `make` lists the targets. Never add logic to
it that belongs in `frontend/package.json` or `backend/magefiles`: add the
script or mage target there and call it from the `Makefile`.

On Windows, run `make` from Git Bash (the `Makefile` stops otherwise).

## Checks before every commit

```sh
make check            # go tool mage ci + pnpm lint, next typegen, tsc --noEmit, pnpm test
```

`make check-backend` or `make check-frontend` is enough when a change only
touches one side. Tasks that change a user flow also run `make e2e` (after
`make seed` once), as `frontend/AGENTS.md` requires.

`check-frontend` needs `API_BASE_URL`: keep `frontend/.env.local` with
`API_BASE_URL=https://localhost:8443`, never staged.

## Commit scope

Changes that only touch root files (`Makefile`, `README.md`, `AGENTS.md`,
`.github/dependabot.yml`) have no scope: `chore: ...`, `docs: ...`,
`ci: ...`. Changes to `.github/workflows/<project>.yml` use that project as
the scope (`ci(frontend): ...`).
