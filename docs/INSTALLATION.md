# Installation Guide

This guide covers deploying Optics in various environments.

## Table of Contents
- [Docker Deployment](#docker-deployment)
- [NFS Mount Configuration](#nfs-mount-configuration)
- [Proxmox LXC Deployment](#proxmox-lxc-deployment)
- [Manual Installation](#manual-installation)
- [Reverse Proxy Setup](#reverse-proxy-setup)

---

## Docker Deployment

### Prerequisites
- Docker Engine 20.10+
- Docker Compose v2+
- 2GB RAM minimum
- Network storage accessible via NFS or local path

### Quick Start

1. **Clone and configure**
   ```bash
   git clone https://github.com/sp00nznet/Optics.git
   cd Optics
   cp .env.example .env
   ```

2. **Edit `.env` file**
   ```bash
   # Database
   POSTGRES_USER=optics
   POSTGRES_PASSWORD=your_secure_password
   POSTGRES_DB=optics

   # JWT Secret (generate with: openssl rand -hex 32)
   JWT_SECRET=your_jwt_secret_here

   # Storage path (local or mount point)
   MODELS_PATH=/path/to/your/models
   ```

3. **Start services**
   ```bash
   docker compose up -d
   ```

4. **Verify deployment**
   ```bash
   docker compose ps
   docker compose logs -f
   ```

### Docker Compose Configuration

The default `docker-compose.yml` includes:
- **postgres**: PostgreSQL 16 database
- **backend**: Go API server on port 8080
- **frontend**: Nginx serving React app on port 3333

To customize ports, edit `docker-compose.yml`:
```yaml
services:
  frontend:
    ports:
      - "80:80"  # Change 3333 to 80
```

---

## NFS Mount Configuration

### On the Docker Host

1. **Install NFS client**
   ```bash
   # Ubuntu/Debian
   apt-get install nfs-common

   # RHEL/CentOS
   yum install nfs-utils
   ```

2. **Create mount point**
   ```bash
   mkdir -p /mnt/3d-models
   ```

3. **Add to /etc/fstab**
   ```
   nfs-server:/path/to/share /mnt/3d-models nfs defaults,_netdev 0 0
   ```

4. **Mount the share**
   ```bash
   mount -a
   ```

5. **Update docker-compose.yml**
   ```yaml
   services:
     backend:
       volumes:
         - /mnt/3d-models:/mnt/3d-models:ro
   ```

### Multiple Mount Points

To add multiple model directories:

```yaml
services:
  backend:
    volumes:
      - /mnt/3d-models:/mnt/3d-models:ro
      - /mnt/lego-models:/mnt/lego-models:ro
    environment:
      - STORAGE_MOUNT_PATH=/mnt/3d-models
      # Additional paths can be scanned via admin panel
```

---

## Proxmox LXC Deployment

### Create Container

1. **Download template** (Ubuntu 22.04 recommended)

2. **Create LXC container**
   - ID: Choose available ID
   - Hostname: optics
   - Disk: 20GB minimum
   - CPU: 2 cores
   - RAM: 2048MB
   - Network: DHCP or static IP

3. **Container configuration**
   ```bash
   # Enable nesting for Docker
   pct set <CTID> -features nesting=1
   ```

### Install Docker in LXC

```bash
# Update system
apt-get update && apt-get upgrade -y

# Install Docker
curl -fsSL https://get.docker.com | sh

# Install Docker Compose
apt-get install docker-compose-plugin -y

# Start Docker
systemctl enable docker
systemctl start docker
```

### Deploy Optics

```bash
# Clone repository
cd /opt
git clone https://github.com/sp00nznet/Optics.git
cd Optics

# Configure
cp .env.example .env
nano .env  # Edit settings

# Mount NFS (see NFS section above)
mkdir -p /mnt/3d-models
mount -t nfs nfs-server:/share /mnt/3d-models

# Start
docker compose up -d
```

### Persist NFS Mount

Add to `/etc/fstab`:
```
nfs-server:/share /mnt/3d-models nfs defaults,_netdev 0 0
```

---

## Manual Installation

### Backend (Go)

1. **Install Go 1.21+**
   ```bash
   wget https://go.dev/dl/go1.21.linux-amd64.tar.gz
   tar -C /usr/local -xzf go1.21.linux-amd64.tar.gz
   export PATH=$PATH:/usr/local/go/bin
   ```

2. **Build backend**
   ```bash
   cd backend
   go mod download
   go build -o server ./cmd/server
   ```

3. **Run backend**
   ```bash
   export DATABASE_URL="postgres://user:pass@localhost:5432/optics"
   export JWT_SECRET="your-secret"
   export STORAGE_MOUNT_PATH="/mnt/3d-models"
   ./server
   ```

### Frontend (Node.js)

1. **Install Node.js 20+**
   ```bash
   curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
   apt-get install -y nodejs
   ```

2. **Build frontend**
   ```bash
   cd frontend
   npm install
   npm run build
   ```

3. **Serve with Nginx**
   ```bash
   cp -r dist/* /var/www/html/
   ```

### PostgreSQL

1. **Install PostgreSQL 16**
   ```bash
   apt-get install postgresql-16
   ```

2. **Create database**
   ```sql
   CREATE USER optics WITH PASSWORD 'your_password';
   CREATE DATABASE optics OWNER optics;
   ```

---

## Reverse Proxy Setup

### Nginx

```nginx
server {
    listen 80;
    server_name optics.example.com;

    location / {
        proxy_pass http://localhost:3333;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }

    location /api {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### Traefik (Docker)

```yaml
labels:
  - "traefik.enable=true"
  - "traefik.http.routers.optics.rule=Host(`optics.example.com`)"
  - "traefik.http.services.optics.loadbalancer.server.port=3333"
```

### Caddy

```
optics.example.com {
    reverse_proxy localhost:3333

    handle_path /api/* {
        reverse_proxy localhost:8080
    }
}
```

---

## Post-Installation

### First-Time Setup

1. **Register admin account**
   - Navigate to http://your-server:3333/register
   - First registered user automatically becomes admin

2. **Start initial scan**
   - Go to Admin panel
   - Click "Start Scan"
   - Monitor progress in scan status

3. **Organize your library**
   - Create categories
   - Add tags
   - Assign models to categories

### Health Check

Verify all services are running:

```bash
# Check containers
docker compose ps

# Check API health
curl http://localhost:8080/api/health

# Check frontend
curl -I http://localhost:3333
```

---

## Troubleshooting

### Container won't start
```bash
docker compose logs backend
docker compose logs frontend
```

### NFS mount issues
```bash
# Test mount manually
mount -t nfs -o vers=4 server:/share /mnt/test

# Check permissions
ls -la /mnt/3d-models
```

### Database connection failed
```bash
# Check PostgreSQL is running
docker compose logs postgres

# Verify connection
docker compose exec postgres psql -U optics -d optics
```

### Models not appearing
- Verify mount path in `.env` matches container volume
- Check scanner logs: `docker compose logs backend | grep -i scan`
- Ensure file extensions are in supported list (stl, scad, obj, 3mf)
