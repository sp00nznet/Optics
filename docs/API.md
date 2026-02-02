# API Reference

Optics provides a RESTful API for managing 3D models. All endpoints are prefixed with `/api`.

## Authentication

Most endpoints require JWT authentication. Include the token in the Authorization header:

```
Authorization: Bearer <token>
```

### Obtain Token

```http
POST /api/auth/login
Content-Type: application/json

{
  "username": "user@example.com",
  "password": "password"
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": "uuid",
    "username": "user",
    "email": "user@example.com",
    "role": "admin"
  }
}
```

---

## Endpoints

### Authentication

#### Register
```http
POST /api/auth/register
```

**Body:**
```json
{
  "username": "newuser",
  "email": "user@example.com",
  "password": "securepassword"
}
```

**Notes:**
- First registered user becomes admin
- Password must be at least 6 characters

#### Login
```http
POST /api/auth/login
```

**Body:**
```json
{
  "email": "user@example.com",
  "password": "password"
}
```

#### Get Current User
```http
GET /api/auth/me
Authorization: Bearer <token>
```

---

### Models

#### List Models
```http
GET /api/models
```

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `page` | int | Page number (default: 1) |
| `limit` | int | Items per page (default: 20, max: 100) |
| `search` | string | Search term |
| `category` | uuid | Filter by category ID |
| `tags` | string | Comma-separated tag IDs |
| `type` | string | Filter by file type (stl, scad, etc.) |
| `sort` | string | Sort field (name, created_at, file_size) |
| `order` | string | Sort order (asc, desc) |

**Response:**
```json
{
  "data": [
    {
      "id": "uuid",
      "name": "Model Name",
      "slug": "model-name-abc123",
      "file_path": "path/to/model.stl",
      "file_type": "stl",
      "file_size": 1234567,
      "source": "thingiverse",
      "source_id": "12345",
      "category": {
        "id": "uuid",
        "name": "Category"
      },
      "tags": [
        {"id": "uuid", "name": "tag", "color": "#6366f1"}
      ],
      "indexed_at": "2024-01-15T10:30:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 150,
    "pages": 8
  }
}
```

#### Get Model
```http
GET /api/models/:id
```

**Response:**
```json
{
  "data": {
    "id": "uuid",
    "name": "Model Name",
    "slug": "model-name-abc123",
    "description": "Model description",
    "file_path": "path/to/model.stl",
    "file_type": "stl",
    "file_size": 1234567,
    "file_hash": "sha256hash",
    "source": "thingiverse",
    "source_id": "12345",
    "author": "Designer Name",
    "license": "CC-BY",
    "category": {...},
    "tags": [...],
    "indexed_at": "2024-01-15T10:30:00Z",
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  }
}
```

#### Download Model File
```http
GET /api/models/:id/file
```

Returns the raw model file with appropriate Content-Type.

**Supports:**
- Regular files from filesystem
- Files inside ZIP archives (virtual path format)

#### Update Model
```http
PUT /api/models/:id
Authorization: Bearer <token>
```

**Body:**
```json
{
  "name": "New Name",
  "description": "Updated description"
}
```

#### Delete Model
```http
DELETE /api/models/:id
Authorization: Bearer <token>
```

Removes model from index (does not delete file).

---

### Categories

#### List Categories
```http
GET /api/categories
```

Returns hierarchical tree of categories.

**Response:**
```json
{
  "data": [
    {
      "id": "uuid",
      "name": "Category Name",
      "slug": "category-name",
      "parent_id": null,
      "model_count": 42,
      "children": [
        {
          "id": "uuid",
          "name": "Subcategory",
          "parent_id": "parent-uuid",
          "model_count": 15,
          "children": []
        }
      ]
    }
  ]
}
```

#### Create Category
```http
POST /api/categories
Authorization: Bearer <token>
```

**Body:**
```json
{
  "name": "New Category",
  "parent_id": "optional-parent-uuid",
  "description": "Category description"
}
```

#### Update Category
```http
PUT /api/categories/:id
Authorization: Bearer <token>
```

#### Delete Category
```http
DELETE /api/categories/:id
Authorization: Bearer <token>
```

