# Configuration

Two layers, and it matters which is which:

- **`.env`** is read by Docker Compose. It sets the container's ports, the Postgres
  credentials, and the secrets that get passed through.
- **`OPTICS_*` variables** are read by the Go backend itself. `docker-compose.yml`
  builds these from `.env`; you only set them by hand when running the backend
  outside Docker.

## .env (Docker Compose)

| Variable | Default | Notes |
|----------|---------|-------|
| `DB_USER` | `optics` | Postgres user |
| `DB_PASSWORD` | — | **required** |
| `DB_NAME` | `optics` | |
| `DB_PORT` | `5432` | host port for Postgres |
| `JWT_SECRET` | — | **required**; `openssl rand -hex 32` |
| `BACKEND_PORT` | `8080` | host port for the API |
| `FRONTEND_PORT` | `3333` | host port for the web UI |

`MODELS_PATH` is used only by `docker-compose.prod.yml`, to bind-mount your own
library in place of `sample-models/`.

## OPTICS_* (backend)

The backend reads these directly. Names map to the config keys with `.` → `_`.

| Variable | Default |
|----------|---------|
| `OPTICS_SERVER_HOST` | `0.0.0.0` |
| `OPTICS_SERVER_PORT` | `8080` |
| `OPTICS_DATABASE_HOST` | `localhost` |
| `OPTICS_DATABASE_PORT` | `5432` |
| `OPTICS_DATABASE_NAME` | `optics` |
| `OPTICS_DATABASE_USER` | `optics` |
| `OPTICS_DATABASE_PASSWORD` | — (**required**, no default) |
| `OPTICS_DATABASE_SSLMODE` | `disable` |
| `OPTICS_JWT_SECRET` | — (**required**, no default) |
| `OPTICS_JWT_EXPIRY` | `24h` |
| `OPTICS_STORAGE_MOUNT_PATH` | `/mnt/3d-models` |
| `OPTICS_STORAGE_THUMBNAIL_PATH` | `/data/thumbnails` |

The server refuses to start without `OPTICS_JWT_SECRET` and
`OPTICS_DATABASE_PASSWORD`.

## config.yaml

Instead of environment variables the backend will read a `config.yaml` from the
working directory, `./config/`, or `/etc/optics/`. Environment variables win over
the file. See [`backend/config.yaml.example`](../backend/config.yaml.example). This
is mainly useful for the scanning options, which have no environment equivalent:

```yaml
scanning:
  extensions: [stl, scad, obj, 3mf, step, stp]
  ignore_patterns:
    - "System Volume Information"
    - "*.tmp"
  batch_size: 100
```

`ignore_patterns` are matched against directory *names* with Go's
[`filepath.Match`](https://pkg.go.dev/path/filepath#Match); a match skips the whole
subtree.

## Scan paths

`OPTICS_STORAGE_MOUNT_PATH` is the default scan root. Extra roots can be added at
runtime from **Admin → Settings**, which stores them in the database — no restart
needed. Anything you add must be visible inside the backend container, so mount it
in `docker-compose.yml` first.

## Frontend

The frontend needs no configuration. It calls `/api` relatively, and nginx in the
frontend container proxies that to the backend — so there is no API URL to set and
no CORS to arrange. In dev, `vite.config.ts` proxies `/api` to `localhost:8080`
instead.
