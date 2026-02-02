import axios from 'axios'

const API_BASE_URL = '/api'

export const apiClient = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Add auth token to requests
apiClient.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Handle 401 responses
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

// Types
export interface User {
  id: string
  username: string
  email: string
  role: string
  created_at: string
}

export interface Model {
  id: string
  name: string
  slug: string
  description?: string
  file_path: string
  file_type: string
  file_size?: number
  file_hash?: string
  thumbnail_path?: string
  source?: string
  source_id?: string
  license?: string
  author?: string
  category_id?: string
  category?: Category
  tags?: Tag[]
  indexed_at: string
  created_at: string
  updated_at: string
}

export interface Category {
  id: string
  name: string
  slug: string
  parent_id?: string
  description?: string
  icon?: string
  created_at: string
  children?: Category[]
  model_count?: number
}

export interface Tag {
  id: string
  name: string
  color: string
}

export interface Collection {
  id: string
  user_id: string
  name: string
  description?: string
  is_public: boolean
  created_at: string
  model_count?: number
}

export interface ModelListResponse {
  models: Model[]
  total: number
  page: number
  limit: number
  total_pages: number
}

export interface DuplicateGroup {
  id: string
  primary_model_id: string
  created_at: string
  models: Model[]
}

export interface ScanStatus {
  running: boolean
  started_at?: string
  files_scanned: number
  files_indexed: number
  files_skipped: number
  errors: number
  current_path?: string
  last_error?: string
  completed_at?: string
}

export interface Stats {
  total_models: number
  total_categories: number
  total_tags: number
  total_users: number
  duplicate_groups: number
  models_by_type: Record<string, number>
  models_by_source: Record<string, number>
  uncategorized_count: number
}

// Auth API
export const authApi = {
  register: (data: { username: string; email: string; password: string }) =>
    apiClient.post<{ token: string; user: User }>('/auth/register', data),

  login: (data: { email: string; password: string }) =>
    apiClient.post<{ token: string; user: User }>('/auth/login', data),

  me: () => apiClient.get<User>('/auth/me'),

  refresh: () => apiClient.post<{ token: string }>('/auth/refresh'),
}

// Models API
export const modelsApi = {
  list: (params?: {
    page?: number
    limit?: number
    category?: string
    type?: string
    source?: string
    q?: string
  }) => apiClient.get<ModelListResponse>('/models', { params }),

  get: (id: string) => apiClient.get<Model>(`/models/${id}`),

  update: (id: string, data: Partial<Model>) =>
    apiClient.put(`/models/${id}`, data),

  delete: (id: string) => apiClient.delete(`/models/${id}`),

  setCategory: (id: string, categoryId: string | null) =>
    apiClient.put(`/models/${id}/category`, { category_id: categoryId }),

  addTags: (id: string, tagIds: string[]) =>
    apiClient.post(`/models/${id}/tags`, { tag_ids: tagIds }),

  removeTag: (id: string, tagId: string) =>
    apiClient.delete(`/models/${id}/tags/${tagId}`),

  toggleFavorite: (id: string) =>
    apiClient.post<{ favorited: boolean }>(`/models/${id}/favorite`),

  getFileUrl: (id: string) => `/api/models/${id}/file`,

  getThumbnailUrl: (id: string) => `/api/models/${id}/thumbnail`,
}

// Search API
export const searchApi = {
  search: (q: string) =>
    apiClient.get<{ results: Model[]; count: number }>('/search', { params: { q } }),
}

// Categories API
export const categoriesApi = {
  list: () => apiClient.get<Category[]>('/categories'),

  get: (id: string) => apiClient.get<Category>(`/categories/${id}`),

  create: (data: { name: string; parent_id?: string; description?: string; icon?: string }) =>
    apiClient.post<Category>('/categories', data),

  update: (id: string, data: Partial<Category>) =>
    apiClient.put<Category>(`/categories/${id}`, data),

  delete: (id: string) => apiClient.delete(`/categories/${id}`),
}

// Tags API
export const tagsApi = {
  list: () => apiClient.get<Tag[]>('/tags'),

  create: (data: { name: string; color?: string }) =>
    apiClient.post<Tag>('/tags', data),

  delete: (id: string) => apiClient.delete(`/tags/${id}`),
}

// Collections API
export const collectionsApi = {
  list: () => apiClient.get<Collection[]>('/collections'),

  create: (data: { name: string; description?: string; is_public?: boolean }) =>
    apiClient.post<Collection>('/collections', data),

  addModel: (collectionId: string, modelId: string) =>
    apiClient.post(`/collections/${collectionId}/models`, { model_id: modelId }),

  removeModel: (collectionId: string, modelId: string) =>
    apiClient.delete(`/collections/${collectionId}/models/${modelId}`),

  delete: (id: string) => apiClient.delete(`/collections/${id}`),
}

// Admin User type
export interface AdminUser {
  id: string
  username: string
  email: string
  role: string
  created_at: string
}

// Directory browser types
export interface DirectoryEntry {
  name: string
  path: string
}

export interface BrowseResult {
  current: string
  parent: string
  directories: DirectoryEntry[]
}

// Admin API
export const adminApi = {
  // Settings
  getSettings: () => apiClient.get<Record<string, string>>('/admin/settings'),

  updateSetting: (key: string, value: string) =>
    apiClient.put('/admin/settings', { key, value }),

  getModelPath: () => apiClient.get<{ path: string }>('/admin/settings/model-path'),

  setModelPath: (path: string) =>
    apiClient.put('/admin/settings/model-path', { path }),

  getModelPaths: () => apiClient.get<{ paths: string[] }>('/admin/settings/model-paths'),

  setModelPaths: (paths: string[]) =>
    apiClient.put('/admin/settings/model-paths', { paths }),

  addModelPath: (path: string) =>
    apiClient.post<{ paths: string[] }>('/admin/settings/model-paths/add', { path }),

  removeModelPath: (path: string) =>
    apiClient.post<{ paths: string[] }>('/admin/settings/model-paths/remove', { path }),

  browseDirectory: (path?: string) =>
    apiClient.get<BrowseResult>('/admin/browse', { params: { path } }),

  // User management
  listUsers: () => apiClient.get<AdminUser[]>('/admin/users'),

  createUser: (data: { username: string; email: string; password: string; role: string }) =>
    apiClient.post<AdminUser>('/admin/users', data),

  updateUser: (id: string, data: { username?: string; email?: string; password?: string; role?: string }) =>
    apiClient.put(`/admin/users/${id}`, data),

  deleteUser: (id: string) => apiClient.delete(`/admin/users/${id}`),

  // Database management
  clearDatabase: (confirm: string) =>
    apiClient.post('/admin/clear-database', { confirm }),

  clearModels: (confirm: string) =>
    apiClient.post('/admin/clear-models', { confirm }),

  // Scanner
  startScan: () => apiClient.post('/admin/scan'),

  getScanStatus: () => apiClient.get<ScanStatus>('/admin/scan/status'),

  stopScan: () => apiClient.post('/admin/scan/stop'),

  // Duplicates
  detectDuplicates: () => apiClient.post('/admin/duplicates/detect'),

  listDuplicates: () => apiClient.get<DuplicateGroup[]>('/admin/duplicates'),

  resolveDuplicate: (groupId: string, primaryModelId: string, deleteOthers: boolean) =>
    apiClient.post(`/admin/duplicates/${groupId}/resolve`, {
      primary_model_id: primaryModelId,
      delete_others: deleteOthers,
    }),

  // Stats
  getStats: () => apiClient.get<Stats>('/admin/stats'),
}
