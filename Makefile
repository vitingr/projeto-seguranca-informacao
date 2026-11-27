# Entry point for both projects. Every target only delegates: the frontend
# commands live in frontend/package.json (run with pnpm) and the backend ones
# in backend/magefiles (run with "go tool mage"). Run "make" to list them.

SHELL := bash
.SHELLFLAGS := -eu -o pipefail -c
# Each recipe runs in one shell, so "set -e" stops it at the first failure.
.ONESHELL:
# Prerequisites run one at a time, left to right (infra-up before migrate).
.NOTPARALLEL:
.DEFAULT_GOAL := help
MAKEFLAGS += --no-builtin-rules --no-print-directory --warn-undefined-variables

# GNU make for Windows only honors SHELL := bash when it is started from Git
# Bash; from PowerShell or cmd it silently runs the recipes in cmd.exe.
ifdef OS
ifeq ($(OS),Windows_NT)
ifndef MSYSTEM
$(error On Windows, run make from Git Bash (MSYSTEM is not set, so this is PowerShell or cmd))
endif
endif
endif

PNPM := pnpm --dir frontend
MAGE := go -C backend tool mage

# The process environment wins over frontend/.env.local in Next, and the
# Playwright webServer inherits it, so the e2e target points the frontend at
# the docker compose stack. API_URL (the /api proxy) is set too, in case
# .env.local sets it.
E2E_API_BASE_URL := https://localhost

.PHONY: help \
	install install-frontend install-backend e2e-install \
	dev dev-frontend dev-backend \
	cert infra-up infra-down migrate migrate-down seed \
	lint lint-frontend lint-backend typecheck-frontend \
	test test-frontend test-backend \
	build build-frontend build-backend \
	e2e demo check check-frontend check-backend check-frontend-env

##@ General

help: ## List the targets
	@awk 'BEGIN { FS = ":.*## " }
	  /^##@ / { printf "\n%s\n", substr($$0, 5) }
	  /^[a-zA-Z0-9_-]+:.*## / { printf "  %-20s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

##@ Setup

install: install-frontend install-backend ## Install the frontend and backend dependencies

install-frontend: ## pnpm install in frontend/
	$(PNPM) install

install-backend: ## go mod download in backend/
	go -C backend mod download

e2e-install: ## Install the Chromium used by Playwright
	$(PNPM) e2e:install

##@ Development

dev: check-frontend-env ## Run backend and frontend together (Ctrl+C stops both)
	@set -m
	$(MAKE) dev-backend & backend=$$!
	$(MAKE) dev-frontend & frontend=$$!
	# Stop a whole process tree: taskkill on Windows (Git Bash), the job's
	# process group elsewhere (set -m gives each job its own group).
	stop() {
	  for pid in "$$@"; do
	    kill -0 "$$pid" 2>/dev/null || continue
	    if [ -r "/proc/$$pid/winpid" ]; then
	      taskkill //F //T //PID "$$(cat "/proc/$$pid/winpid")" >/dev/null 2>&1 || true
	    else
	      kill -TERM -- "-$$pid" 2>/dev/null || kill -TERM "$$pid" 2>/dev/null || true
	    fi
	  done
	}
	trap 'stop $$backend $$frontend' EXIT
	trap 'exit 130' INT TERM
	# When either side exits, the EXIT trap stops the other one.
	while kill -0 $$backend 2>/dev/null && kill -0 $$frontend 2>/dev/null; do sleep 1; done
	if ! kill -0 $$backend 2>/dev/null; then name=backend; pid=$$backend; else name=frontend; pid=$$frontend; fi
	status=0; wait $$pid || status=$$?
	echo "make dev: the $$name exited with status $$status; stopping the other side" >&2
	exit $$(( status == 0 ? 1 : status ))

dev-frontend: check-frontend-env ## pnpm dev (Next over HTTPS on https://localhost:3000)
	$(PNPM) dev

dev-backend: ## go tool mage run (backend on the host, https://localhost:8443)
	$(MAGE) run

##@ Infrastructure (Docker, backend/)

cert: ## go tool mage cert (development TLS certificate, kept if it exists)
	$(MAGE) cert

infra-up: cert ## go tool mage dockerUp (PostgreSQL + backend container)
	$(MAGE) dockerUp

infra-down: ## go tool mage dockerDown
	$(MAGE) dockerDown

migrate: ## go tool mage migrateUp
	$(MAGE) migrateUp

migrate-down: ## go tool mage migrateDown (rolls back one migration)
	$(MAGE) migrateDown

seed: ## go tool mage seed (creates the first admin; run it once per database)
	$(MAGE) seed

##@ Quality

lint: lint-frontend lint-backend ## Lint both projects

lint-frontend: ## pnpm lint
	$(PNPM) lint

lint-backend: ## go tool mage lint
	$(MAGE) lint

typecheck-frontend: check-frontend-env ## next typegen + tsc --noEmit
	$(PNPM) exec next typegen
	$(PNPM) exec tsc --noEmit

test: test-frontend test-backend ## Unit tests of both projects

test-frontend: ## pnpm test
	$(PNPM) test

test-backend: ## go tool mage test
	$(MAGE) test

build: build-frontend build-backend ## Build both projects

build-frontend: check-frontend-env ## pnpm build
	$(PNPM) build

build-backend: ## go tool mage build (bin/backend_si)
	$(MAGE) build

e2e: infra-up migrate ## Start the stack, migrate and run pnpm e2e (run make seed once before)
	API_BASE_URL=$(E2E_API_BASE_URL) API_URL=$(E2E_API_BASE_URL) $(PNPM) e2e

demo: infra-up migrate ## Class demo: e2e/enunciado.spec.ts headed and slowed down, then the HTML report
	API_BASE_URL=$(E2E_API_BASE_URL) API_URL=$(E2E_API_BASE_URL) $(PNPM) demo

check: check-backend check-frontend ## Every check required before a commit

check-backend: ## go tool mage ci (fmt, lint, vuln, test)
	$(MAGE) ci

check-frontend: lint-frontend typecheck-frontend test-frontend ## pnpm lint, typegen, tsc and pnpm test

# Next loads src/config/env through next.config.ts, which needs API_BASE_URL.
check-frontend-env:
	@if [ -n "$${API_BASE_URL:-}" ]; then exit 0; fi
	if [ -f frontend/.env.local ] &&
	   grep -Eq '^[[:space:]]*API_BASE_URL[[:space:]]*=[[:space:]]*[^[:space:]#]' frontend/.env.local; then
	  exit 0
	fi
	echo "API_BASE_URL is not set. Create frontend/.env.local (see frontend/.env.example) with" >&2
	echo "  API_BASE_URL=https://localhost:8443" >&2
	echo "the backend of make dev-backend, and leave API_URL unset so the /api proxy uses it too." >&2
	exit 1
