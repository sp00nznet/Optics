# Architecture Overview

This document describes the high-level architecture of Optics.

## System Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Client (Browser)                                │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────────────────┐│
│  │   React     │ │  Three.js   │ │  Zustand    │ │    TanStack Query       ││
│  │   Router    │ │  3D Viewer  │ │   Store     │ │    (Data Fetching)      ││
│  └─────────────┘ └─────────────┘ └─────────────┘ └─────────────────────────┘│
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      │ HTTP/REST
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Nginx (Reverse Proxy)                           │
│                         Port 3333 (or 80 in production)                      │
└─────────────────────────────────────────────────────────────────────────────┘
                          │                           │
                          │ Static Files              │ /api/*
                          ▼                           ▼
┌─────────────────────────────────┐   ┌──────────────────────────────────────┐
│       Frontend Container        │   │          Backend Container            │
│   ┌─────────────────────────┐  │   │   ┌────────────────────────────────┐  │
│   │   Built React App       │  │   │   │         Gin Framework          │  │
│   │   (Static HTML/JS/CSS)  │  │   │   │                                │  │
│   └─────────────────────────┘  │   │   │  ┌──────┐ ┌──────┐ ┌────────┐ │  │
│                                 │   │   │  │ Auth │ │Models│ │ Admin  │ │  │
│                                 │   │   │  └──────┘ └──────┘ └────────┘ │  │
│                                 │   │   │                                │  │
│                                 │   │   │  ┌─────────────────────────┐  │  │
│                                 │   │   │  │      Services           │  │  │
│                                 │   │   │  │  Scanner | Indexer      │  │  │
│                                 │   │   │  │  Duplicates | Thumbnail │  │  │
│                                 │   │   │  └─────────────────────────┘  │  │
│                                 │   │   └────────────────────────────────┘  │
└─────────────────────────────────┘   └──────────────────────────────────────┘
                                                       │
                                      ┌────────────────┼────────────────┐
                                      │                │                │
                                      ▼                ▼                ▼
                            ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
                            │  PostgreSQL  │  │   NFS Mount  │  │  Thumbnails  │
                            │   Database   │  │  (3D Models) │  │    Cache     │
                            │              │  │              │  │              │
                            │  - users     │  │  /mnt/       │  │  /data/      │
                            │  - models    │  │  3d-models/  │  │  thumbnails/ │
                            │  - categories│  │              │  │              │
                            │  - tags      │  │  *.stl       │  │              │
                            └──────────────┘  │  *.scad      │  └──────────────┘
                                              │  *.zip       │
                                              └──────────────┘
```

## Component Details

### Frontend

Built with React 18 and TypeScript, bundled with Vite.

| Component | Purpose |
|-----------|---------|
| **React Router** | Client-side routing between pages |
| **Three.js** | WebGL-based 3D model rendering |
| **Zustand** | Lightweight state management for auth |
| **TanStack Query** | Server state management and caching |
| **Tailwind CSS** | Utility-first styling |
| **Lucide React** | Icon library |

#### Key Components

- **ModelViewer**: Interactive 3D viewer using Three.js with OrbitControls
- **ScadViewer**: Displays OpenSCAD code or converts STL to polyhedron
- **ModelThumbnail**: Renders 3D thumbnails using shared WebGL context
- **ThumbnailService**: Singleton service managing thumbnail generation

### Backend

Built with Go and the Gin web framework.

| Layer | Responsibility |
|-------|----------------|
| **Handlers** | HTTP request/response handling |
| **Middleware** | Authentication, CORS, logging |
| **Services** | Business logic (scanning, indexing) |
| **Models** | Data structures and database queries |
| **Config** | Environment configuration |

#### Services

- **Scanner**: Walks filesystem, indexes new files, handles ZIP archives
- **Indexer**: Creates database records for discovered models
- **Duplicates**: Detects duplicate files using SHA-256 hashes
- **Thumbnail**: (Future) Server-side thumbnail generation

### Database Schema

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│     users       │     │    categories   │     │      tags       │
├─────────────────┤     ├─────────────────┤     ├─────────────────┤
│ id (PK)         │     │ id (PK)         │     │ id (PK)         │
│ username        │     │ name            │     │ name            │
│ email           │     │ slug            │     │ color           │
│ password_hash   │     │ parent_id (FK)  │     └────────┬────────┘
│ role            │     │ description     │              │
└────────┬────────┘     └────────┬────────┘              │
         │                       │                       │
         │              ┌────────┴────────┐              │
         │              ▼                 │              │
         │     ┌─────────────────┐        │     ┌────────┴────────┐
         │     │     models      │        │     │   model_tags    │
         │     ├─────────────────┤        │     ├─────────────────┤
         │     │ id (PK)         │◄───────┘     │ model_id (FK)   │
         │     │ name            │◄─────────────│ tag_id (FK)     │
         │     │ slug            │              └─────────────────┘
         │     │ file_path       │
         │     │ file_type       │
         │     │ file_size       │
         │     │ file_hash       │
         │     │ source          │
         │     │ category_id(FK) │
         │     └────────┬────────┘
         │              │
         │     ┌────────┴────────┐
         │     ▼                 ▼
┌────────┴────────┐     ┌─────────────────┐
│   favorites     │     │  collections    │
├─────────────────┤     ├─────────────────┤
│ user_id (FK)    │     │ id (PK)         │
│ model_id (FK)   │     │ user_id (FK)    │
└─────────────────┘     │ name            │
                        │ is_public       │
                        └─────────────────┘
```

## Data Flow

### Model Scanning

```
Filesystem           Scanner              Database
    │                   │                    │
    │   Walk files      │                    │
    │ ◄─────────────────│                    │
    │                   │                    │
    │   File metadata   │                    │
    │──────────────────►│                    │
    │                   │                    │
    │                   │   Check exists     │
    │                   │───────────────────►│
    │                   │                    │
    │                   │   Insert model     │
    │                   │───────────────────►│
    │                   │                    │
```

### Model Viewing

```
Browser              Nginx              Backend            Storage
   │                   │                   │                  │
   │  GET /api/models  │                   │                  │
   │──────────────────►│                   │                  │
   │                   │  Proxy to :8080   │                  │
   │                   │──────────────────►│                  │
   │                   │                   │                  │
   │                   │   Query DB        │                  │
   │                   │                   │                  │
   │                   │◄──────────────────│                  │
   │◄──────────────────│                   │                  │
   │                   │                   │                  │
   │  GET /models/:id/file                 │                  │
   │──────────────────►│──────────────────►│                  │
   │                   │                   │   Read file      │
   │                   │                   │─────────────────►│
   │                   │                   │◄─────────────────│
   │◄──────────────────│◄──────────────────│                  │
   │                   │                   │                  │
   │  Render in Three.js                   │                  │
   │                   │                   │                  │
```

### Thumbnail Generation

```
ModelCard          ThumbnailService         Three.js
    │                    │                      │
    │  getThumbnail(url) │                      │
    │───────────────────►│                      │
    │                    │                      │
    │                    │  Check cache         │
    │                    │                      │
    │                    │  Load STL            │
    │                    │─────────────────────►│
    │                    │                      │
    │                    │  Render to canvas    │
    │                    │◄─────────────────────│
    │                    │                      │
    │                    │  toDataURL()         │
    │                    │                      │
    │  dataUrl           │                      │
    │◄───────────────────│                      │
    │                    │                      │
    │  Display as <img>  │                      │
    │                    │                      │
```

## Security

### Authentication Flow

1. User submits credentials
2. Backend validates against bcrypt hash
3. JWT token issued with 24h expiration
4. Token stored in localStorage
5. Subsequent requests include Bearer token
6. Middleware validates token on protected routes

### Authorization

| Role | Capabilities |
|------|--------------|
| **user** | Browse, search, favorites, collections |
| **admin** | All user capabilities + scan, delete, manage |

## Performance Considerations

### Frontend

- **Thumbnail caching**: Rendered thumbnails cached as data URLs
- **Single WebGL context**: Prevents browser context limits
- **Lazy loading**: Models loaded on demand
- **Pagination**: Large lists paginated

### Backend

- **Connection pooling**: pgx pool manages database connections
- **Read-only mounts**: Models served directly from NFS
- **Streaming**: Large files streamed, not buffered in memory
- **Batch scanning**: Files processed in batches

### Database

- **Indexed columns**: category_id, file_hash, file_type
- **Full-text search**: PostgreSQL tsvector on name
- **UUID primary keys**: Distributed-friendly identifiers

## Deployment Topology

### Single Host (Development)

```
┌─────────────────────────────────────┐
│           Docker Host               │
│  ┌─────────┐ ┌─────────┐ ┌───────┐ │
│  │Frontend │ │ Backend │ │Postgres│ │
│  │  :3333  │ │  :8080  │ │ :5432 │ │
│  └─────────┘ └─────────┘ └───────┘ │
│              │                      │
│              ▼                      │
│       /mnt/3d-models (bind mount)   │
└─────────────────────────────────────┘
```

### Production (Proxmox LXC)

```
┌─────────────────────────────────────┐
│         Proxmox LXC Container       │
│  ┌─────────────────────────────────┐│
│  │         Docker Engine           ││
│  │  ┌─────────┐ ┌─────────┐       ││
│  │  │Frontend │ │ Backend │       ││
│  │  └─────────┘ └─────────┘       ││
│  └─────────────────────────────────┘│
│              │                      │
│              ▼                      │
│       /mnt/3d-models (NFS mount)    │
│              │                      │
└──────────────│──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│          NAS / File Server          │
│       /pool/3d-models               │
│       /pool/lego-models             │
└─────────────────────────────────────┘
```

## Future Enhancements

- **Server-side thumbnails**: Pre-render and cache PNG thumbnails
- **Visual similarity**: Compare thumbnails for near-duplicates
- **Full-text search**: Add Meilisearch for better fuzzy matching
- **WebSocket**: Real-time scan progress updates
- **Multi-tenant**: Organization support
- **Model conversion**: Convert between formats
- **STL repair**: Automatic mesh repair
