# Development Guide

This guide covers setting up a local development environment for Optics.

## Prerequisites

- Go 1.21+
- Node.js 20+
- PostgreSQL 16
- Git

## Quick Start

### 1. Clone Repository

```bash
git clone https://github.com/sp00nznet/Optics.git
cd Optics
```

### 2. Start Database

```bash
# Using Docker
docker run -d \
  --name optics-postgres \
  -e POSTGRES_USER=optics \
  -e POSTGRES_PASSWORD=optics \
  -e POSTGRES_DB=optics \
  -p 5432:5432 \
  postgres:16-alpine

# Or use existing PostgreSQL
createdb optics
```

### 3. Backend Setup

```bash
cd backend

# Install dependencies
go mod download

# Set environment variables
export DATABASE_URL="postgres://optics:optics@localhost:5432/optics?sslmode=disable"
export JWT_SECRET="development-secret-change-in-prod"
export STORAGE_MOUNT_PATH="./test-models"

# Run with hot reload (install air first)
go install github.com/cosmtrek/air@latest
air

# Or run directly
go run ./cmd/server
```

### 4. Frontend Setup

```bash
cd frontend

# Install dependencies
npm install

# Start development server
npm run dev
```

### 5. Access Application

- Frontend: http://localhost:5173 (Vite dev server)
- Backend: http://localhost:8080

---

## Project Structure

### Backend

```
backend/
├── cmd/
│   └── server/
│       └── main.go           # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go         # Configuration management
│   ├── database/
│   │   └── postgres.go       # Database connection & migrations
│   ├── handlers/
│   │   ├── admin.go          # Admin endpoints
│   │   ├── auth.go           # Authentication
│   │   ├── categories.go     # Category CRUD
│   │   └── models.go         # Model CRUD
│   ├── middleware/
│   │   └── auth.go           # JWT middleware
│   ├── models/
│   │   ├── category.go       # Category model
│   │   ├── model.go          # Model model
│   │   └── user.go           # User model
│   ├── router/
│   │   └── router.go         # Route definitions
│   └── services/
│       ├── duplicates.go     # Duplicate detection
│       ├── indexer.go        # Model indexing
│       ├── scanner.go        # Filesystem scanner
│       └── thumbnail.go      # Thumbnail generation
├── go.mod
├── go.sum
└── Dockerfile
```

### Frontend

```
frontend/
├── src/
│   ├── api/
│   │   └── client.ts         # API client
│   ├── components/
│   │   ├── CategoryTree.tsx  # Category navigation
│   │   ├── Layout.tsx        # App layout
│   │   ├── ModelCard.tsx     # Model card component
│   │   ├── ModelGrid.tsx     # Model grid
│   │   ├── ModelThumbnail.tsx# 3D thumbnail
│   │   ├── ModelViewer.tsx   # 3D model viewer
│   │   ├── ScadViewer.tsx    # OpenSCAD code viewer
│   │   ├── SearchBar.tsx     # Search component
│   │   └── TagManager.tsx    # Tag management
│   ├── hooks/
│   │   └── useDebounce.ts    # Debounce hook
│   ├── pages/
│   │   ├── Admin.tsx         # Admin panel
│   │   ├── Browse.tsx        # Model browser
│   │   ├── Categories.tsx    # Category management
│   │   ├── Duplicates.tsx    # Duplicate viewer
│   │   ├── Home.tsx          # Home page
│   │   ├── Login.tsx         # Login page
│   │   ├── ModelDetail.tsx   # Model detail view
│   │   └── Register.tsx      # Registration
│   ├── services/
│   │   └── thumbnailService.ts # Thumbnail rendering
│   ├── store/
│   │   └── auth.ts           # Auth state (Zustand)
│   ├── App.tsx               # Root component
│   └── main.tsx              # Entry point
├── package.json
├── vite.config.ts
└── Dockerfile
```

---

## Development Workflow

### Adding a New API Endpoint

1. **Define the handler** in `backend/internal/handlers/`
   ```go
   func (h *Handler) NewEndpoint(c *gin.Context) {
       // Implementation
   }
   ```

2. **Add the route** in `backend/internal/router/router.go`
   ```go
   api.GET("/new-endpoint", handler.NewEndpoint)
   ```

3. **Add API client method** in `frontend/src/api/client.ts`
   ```typescript
   async newEndpoint(): Promise<Response> {
       return this.get('/new-endpoint')
   }
   ```

### Adding a New Component

1. Create component in `frontend/src/components/`
2. Export from component file
3. Import where needed

### Database Migrations

Migrations are embedded in `backend/internal/database/postgres.go`:

```go
const schema = `
CREATE TABLE IF NOT EXISTS new_table (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL
);
`
```

For production, consider using a migration tool like `golang-migrate`.

---

## Testing

### Backend Tests

```bash
cd backend
go test ./...

# With coverage
go test -cover ./...

# Verbose
go test -v ./...
```

### Frontend Tests

```bash
cd frontend

# Run tests
npm test

# Watch mode
npm run test:watch

# Coverage
npm run test:coverage
```

### E2E Tests

```bash
# Start services
docker compose up -d

# Run Playwright tests
npm run test:e2e
```

---

## Code Style

### Go

Follow standard Go conventions:
- Use `gofmt` for formatting
- Follow [Effective Go](https://golang.org/doc/effective_go)
- Run `go vet` before commits

```bash
gofmt -w .
go vet ./...
```

### TypeScript

Follow the project ESLint configuration:

```bash
npm run lint
npm run lint:fix
```

---

## Building for Production

### Backend

```bash
cd backend
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server
```

### Frontend

```bash
cd frontend
npm run build
```

### Docker

```bash
docker compose build
docker compose up -d
```

---

## Debugging

### Backend

Use delve for debugging:

```bash
go install github.com/go-delve/delve/cmd/dlv@latest
dlv debug ./cmd/server
```

VS Code launch configuration:
```json
{
  "name": "Debug Backend",
  "type": "go",
  "request": "launch",
  "mode": "debug",
  "program": "${workspaceFolder}/backend/cmd/server"
}
```

### Frontend

Use browser DevTools or VS Code debugger:

```json
{
  "name": "Debug Frontend",
  "type": "chrome",
  "request": "launch",
  "url": "http://localhost:5173"
}
```

---

## Common Tasks

### Reset Database

```bash
docker compose down -v
docker compose up -d postgres
# Wait for postgres to start
docker compose up -d
```

### Clear Thumbnail Cache

Thumbnails are cached in browser memory. Clear by:
1. Hard refresh (Ctrl+Shift+R)
2. Clear browser cache
3. Call `thumbnailService.clearCache()` in console

### Add Test Models

Create `test-models/` directory and add sample STL files:

```bash
mkdir -p test-models/category1 test-models/category2
# Add .stl files to these directories
```

---

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests
5. Submit a pull request

### Commit Messages

Follow conventional commits:
- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation
- `refactor:` Code refactoring
- `test:` Adding tests
- `chore:` Maintenance

Example:
```
feat: add duplicate detection by visual similarity
```

---

## Troubleshooting

### Port already in use

```bash
# Find process
lsof -i :8080
lsof -i :5173

# Kill process
kill -9 <PID>
```

### Database connection issues

```bash
# Check PostgreSQL is running
docker ps | grep postgres

# Test connection
psql -h localhost -U optics -d optics
```

### CORS errors

Ensure the backend CORS configuration allows your frontend origin:

```go
// router.go
AllowOrigins: []string{"http://localhost:5173", "http://localhost:3333"}
```
