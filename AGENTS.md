# AGENTS.md

## 1. Project Overview

- A chat application monorepo with a Go backend and a SvelteKit frontend. No root README exists.
- `backend/` — Go 1.26.5 (per `go.mod`), HTTP API using `github.com/go-chi/chi/v5`, `jackc/pgx/v5`, JWT auth (`golang-jwt/jwt/v5`), and sqlc-generated database code (PostgreSQL).
- `frontend/` — SvelteKit (Svelte 5 runes mode), TypeScript (strict), Vite, Vitest with Playwright browser provider, Prettier, ESLint.
- PostgreSQL is the only datastore. Infra is defined in `docker-compose.yml` (Postgres `postgres:latest` + backend image). No CI config, Makefile, or task runner exists in the repo.
- Runtime: Go toolchain must support `go 1.26.5` (`go vet`/`go build` verified with Go 1.27.1). Node/npm version is NOT pinned (no `.nvmrc`, no `engines` field).

## 2. Setup

Prerequisites: Go, Node.js, npm, Docker with compose plugin.

```sh
# 1. Install Go dependencies
cd backend
go mod download        # [NETWORK]

# 2. Install frontend dependencies (requires /home/hermitk/Projects/Bluenite to exist)
cd ../frontend
npm ci                 # [NETWORK] runs `prepare` (svelte-kit sync) automatically

# 3. Start Postgres + backend (DB schema auto-applies on first boot)
cd ..
docker compose up --build   # [NETWORK] pulls postgres image, builds backend
```

- Backend reads all config from environment variables: `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE`, `JWT_SECRET` (fatal if unset), `PORT` (default `8080`), `CORS_ORIGINS` (comma-separated; defaults to `http://localhost:5173,http://localhost:5174`). `docker-compose.yml` provides dev values for all of them.
- Host port for Postgres is `5433` (mapped to container `5432`). Use `localhost:5433` when connecting locally.
- sqlc is required only when SQL queries/schema change; the generated code in `backend/internal/db/` is committed, so a plain `go build` does not need sqlc.
- No `.env` files or `.env.example` exist; environment is supplied via `docker-compose.yml`.

## 3. Development Commands

Backend (run from `backend/`):

```sh
go run ./cmd/server                  # start API locally (requires Postgres + env vars)
go build ./...                       # build all packages
go vet ./...                         # static checks
go test ./...                        # run tests (none exist yet)
sqlc generate                        # regenerate DB code from sql/queries + sql/schema (requires sqlc)
```

Frontend (run from `frontend/`):

```sh
npm run dev                          # Vite dev server (http://localhost:5173)
npm run build                        # production build (vite build)
npm run preview                      # preview production build
npm run check                        # typecheck via svelte-check
npm run lint                         # prettier --check . && eslint .
npm run format                       # prettier --write .
npm test                             # vitest run (client browser + server projects)
npm run test:unit -- --watch         # vitest watch mode
```

Full stack (from repo root):

```sh
docker compose up --build            # [NETWORK] Postgres + backend; backend on http://localhost:8080
curl http://localhost:8080/health    # expect {"status":"OK"}
```

## 4. Verification

Run all of the following before claiming a task is complete:

```sh
cd backend && go build ./... && go vet ./... && go test ./...
cd ../frontend && npm run check && npm run lint && npm test
docker compose up --build && curl -fsS http://localhost:8080/health
```

If SQL queries or schema changed, `sqlc generate` MUST also pass and its output MUST be committed.

## 5. Architecture