#### Assign Model to Category
```http
PUT /api/models/:id/category
Authorization: Bearer <token>
```

**Body:**
```json
{
  "category_id": "uuid"
}
```

---

### Tags

#### List Tags
```http
GET /api/tags
```

**Response:**
```json
{
  "data": [
    {
      "id": "uuid",
      "name": "Tag Name",
      "color": "#6366f1",
      "model_count": 25
    }
  ]
}
```

#### Create Tag
```http
POST /api/tags
Authorization: Bearer <token>
```

**Body:**
```json
{
  "name": "New Tag",
  "color": "#22c55e"
}
```

#### Add Tags to Model
```http
POST /api/models/:id/tags
Authorization: Bearer <token>
```

**Body:**
```json
{
  "tag_ids": ["uuid1", "uuid2"]
}
```

#### Remove Tag from Model
```http
DELETE /api/models/:id/tags/:tagId
Authorization: Bearer <token>
```

---

### Favorites

#### Toggle Favorite
```http
POST /api/models/:id/favorite
Authorization: Bearer <token>
```

Returns updated favorite status.

#### List Favorites
```http
GET /api/favorites
Authorization: Bearer <token>
```

---

### Collections

#### List Collections
```http
GET /api/collections
Authorization: Bearer <token>
```

#### Create Collection
```http
POST /api/collections
Authorization: Bearer <token>
```

**Body:**
```json
{
  "name": "My Collection",
  "description": "Collection description",
  "is_public": false
}
```

#### Add Model to Collection
```http
POST /api/collections/:id/models
Authorization: Bearer <token>
```

**Body:**
```json
{
  "model_id": "uuid"
}
```

---

### Admin

All admin endpoints require authentication with admin role.

#### Start Scan
```http
POST /api/admin/scan
Authorization: Bearer <token>
```

Triggers a full filesystem scan.

#### Get Scan Status
```http
GET /api/admin/scan/status
Authorization: Bearer <token>
```

**Response:**
```json
{
  "running": true,
  "started_at": "2024-01-15T10:30:00Z",
  "files_scanned": 1500,
  "files_indexed": 1200,
  "files_skipped": 300,
  "errors": 5,
  "current_path": "/mnt/3d-models/category/file.stl",
  "last_error": "error message if any"
}
```

#### Stop Scan
```http
POST /api/admin/scan/stop
Authorization: Bearer <token>
```

#### Detect Duplicates
```http
POST /api/admin/duplicates/detect
Authorization: Bearer <token>
```

Runs duplicate detection based on file hashes.

#### List Duplicates
```http
GET /api/admin/duplicates
Authorization: Bearer <token>
```

**Response:**
```json
{
  "data": [
    {
      "id": "uuid",
      "hash": "sha256hash",
      "models": [
        {"id": "uuid", "name": "Model 1", "file_path": "..."},
        {"id": "uuid", "name": "Model 2", "file_path": "..."}
      ]
    }
  ]
}
```

---

## Error Responses

All errors follow this format:

```json
{
  "error": "Error message",
  "code": "ERROR_CODE"
}
```

### Common Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `UNAUTHORIZED` | 401 | Missing or invalid token |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `NOT_FOUND` | 404 | Resource not found |
| `VALIDATION_ERROR` | 400 | Invalid request body |
| `INTERNAL_ERROR` | 500 | Server error |

---

## Rate Limiting

Currently no rate limiting is implemented. For production deployments, consider adding rate limiting via reverse proxy (nginx, Traefik).

---

## Examples

### cURL

```bash
# Login
TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"password"}' \
  | jq -r '.token')

# List models
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/models?limit=10

# Search models
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/models?search=benchy"

# Start scan
curl -X POST -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/admin/scan
```

### JavaScript

```javascript
const api = {
  baseUrl: '/api',
  token: localStorage.getItem('token'),

  async get(path) {
    const res = await fetch(`${this.baseUrl}${path}`, {
      headers: { 'Authorization': `Bearer ${this.token}` }
    });
    return res.json();
  },

  async getModels(params = {}) {
    const query = new URLSearchParams(params).toString();
    return this.get(`/models?${query}`);
  }
};

// Usage
const models = await api.getModels({ search: 'benchy', limit: 20 });
```
