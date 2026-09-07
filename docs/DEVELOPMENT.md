# Development

## Requirements

Go 1.21+, Node 20+, Docker (for Postgres).

## Running outside Docker

Postgres in a container, backend and frontend on the host:

```bash
docker compose up -d postgres
```

Backend:

```bash
cd backend
go mod download

export OPTICS_DATABASE_HOST=localhost
export OPTICS_DATABASE_PASSWORD=optics_dev_password_2024   # must match .env
export OPTICS_JWT_SECRET=dev-secret-not-for-production
export OPTICS_STORAGE_MOUNT_PATH=../sample-models
export OPTICS_STORAGE_THUMBNAIL_PATH=/tmp/optics-thumbnails

go run ./cmd/server
```

Frontend:

```bash
cd frontend
npm install
npm run dev
```

Vite serves on **<http://localhost:3000>** and proxies `/api` to `localhost:8080`
(see `vite.config.ts`). The backend listens on 8080.

Note the variable names: the backend reads `OPTICS_*`, while `.env` holds the
Compose-level `DB_*` names. See [CONFIGURATION.md](CONFIGURATION.md).

## Layout

```
backend/
  cmd/server/main.go              entry point: config, migrations, router, shutdown
  internal/config/                viper config, OPTICS_* env overrides
  internal/database/postgres.go   pool + embedded schema migrations
  internal/handlers/              admin, auth, categories, models
  internal/middleware/auth.go     JWT, RequireAuth / RequireAdmin / OptionalAuth
  internal/models/                request, response and DB structs
  internal/router/router.go       every route, in one place
  internal/services/
    scanner.go                    filesystem + ZIP walk, category creation
    duplicates.go                 SHA-256 grouping, resolve

frontend/src/
  api/client.ts                   axios instance, token interceptor, types
  components/                     Layout, ModelCard/Grid/Thumbnail, ModelViewer,
                                  ScadViewer, CategoryTree, SearchBar, TagManager
  pages/                          Home, Browse, ModelDetail, Categories,
                                  Duplicates, Admin, Login, Register
  services/thumbnailService.ts    shared WebGL context, queued STL thumbnails
  store/auth.ts                   Zustand + persist

test/smoke.mjs                    headless end-to-end run + screenshots
```

## Things worth knowing

**Migrations** are an embedded SQL string in `internal/database/postgres.go`, run on
every start. They must stay idempotent — `CREATE TABLE IF NOT EXISTS`, and additive
`ALTER`s guarded the same way.

**Adding an endpoint**: handler in `internal/handlers/`, route in
`internal/router/router.go` (all routes live there, with their access level), client
method and types in `frontend/src/api/client.ts`.

**Thumbnails** render in the browser, not on the server. Every STL goes through the
single queued WebGL context in `thumbnailService.ts` — browsers cap concurrent
contexts, so don't create renderers per component.

**Archive paths.** A file inside a ZIP is stored as `archive.zip!inner/path.stl`.
Anything that touches `file_path` has to handle that separator; `ServeFile` splits on
it and streams from the archive.

**CORS** is a hand-rolled middleware at the bottom of `router.go` that allows `*`.
In Docker it is unused — nginx proxies `/api` same-origin.

## Checks

```bash
cd backend && gofmt -l . && go vet ./... && go build ./...
cd frontend && npm run lint && npm run build
```

`npm run build` type-checks with `tsc` first, so it catches what `lint` won't.

There are no unit tests. The end-to-end check is `test/smoke.mjs`, which drives the
real stack in a headless browser:

```bash
docker compose up -d
cd test && npm install && npx playwright install chromium
npm run smoke
```

It registers a user, scans `sample-models/`, visits every page, and fails on a
console error, a failed request, a 5xx, an unexpected redirect, or a view that never
finished rendering. It writes `docs/screenshots/`, so the README screenshots refresh
whenever it runs. Point it elsewhere with `OPTICS_URL` and `OPTICS_API`.

## Reset

```bash
docker compose down -v && docker compose up -d
```

Wipes the database and thumbnails. The first account you register afterwards is
admin again.