- `backend/cmd/server/main.go` — entry point. Reads `PORT`, calls `db.Connect` (env-based DSN via pgxpool, pings DB), builds the chi router, serves on `:PORT`.
- `backend/internal/routes/` — all HTTP handlers and routing (`routes.go` builds the `chi.Mux`). Public rate-limited routes: `POST /register`, `POST /login`, `POST /token/refresh`. JWT-protected group: `/users/me/*` (get user, rooms, settings, patch settings, logout) and `/rooms/*` (create, participants, messages CRUD). In-memory per-IP rate limiter (`rateLimit`) guards the auth endpoints; CORS origins come from `CORS_ORIGINS`.
- `backend/internal/db/` — sqlc-generated code (`db.go`, `models.go`, `querier.go`, `*_sql.go`), plus hand-written `connect.go`. Source of truth is `backend/sql/queries/*.sql` + `backend/sql/schema/001_init.sql` (config: `backend/sqlc.yaml`, pgx/v5, snake_case JSON tags).
- `backend/internal/auth/jwt.go` — JWT middleware and signing; `backend/internal/utils/utils.go` — `WriteJSON`/`WriteError` helpers; `backend/internal/constants/constants.go` — `UserIDContextKey` and `USER_COLORS`.
- Schema: `users`, `user_settings`, `rooms`, `room_participants`, `messages` (with `reply_to_id`), `refresh_tokens` (stores SHA-256 token hashes). Applied on first Postgres boot via `/docker-entrypoint-initdb.d` mount of `backend/sql/schema`.
- `frontend/src/hooks.server.ts` — auth gate for every request. Validates the `access_token` cookie's `exp` (`src/lib/server/auth.ts`), refreshes it via `POST /token/refresh` when expired, redirects `303` to `/login` when unauthenticated. Public routes: `/login`, `/register` (`src/lib/constants.ts`).
- `frontend/src/lib/api.ts` — `Api` class wrapping `fetch` with `credentials: 'include'`; singleton `api = new Api('http://localhost:8080')` (hardcoded base URL). Auth state is a cookie (`access_token`, `httpOnly: false`, `sameSite: strict`).
- `frontend/src/routes/` — `+page.svelte` (chat UI), `(protected)/...` (settings), `(public)/login`, `(public)/register`; `src/lib/components/` — `ChatArea.svelte`, `Sidebar.svelte`; `src/lib/types.ts`, `src/lib/utils.ts` (cookie helpers).
- Data flow: browser → SvelteKit server hook (auth check) → `Api` → Go API (`:8080`) → sqlc queries → PostgreSQL. `docker-compose.yml` wires Postgres schema init, backend env, and the `messaging_network`.
- SvelteKit is configured inside `frontend/vite.config.ts` via the `sveltekit({...})` plugin (compilerOptions force runes mode, adapter-auto). No `svelte.config.js` exists.

## 6. Code Conventions

- Go: router built in `routes.go` by composing `chi` middleware; handlers are `func(...) http.HandlerFunc` taking `*db.Queries` (and `secret` where needed). Respond with `utils.WriteJSON`/`utils.WriteError`; always set HTTP status codes. Read env at startup; `log.Fatal` on missing required config (`JWT_SECRET`). Wrap errors with `%w`.
- SQL: queries live in `backend/sql/queries/*.sql` with named parameters (`sqlc`-style, e.g. `$1`/`@name`); regenerate with `sqlc generate`; NEVER hand-edit `backend/internal/db/*.go` generated files — edit the SQL and regenerate.
- Frontend: Svelte 5 runes mode is enforced by the Vite config — write runes-based components, not legacy lifecycle code. Prettier settings are tabs, single quotes, `trailingComma: 'es5'`, `printWidth: 100` (see `prettier.config.js`); ESLint extends prettier, so format before linting.
- TypeScript: strict mode; `src/lib/types.ts` is the shared contract for API request/response shapes.
- Tests: none exist yet. Vitest requires assertions in every test (`expect.requireAssertions: true`). Component tests follow `src/**/*.svelte.{test,spec}.{js,ts}` (client project, headless Chromium via Playwright; `src/lib/server/**` excluded); non-component tests follow `src/**/*.{test,spec}.{js,ts}` (node environment). Backend test convention is `*_test.go` with `go test ./...`: TODO: verify backend test framework/patterns.
- Auth: store only token hashes in DB (`refresh_tokens.token_hash`); JWTs carry `user_id`; the frontend refreshes via the server hook, not per-request.

## 7. Agent Rules

- MUST run the Verification section before claiming completion.
- MUST NOT hand-edit sqlc-generated files in `backend/internal/db/`; regenerate from SQL instead.
- MUST NOT commit real secrets. Values in `docker-compose.yml` (`POSTGRES_PASSWORD`, `JWT_SECRET`) are dev-only; never promote them to production, never add new secrets in cleartext to tracked files.
- MUST NOT modify files outside this repo, including `/home/hermitk/Projects/Bluenite` (the external `@hermitk/bluenite` dependency). Do not edit `frontend/node_modules/`.
- MUST NOT alter `frontend/package-lock.json` except through `npm ci`/`npm install` when dependencies actually change; keep the lockfile in sync.
- MUST NOT commit `.env` files (gitignored). Configuration changes belong in `docker-compose.yml` or documented env vars.
- MAY run `sqlc generate` only after editing `backend/sql/` files; commit the regenerated output.
- MAY run `docker compose up --build` to exercise the stack, but MUST NOT leave containers running when the task is done (`docker compose down`).
- Never remove or restructure `docker-compose.yml` healthcheck, schema-init mount, or network wiring without testing the full stack.
- The working tree currently contains uncommitted modifications (see `git status`); treat them as user work — do not discard or overwrite without confirmation.

