# API Reference

Base URL `http://localhost:8080`. Everything except `/health` lives under `/api`.

## Authentication

`POST /api/auth/login` returns a JWT. Send it as `Authorization: Bearer <token>`.

```http
POST /api/auth/login
Content-Type: application/json

{ "email": "you@example.com", "password": "your-password" }
```

Login is **by email**, not username. Register takes all three:

```http
POST /api/auth/register
Content-Type: application/json

{ "username": "you", "email": "you@example.com", "password": "your-password" }
```

Both return `{ "token": "...", "user": { "id", "username", "email", "role", "created_at" } }`.
The first account to register becomes `admin`; everyone after is `user`.

## Access levels

| Level | Meaning |
|-------|---------|
| public | no token needed |
| optional | works without a token; a token adds per-user data (favourites) |
| auth | any logged-in user |
| admin | `role = admin` |

## Endpoints

### Auth

| Method | Path | Access |
|--------|------|--------|
| POST | `/api/auth/register` | public |
| POST | `/api/auth/login` | public |
| POST | `/api/auth/refresh` | auth |
| GET | `/api/auth/me` | auth |

### Models

| Method | Path | Access |
|--------|------|--------|
| GET | `/api/models` | optional |
| GET | `/api/models/:id` | optional |
| GET | `/api/models/:id/file` | public |
| GET | `/api/models/:id/thumbnail` | public |
| PUT | `/api/models/:id` | auth |
| DELETE | `/api/models/:id` | admin |
| PUT | `/api/models/:id/category` | auth |
| POST | `/api/models/:id/tags` | auth |
| DELETE | `/api/models/:id/tags/:tagId` | auth |
| POST | `/api/models/:id/favorite` | auth |
| GET | `/api/search?q=` | optional |

`GET /api/models` query parameters:

| Param | Type | Notes |
|-------|------|-------|
| `category` | UUID | category ID, not slug |
| `tags` | UUID list | repeat the param |
| `type` | string | `stl`, `scad`, `obj`, … |
| `source` | string | e.g. `thingiverse` |
| `q` | string | matches name and description |
| `page` | int | 1-based, default 1 |
| `limit` | int | default 20, max 100 |

Response: `{ "models": [...], "total": n, "page": n, "limit": n }`.

`GET /api/models/:id/file` streams the model. For a file indexed inside an archive
(`file_path` of the form `archive.zip!inner/path.stl`) it is streamed out of the ZIP
without extracting it.

### Categories and tags

| Method | Path | Access |
|--------|------|--------|
| GET | `/api/categories` | public |
| GET | `/api/categories/:id` | public |
| POST | `/api/categories` | admin |
| PUT | `/api/categories/:id` | admin |
| DELETE | `/api/categories/:id` | admin |
| GET | `/api/tags` | public |
| POST | `/api/tags` | admin |
| DELETE | `/api/tags/:id` | admin |

### Collections

All require auth.

| Method | Path |
|--------|------|
| GET | `/api/collections` |
| POST | `/api/collections` |
| DELETE | `/api/collections/:id` |
| POST | `/api/collections/:id/models` |
| DELETE | `/api/collections/:id/models/:modelId` |

### Admin

All require admin.

| Method | Path | Notes |
|--------|------|-------|
| GET | `/api/admin/stats` | totals, counts by type and source |
| POST | `/api/admin/scan` | starts a scan, returns `202` |
| GET | `/api/admin/scan/status` | progress, skip reasons, error log |
| POST | `/api/admin/scan/stop` | |
| POST | `/api/admin/duplicates/detect` | idempotent; returns `202` |
| GET | `/api/admin/duplicates` | groups with their members |
| POST | `/api/admin/duplicates/:id/resolve` | body: `primary_model_id`, `delete_others` |
| GET | `/api/admin/settings` | |
| PUT | `/api/admin/settings` | body: `key`, `value` |
| GET / PUT | `/api/admin/settings/model-path` | single scan path |
| GET / PUT | `/api/admin/settings/model-paths` | multiple scan paths |
| POST | `/api/admin/settings/model-paths/add` | |
| POST | `/api/admin/settings/model-paths/remove` | |
| GET | `/api/admin/browse?path=` | directory browser for picking scan paths |
| GET / POST | `/api/admin/users` | list / create |
| PUT / DELETE | `/api/admin/users/:id` | |
| POST | `/api/admin/clear-models` | drops indexed models, keeps users |
| POST | `/api/admin/clear-database` | drops everything |

### Health

`GET /health` → `{"status":"ok"}`. No auth, not under `/api`.

## Scan status

```json
{
  "running": false,
  "files_scanned": 12,
  "files_indexed": 7,
  "files_skipped": 4,
  "skip_reasons": {
    "wrong_extension": 4,
    "already_indexed": 0,
    "extension_counts": { ".txt": 2, ".md": 1, ".png": 1 }
  },
  "errors": 0,
  "error_log": [],
  "started_at": "...",
  "completed_at": "..."
}
```

## Errors

Errors are `{ "error": "message" }` with the usual codes: `400` bad input, `401`
missing/invalid token, `403` not admin, `404` not found, `409` username or email
already taken.
