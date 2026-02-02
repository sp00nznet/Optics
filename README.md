# Optics

<div align="center">

![Optics Logo](https://img.shields.io/badge/Optics-3D%20Model%20Library-blue?style=for-the-badge&logo=data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSIyNCIgaGVpZ2h0PSIyNCIgdmlld0JveD0iMCAwIDI0IDI0IiBmaWxsPSJub25lIiBzdHJva2U9IiNmZmZmZmYiIHN0cm9rZS13aWR0aD0iMiIgc3Ryb2tlLWxpbmVjYXA9InJvdW5kIiBzdHJva2UtbGluZWpvaW49InJvdW5kIj48cGF0aCBkPSJtMjEuNjQgMy42NC0xLjI4LTEuMjhhMS4yMSAxLjIxIDAgMCAwLTEuNzIgMEwyLjM2IDE4LjY0YTEuMjEgMS4yMSAwIDAgMCAwIDEuNzJsMS4yOCAxLjI4YTEuMiAxLjIgMCAwIDAgMS43MiAwTDIxLjY0IDUuMzZhMS4yIDEuMiAwIDAgMCAwLTEuNzJ6Ii8+PHBhdGggZD0ibTE0IDdsMyAzIi8+PHBhdGggZD0iTTUgNnYxNGMwIDEuMSA5IDEuMSA5IDB2LTE0YzAtMS4xLTktMS4xLTkgMHoiLz48L3N2Zz4=)

**A powerful web-based 3D model library manager for browsing, organizing, and managing large collections of STL, SCAD, and other 3D model files.**

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go)](https://golang.org)
[![React](https://img.shields.io/badge/React-18+-61DAFB?style=flat-square&logo=react)](https://reactjs.org)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-336791?style=flat-square&logo=postgresql)](https://postgresql.org)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat-square&logo=docker)](https://docker.com)
[![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)](LICENSE)

[Features](#features) | [Quick Start](#quick-start) | [Documentation](#documentation) | [Screenshots](#screenshots)

</div>

---

## Overview

Optics is designed to help makers, designers, and 3D printing enthusiasts manage large collections of 3D models. Whether you have a curated collection or a massive Thingiverse archive, Optics makes it easy to browse, search, organize, and preview your models.

### Key Highlights

- **Interactive 3D Previews** - View STL models directly in your browser with full rotation and zoom
- **OpenSCAD Integration** - View SCAD source code or convert STL files to OpenSCAD polyhedrons
- **Smart Organization** - Categories, tags, and collections to keep your library organized
- **Archive Support** - Scan and serve models from within ZIP files
- **Duplicate Detection** - Find duplicate files using SHA-256 hashing
- **Fast Search** - Full-text search across your entire library

---

## Features

### 3D Model Viewer
- **Real-time 3D rendering** using Three.js
- **STL file support** with proper Z-up to Y-up orientation
- **Interactive controls** - orbit, zoom, pan
- **Grid floor** for scale reference
- **Ambient and directional lighting** for realistic previews

### OpenSCAD Code Viewer
- **Side-by-side layout** with 3D preview and code
- **STL to OpenSCAD conversion** - generates polyhedron() code from STL geometry
- **Native SCAD file viewing** with syntax highlighting
- **One-click copy** to clipboard
- **Import generation** for OBJ and 3MF files

### Library Management
- **Automatic scanning** of model directories
- **ZIP archive support** - index and serve models inside archives
- **Category auto-creation** from folder structure
- **Thingiverse metadata parsing** - extracts Thing IDs from filenames and README files
- **Bulk organization tools**

### Organization Features
- **Hierarchical categories** for logical grouping
- **Colored tags** for flexible labeling
- **User collections** for personal curation
- **Favorites** to bookmark frequently used models

### Search & Discovery
- **Full-text search** across model names and descriptions
- **Filter by category, tags, or file type**
- **Sort by name, date, or size**
- **Paginated results** for large libraries

### Admin Features
- **Scan management** - trigger and monitor library scans
- **Duplicate detection** - identify identical files
- **User management** - first registered user becomes admin
- **Scan status monitoring** with real-time progress

### Thumbnail Generation
- **Automatic 3D thumbnails** rendered in browser
- **Shared WebGL context** prevents browser limits
- **Cached rendering** for performance
- **Fallback placeholders** for non-STL files

---

## Quick Start

### Prerequisites
- Docker and Docker Compose
- 2GB RAM minimum
- Network access to your model storage (NFS/SMB mount)

### Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/sp00nznet/Optics.git
   cd Optics
   ```

2. **Configure environment**
   ```bash
   cp .env.example .env
   # Edit .env with your settings
   ```

3. **Start the services**
   ```bash
   docker compose up -d
   ```

4. **Access the application**
   - Frontend: http://localhost:3333
   - API: http://localhost:8080

5. **Create your account**
   - Register at http://localhost:3333/register
   - First user automatically becomes admin

6. **Start scanning**
   - Go to Admin panel
   - Click "Start Scan" to index your models

For detailed setup instructions, see [docs/INSTALLATION.md](docs/INSTALLATION.md).

---

## Documentation

| Document | Description |
|----------|-------------|
| [Installation Guide](docs/INSTALLATION.md) | Detailed setup instructions |
| [Configuration](docs/CONFIGURATION.md) | Environment variables and options |
| [API Reference](docs/API.md) | REST API documentation |
| [Development](docs/DEVELOPMENT.md) | Contributing and local development |
| [Architecture](docs/ARCHITECTURE.md) | System design and components |

---

## Screenshots

### Model Browser
Browse your library with 3D thumbnail previews, category filters, and search.

### Model Detail View
Side-by-side 3D viewer and OpenSCAD code panel with model metadata.

### Admin Panel
Manage scans, view statistics, and detect duplicates.

---

## Tech Stack

| Component | Technology |
|-----------|------------|
| **Backend** | Go 1.21+ with Gin framework |
| **Frontend** | React 18 + TypeScript + Vite |
| **Database** | PostgreSQL 16 |
| **3D Rendering** | Three.js with STLLoader |
| **Styling** | Tailwind CSS |
| **State Management** | Zustand + TanStack Query |
| **Authentication** | JWT with bcrypt |
| **Containerization** | Docker + Docker Compose |

---

## Supported File Types

| Extension | Preview | OpenSCAD Export |
|-----------|---------|-----------------|
| `.stl` | 3D Viewer | Polyhedron conversion |
| `.scad` | Code view | Native display |
| `.obj` | Placeholder | Import statement |
| `.3mf` | Placeholder | Import statement |
| `.step/.stp` | Placeholder | - |

---

## Project Structure

```
Optics/
├── backend/                 # Go backend
│   ├── cmd/server/         # Application entry point
│   └── internal/
│       ├── config/         # Configuration management
│       ├── database/       # PostgreSQL connection
│       ├── handlers/       # HTTP request handlers
│       ├── middleware/     # Auth middleware
│       ├── models/         # Data models
│       ├── router/         # Route definitions
│       └── services/       # Business logic
├── frontend/               # React frontend
│   └── src/
│       ├── api/           # API client
│       ├── components/    # React components
│       ├── hooks/         # Custom hooks
│       ├── pages/         # Page components
│       ├── services/      # Frontend services
│       └── store/         # State management
├── docs/                   # Documentation
├── docker-compose.yml      # Container orchestration
└── .env.example           # Environment template
```

---

## Contributing

Contributions are welcome! Please read [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) for guidelines.

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## Acknowledgments

- [Three.js](https://threejs.org/) for 3D rendering
- [Gin](https://gin-gonic.com/) for the Go web framework
- [Tailwind CSS](https://tailwindcss.com/) for styling
- The 3D printing community for inspiration

---

<div align="center">

**Built with love for the maker community**

[Report Bug](https://github.com/sp00nznet/Optics/issues) | [Request Feature](https://github.com/sp00nznet/Optics/issues)

</div>