## 8. Change Workflow

- Branch from `main` for feature work. Current default branch is `main`. TODO: verify branch naming convention (no convention documented in repo).
- Commits follow a conventional prefix from history (`feat:`, `fix:`), one logical change per commit. Example: `feat: implement user settings management and authentication flow`.
- No CI or PR automation is configured in this repo (no `.github/`). TODO: verify PR/merge expectations with the maintainer.
- Update docs/files when the interface changes: API routes ↔ `frontend/src/lib/api.ts` and `src/lib/types.ts` MUST be kept in sync with `backend/internal/routes/`; env vars and ports change → update `docker-compose.yml`; schema/query changes → regenerate sqlc code and keep `backend/sql/` authoritative.
- Add tests for new behavior. Frontend logic → vitest per the projects in `vite.config.ts`; backend logic → `*_test.go` files (none exist yet).
- Prefer atomic, reviewable diffs; run `git status` before and after to confirm only intended files changed.

## 9. Troubleshooting

- `backend` starts but fails to connect to DB: Postgres not running or env not set. Start it with `docker compose up -d postgres` and set `DB_HOST=localhost`, `DB_PORT=5433` (host port differs from container `5432`) for local runs.
- `go: no Go version` or toolchain errors: Go MUST be at least 1.26.x; `go.mod` declares `go 1.26.5`.
- Frontend `npm ci`/`npm install` fails on `@hermitk/bluenite`: the directory `/home/hermitk/Projects/Bluenite` MUST exist; the installed package is a symlink under `frontend/node_modules/@hermitk/bluenite`.
- `npm run check` fails with missing `.svelte-kit`: run `npm run prepare` (or `npx svelte-kit sync`) first; `npm ci` runs it automatically.
- Vitest browser tests fail with "browser not found": install the Playwright Chromium binary with `npx playwright install chromium` [NETWORK].
- Port conflicts: API expects `8080`; dev Vite server `5173` (CORS also allows `5174`); Postgres on host `5433`.
- Wrong/cached DB state: `docker compose down -v` [DESTRUCTIVE] wipes the `postgres_data` volume, then `docker compose up --build` re-applies `backend/sql/schema/001_init.sql`.
- `JWT_SECRET is not set` fatal at startup: export `JWT_SECRET` or rely on `docker-compose.yml`.
- Stale sqlc output after SQL edits: run `sqlc generate` in `backend/` (install sqlc with `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest` [NETWORK] if missing).

## 10. Definition of Done

- [ ] `cd backend && go build ./...` exits 0.
- [ ] `cd backend && go vet ./...` exits 0.
- [ ] `cd backend && go test ./...` exits 0 (passes even with no test files).
- [ ] `cd frontend && npm run check` exits 0.
- [ ] `cd frontend && npm run lint` exits 0 (prettier --check and eslint both clean).
- [ ] `cd frontend && npm test` exits 0 (vitest runs client and server projects).
- [ ] `docker compose up --build` succeeds and `curl http://localhost:8080/health` returns `{"status":"OK"}`.
- [ ] If `backend/sql/` changed: `sqlc generate` ran cleanly, regenerated `backend/internal/db/` is committed, and schema changes are reflected in `docker-compose.yml`'s init mount.
- [ ] API route changes are mirrored in `frontend/src/lib/api.ts` and `frontend/src/lib/types.ts`.
- [ ] New behavior has tests (frontend: vitest; backend: `*_test.go` where applicable).
- [ ] No secrets added to tracked files; no `.env` committed; no `node_modules` or generated build artifacts committed.
- [ ] New env vars/ports documented where they are introduced; README/docs updated if user-facing behavior changed.
- [ ] `git status` shows only intended changes; no destructive commands were run.