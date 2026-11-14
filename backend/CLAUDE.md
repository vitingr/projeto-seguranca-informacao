# Agent instructions

This file applies to the backend. The git repository is the parent folder
(`meuprojeto/`), which contains `backend/` and `frontend/`. `mage` commands
run inside `backend/`; paths given to `git add` are relative to the
repository root (`backend/...`).

## Language

Everything you write is in **English**, with no exceptions: code
identifiers, comments, log messages, user-facing API messages (the `message`
field of responses), tests, specs, plans, README, branch names and commit
messages.

Existing content written in Portuguese (comments, logs, API messages, specs,
README) must not be translated as part of another task. Translating it is a
task of its own, done in dedicated commits (for example
`refactor: translate comments and logs to english` and
`docs: translate specs and readme to english`). Until then, keep existing
file and folder names unchanged, such as `specs/002-login-e-sessao`, so links
and references do not break.

## Workflow (spec-driven)

Before any implementation, read `specs/constitution.md` and the `spec.md` of
the feature in progress. Implement one task at a time, with tests. If
anything in the spec is ambiguous, ask before writing code.

Run `mage ci` before every commit. Until task 0 of the roadmap (upgrade Go)
is done, a `mage vuln` failure that only reports the Go standard library does
not block the commit: note it in the commit body and continue. Any other
failure blocks.

## Branches, commits and PRs (Graphite)

Always use the `gt` CLI instead of `git commit`, `git switch`,
`git checkout -b` or `gh pr create`. In this file, a "commit" means a
`gt create`.

**Create (at the end of each task, without asking for confirmation):**
1. `mage ci` is green (see the exception above).
2. Check the task's checkbox in its `spec.md`.
3. Run an explicit `git add` with only the task's files, **including** the
   `spec.md` with the checked box.
4. `gt create <branch-name> -m "<type>(<scope>): <description>"`.

**Adjust a task that was already created (before `gt submit`):**
`git add <paths>` followed by `gt modify` (no flags). `gt modify` amends the
current branch's commit and restacks the branches above it; it is the only
allowed way to rewrite a commit.

**Rules:**
- Each spec feature is a stack based on `main`; each task is one branch of
  the stack, with one commit and one PR.
- Branch names are in English, lowercase, kebab-case, prefixed with the
  feature number: `001-add-openapi-contract`, `002-implement-argon2id`.
- Never use `-a`/`--all` with `gt create` or `gt modify`: the repository has
  other folders besides `backend/`.
- Before each `gt create`, run `git status` and confirm that `.env`,
  `certs/`, `bin/` and `coverage.out` are not being added.
- Changes to the **content** of specs and plans (text, criteria, decisions)
  go in their own branch, with a `docs(specs)` commit. Checking off a task's
  checkbox goes together with that task's code.
- To inspect the stack: `gt log short`.
- Never run `gt submit`, `gt sync`, `git push`, `git commit --amend`,
  `git rebase`, `--force` or `gh` unless I ask.
- If `mage ci` fails, do not create the branch: fix it, or stop and explain.

## Commit messages (Conventional Commits)

```
<type>(<scope>): <description>

[optional body: why the change was made, not how]

[optional footer]
```

### Allowed types

| Type | When to use |
|---|---|
| `feat` | new functionality |
| `fix` | bug fix |
| `refactor` | code change that does not alter behavior |
| `test` | tests only |
| `docs` | README, specs, comments |
| `chore` | maintenance (dependencies, configs, tooling) |
| `ci` | GitHub Actions |
| `build` | Dockerfile, docker-compose, go.mod |
| `perf` | performance improvement |
| `security` | security fix or hardening |

### Scope

The path of the changed package or area inside `backend/`, without the
`internal/` prefix: `cmd/server`, `config`, `domain`, `security`,
`repository`, `service`, `web`, `migrations`, `magefiles`, `specs`, `api`,
`bruno`, `docker`. If a change touches several packages, use the main one;
if there is no main one, use `backend`.

### Description rules

- Imperative mood ("add", "fix", "remove")
- All lowercase, no trailing period, at most 72 characters
- Breaking change: `!` after the scope and a footer
  `BREAKING CHANGE: <explanation>`
- Footer `Refs: specs/<feature>` when the commit implements a spec task

### Examples

```
feat(security): implement password hashing with argon2id

Refs: specs/002-login-e-sessao
```

```
fix(web): return 401 instead of 403 for anonymous requests
```

```
chore(magefiles): add bruno target to generate the collection
```

```
feat(web)!: move endpoints under the /api prefix

BREAKING CHANGE: the frontend must call /api/login instead of /login
```