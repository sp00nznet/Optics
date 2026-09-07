# Installation

## Requirements

- Docker with Compose v2
- ~2 GB RAM
- Disk for thumbnails and the Postgres volume

## Install

```bash
git clone https://github.com/sp00nznet/Optics.git
cd Optics
cp .env.example .env
```

Edit `.env` and set at minimum:

```bash
DB_PASSWORD=something_long_and_random
JWT_SECRET=$(openssl rand -hex 32)
```

Then:

```bash
docker compose up -d
docker compose ps          # all three should be up, optics-db healthy
curl localhost:8080/health # {"status":"ok"}
```

Open <http://localhost:3333> and register. **The first account becomes admin** —
register yours before exposing the instance to anyone else.

Migrations run automatically when the backend starts; there is no separate step.

## First scan

`docker-compose.yml` mounts `./sample-models` at `/mnt/3d-models`, so a fresh clone
has something to index. Go to **Admin → Scanner → Start Scan**. You should get 7
models in 5 categories, one of them read from inside a ZIP.

## Using your own library

Point `MODELS_PATH` at your directory and add the production overlay, which swaps
the bind mount:

```bash
MODELS_PATH=/srv/3d-models docker compose \
  -f docker-compose.yml -f docker-compose.prod.yml up -d
```

The mount is read-only — Optics indexes and serves your files, it never writes to
them.

If your models live on a network share, mount it on the **host** first and give
Docker the host path. Optics has no NFS or SMB client of its own.

```bash
# /etc/fstab
nfs-server:/export/3d-models /srv/3d-models nfs defaults,_netdev,ro 0 0
```

Then `MODELS_PATH=/srv/3d-models`. Mount before Docker starts, or the container
will bind an empty directory and the scan will find nothing.

## Ports

Change them in `.env` if 3333 or 8080 are taken:

```bash
FRONTEND_PORT=8090
BACKEND_PORT=8091
```

## Upgrading

```bash
git pull
docker compose up -d --build
```

Data lives in the `postgres_data` and `thumbnails_data` volumes and survives a
rebuild. New migrations apply on backend start.

## Backup

```bash
docker exec optics-db pg_dump -U optics optics > optics-backup.sql
```

Restore into a running stack:

```bash
docker exec -i optics-db psql -U optics optics < optics-backup.sql
```

Your model files are not in the backup — they were never Optics' to begin with.

## Troubleshooting

**Backend exits immediately.** `docker compose logs backend`. Almost always a
missing `JWT_SECRET` or `DB_PASSWORD` in `.env`; the server refuses to start
without either.

**Scan finds nothing.** Check the mount is actually populated inside the container:

```bash
docker compose exec backend ls /mnt/3d-models
```

An empty listing means the host path is wrong or the network share was not mounted
before Docker started.

**Scan skips files.** The scan status panel breaks skips down by extension. Only
the extensions in `scanning.extensions` are indexed — see
[CONFIGURATION.md](CONFIGURATION.md).

**Thumbnails are blank.** They render in your browser with WebGL, not on the
server. Check the browser console; a headless or software-GL environment may need
WebGL explicitly enabled.

**Port already in use.** Change `FRONTEND_PORT` / `BACKEND_PORT` in `.env` and
`docker compose up -d` again.

## Uninstall

```bash
docker compose down -v   # -v also deletes the database and thumbnails
```
