# Configuration Guide

Optics is configured through environment variables. This guide covers all available options.

## Environment Variables

### Database

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `POSTGRES_HOST` | PostgreSQL host | `localhost` | Yes |
| `POSTGRES_PORT` | PostgreSQL port | `5432` | No |
| `POSTGRES_USER` | Database username | `optics` | Yes |
| `POSTGRES_PASSWORD` | Database password | - | Yes |
| `POSTGRES_DB` | Database name | `optics` | Yes |
| `DATABASE_URL` | Full connection string (overrides above) | - | No |

### Authentication

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `JWT_SECRET` | Secret key for JWT signing | - | Yes |
| `JWT_EXPIRY` | Token expiration time | `24h` | No |

Generate a secure JWT secret:
```bash
openssl rand -hex 32
```

### Server

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `SERVER_HOST` | API server bind address | `0.0.0.0` | No |
| `SERVER_PORT` | API server port | `8080` | No |
| `GIN_MODE` | Gin framework mode | `release` | No |

### Storage

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `STORAGE_MOUNT_PATH` | Path to model files | `/mnt/3d-models` | Yes |
| `THUMBNAIL_PATH` | Path for cached thumbnails | `/data/thumbnails` | No |

### Scanning

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `SCAN_EXTENSIONS` | File extensions to index | `stl,scad,obj,3mf,step,stp` | No |
| `SCAN_IGNORE_PATTERNS` | Patterns to skip | `System Volume Information,*.tmp` | No |
| `SCAN_BATCH_SIZE` | Files per batch | `100` | No |

---

## Configuration Files

### docker-compose.yml

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: ${POSTGRES_USER:-optics}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      POSTGRES_DB: ${POSTGRES_DB:-optics}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER:-optics}"]
      interval: 5s
      timeout: 5s
      retries: 5

  backend:
    build: ./backend
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgres://${POSTGRES_USER:-optics}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB:-optics}?sslmode=disable
      - JWT_SECRET=${JWT_SECRET}
      - STORAGE_MOUNT_PATH=/mnt/3d-models
    volumes:
      - ${MODELS_PATH:-./models}:/mnt/3d-models:ro
      - thumbnails:/data/thumbnails
    depends_on:
      postgres:
        condition: service_healthy

  frontend:
    build: ./frontend
    ports:
      - "3333:80"
    depends_on:
      - backend

volumes:
  postgres_data:
  thumbnails:
```

### .env.example

```bash
# Database Configuration
POSTGRES_USER=optics
POSTGRES_PASSWORD=change_me_in_production
POSTGRES_DB=optics

# JWT Configuration
# Generate with: openssl rand -hex 32
JWT_SECRET=your_jwt_secret_here

# Storage Configuration
# Path to your 3D model files
MODELS_PATH=/mnt/3d-models

# Optional: Additional model paths (comma-separated)
# ADDITIONAL_PATHS=/mnt/lego-models,/mnt/custom-models
```

---

## Frontend Configuration

The frontend is configured at build time through Vite.

### vite.config.ts

```typescript
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
```

### API Base URL

In production, the frontend uses relative paths (`/api`). For development:

```typescript
// src/api/client.ts
const API_BASE = import.meta.env.VITE_API_URL || '/api'
```

---

## Scanning Configuration

### Supported Extensions

Default extensions that will be indexed:
- `.stl` - Stereolithography
- `.scad` - OpenSCAD source
- `.obj` - Wavefront OBJ
- `.3mf` - 3D Manufacturing Format
- `.step` / `.stp` - STEP CAD format

### Ignore Patterns

Default patterns that will be skipped:
- `System Volume Information`
- `*.tmp`
- `*.bak`
- `.DS_Store`
- `Thumbs.db`

### Archive Support

ZIP files are automatically scanned. Models inside archives use virtual paths:
```
archive.zip!path/inside/model.stl
```

---

## Performance Tuning

### Database Connection Pool

```go
// backend/internal/database/postgres.go
config.MaxConns = 25
config.MinConns = 5
config.MaxConnLifetime = time.Hour
```

### Thumbnail Caching

Thumbnails are rendered client-side and cached in memory. The service uses a single WebGL context to avoid browser limits.

### Large Libraries

For libraries with 100,000+ models:
1. Increase PostgreSQL `shared_buffers`
2. Add database indexes (already included in schema)
3. Consider pagination limits in API calls

---

## Security Configuration

### JWT Settings

```go
// Token expiration
JWT_EXPIRY=24h

// For longer sessions
JWT_EXPIRY=168h  // 7 days
```

### Password Requirements

Passwords are hashed using bcrypt with cost factor 10.

### CORS Configuration

```go
// backend/internal/router/router.go
r.Use(cors.New(cors.Config{
    AllowOrigins:     []string{"*"},
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
    AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
    AllowCredentials: true,
}))
```

For production, restrict `AllowOrigins` to your domain.

---

## Logging

### Backend Logs

```bash
# View all logs
docker compose logs -f backend

# Filter by level
docker compose logs backend 2>&1 | grep -i error
```

### Log Levels

Set via environment:
```bash
GIN_MODE=debug    # Verbose logging
GIN_MODE=release  # Production logging
```

---

## Backup Configuration

### Database Backup

```bash
# Backup
docker compose exec postgres pg_dump -U optics optics > backup.sql

# Restore
docker compose exec -T postgres psql -U optics optics < backup.sql
```

### Volume Backup

```bash
# Backup thumbnails
docker run --rm -v optics_thumbnails:/data -v $(pwd):/backup alpine \
    tar czf /backup/thumbnails.tar.gz -C /data .
```

---

## Healthcheck Configuration

### Docker Healthchecks

```yaml
services:
  backend:
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/api/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
```

### Monitoring Endpoints

- `GET /api/health` - Basic health check
- `GET /api/admin/scan/status` - Scan status (requires auth)
