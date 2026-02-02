import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { adminApi, ScanStatus, AdminUser } from '../api/client'
import {
  Play,
  Square,
  RefreshCw,
  FileBox,
  FolderTree,
  Users,
  Copy,
  AlertCircle,
  Trash2,
  Settings,
  FolderOpen,
  UserPlus,
  Edit2,
  Check,
  X,
  AlertTriangle,
  Database,
} from 'lucide-react'

type AdminTab = 'overview' | 'settings' | 'users' | 'scanner' | 'database'

export default function Admin() {
  const [activeTab, setActiveTab] = useState<AdminTab>('overview')

  return (
    <div className="max-w-6xl mx-auto space-y-6">
      <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
        Admin Dashboard
      </h1>

      {/* Tabs */}
      <div className="border-b border-gray-200 dark:border-gray-700">
        <nav className="flex gap-4">
          <TabButton active={activeTab === 'overview'} onClick={() => setActiveTab('overview')}>
            Overview
          </TabButton>
          <TabButton active={activeTab === 'settings'} onClick={() => setActiveTab('settings')}>
            <Settings className="w-4 h-4 mr-1" />
            Settings
          </TabButton>
          <TabButton active={activeTab === 'users'} onClick={() => setActiveTab('users')}>
            <Users className="w-4 h-4 mr-1" />
            Users
          </TabButton>
          <TabButton active={activeTab === 'scanner'} onClick={() => setActiveTab('scanner')}>
            <FolderOpen className="w-4 h-4 mr-1" />
            Scanner
          </TabButton>
          <TabButton active={activeTab === 'database'} onClick={() => setActiveTab('database')}>
            <Database className="w-4 h-4 mr-1" />
            Database
          </TabButton>
        </nav>
      </div>

      {/* Tab Content */}
      {activeTab === 'overview' && <OverviewTab />}
      {activeTab === 'settings' && <SettingsTab />}
      {activeTab === 'users' && <UsersTab />}
      {activeTab === 'scanner' && <ScannerTab />}
      {activeTab === 'database' && <DatabaseTab />}
    </div>
  )
}

function TabButton({ active, onClick, children }: { active: boolean; onClick: () => void; children: React.ReactNode }) {
  return (
    <button
      onClick={onClick}
      className={`flex items-center px-4 py-2 border-b-2 font-medium text-sm transition-colors ${
        active
          ? 'border-primary-600 text-primary-600'
          : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
      }`}
    >
      {children}
    </button>
  )
}

function OverviewTab() {
  const { data: stats } = useQuery({
    queryKey: ['admin', 'stats'],
    queryFn: () => adminApi.getStats(),
  })

  return (
    <div className="space-y-6">
      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          icon={<FileBox className="w-6 h-6" />}
          label="Total Models"
          value={stats?.data?.total_models || 0}
        />
        <StatCard
          icon={<FolderTree className="w-6 h-6" />}
          label="Categories"
          value={stats?.data?.total_categories || 0}
        />
        <StatCard
          icon={<Users className="w-6 h-6" />}
          label="Users"
          value={stats?.data?.total_users || 0}
        />
        <StatCard
          icon={<Copy className="w-6 h-6" />}
          label="Duplicate Groups"
          value={stats?.data?.duplicate_groups || 0}
        />
      </div>

      {/* Models by Type */}
      {stats?.data?.models_by_type && Object.keys(stats.data.models_by_type).length > 0 && (
        <div className="card p-6">
          <h2 className="text-lg font-medium mb-4">Models by File Type</h2>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            {Object.entries(stats.data.models_by_type).map(([type, count]) => (
              <div key={type} className="p-4 bg-gray-50 dark:bg-gray-700 rounded-lg">
                <p className="text-2xl font-bold text-gray-900 dark:text-white">
                  {count.toLocaleString()}
                </p>
                <p className="text-sm text-gray-500 uppercase">{type}</p>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Uncategorized Models */}
      {stats?.data?.uncategorized_count && stats.data.uncategorized_count > 0 && (
        <div className="card p-6 border-yellow-200 bg-yellow-50 dark:bg-yellow-900/20 dark:border-yellow-800">
          <div className="flex items-start gap-4">
            <AlertCircle className="w-6 h-6 text-yellow-600 flex-shrink-0" />
            <div>
              <h3 className="font-medium text-yellow-800 dark:text-yellow-200">
                Uncategorized Models
              </h3>
              <p className="text-sm text-yellow-700 dark:text-yellow-300 mt-1">
                {stats.data.uncategorized_count.toLocaleString()} models have not been assigned to a category.
              </p>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

function SettingsTab() {
  const queryClient = useQueryClient()
  const [modelPath, setModelPath] = useState('')
  const [isEditing, setIsEditing] = useState(false)

  const { data: pathData, isLoading } = useQuery({
    queryKey: ['admin', 'model-path'],
    queryFn: async () => {
      const result = await adminApi.getModelPath()
      setModelPath(result.data.path)
      return result
    },
  })

  const updatePathMutation = useMutation({
    mutationFn: (path: string) => adminApi.setModelPath(path),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'model-path'] })
      setIsEditing(false)
    },
  })

  if (isLoading) {
    return <div className="text-center py-8 text-gray-500">Loading settings...</div>
  }

  return (
    <div className="space-y-6">
      <div className="card p-6">
        <h2 className="text-lg font-medium mb-4">Model Storage Path</h2>
        <p className="text-sm text-gray-500 mb-4">
          Set the path to your 3D model files. This can be a local path or a network share.
        </p>

        {isEditing ? (
          <div className="flex gap-2">
            <input
              type="text"
              value={modelPath}
              onChange={(e) => setModelPath(e.target.value)}
              className="input flex-1"
              placeholder="/mnt/3d-models or //server/share"
            />
            <button
              onClick={() => updatePathMutation.mutate(modelPath)}
              disabled={updatePathMutation.isPending}
              className="btn btn-primary"
            >
              <Check className="w-4 h-4" />
            </button>
            <button
              onClick={() => {
                setModelPath(pathData?.data.path || '')
                setIsEditing(false)
              }}
              className="btn btn-secondary"
            >
              <X className="w-4 h-4" />
            </button>
          </div>
        ) : (
          <div className="flex items-center gap-4">
            <code className="flex-1 p-3 bg-gray-100 dark:bg-gray-700 rounded text-sm">
              {pathData?.data.path || 'Not set'}
            </code>
            <button onClick={() => setIsEditing(true)} className="btn btn-secondary">
              <Edit2 className="w-4 h-4 mr-1" />
              Edit
            </button>
          </div>
        )}
      </div>
    </div>
  )
}

function UsersTab() {
  const queryClient = useQueryClient()
  const [showCreateForm, setShowCreateForm] = useState(false)
  const [editingUser, setEditingUser] = useState<AdminUser | null>(null)

  const { data: users, isLoading } = useQuery({
    queryKey: ['admin', 'users'],
    queryFn: () => adminApi.listUsers(),
  })

  const createMutation = useMutation({
    mutationFn: (data: { username: string; email: string; password: string; role: string }) =>
      adminApi.createUser(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'users'] })
      setShowCreateForm(false)
    },
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: { username?: string; email?: string; password?: string; role?: string } }) =>
      adminApi.updateUser(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'users'] })
      setEditingUser(null)
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => adminApi.deleteUser(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'users'] })
    },
  })

  if (isLoading) {
    return <div className="text-center py-8 text-gray-500">Loading users...</div>
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h2 className="text-lg font-medium">User Management</h2>
        <button
          onClick={() => setShowCreateForm(true)}
          className="btn btn-primary flex items-center gap-2"
        >
          <UserPlus className="w-4 h-4" />
          Create User
        </button>
      </div>

      {/* Create User Form */}
      {showCreateForm && (
        <UserForm
          onSubmit={(data) => createMutation.mutate(data as { username: string; email: string; password: string; role: string })}
          onCancel={() => setShowCreateForm(false)}
          isLoading={createMutation.isPending}
        />
      )}

      {/* Edit User Form */}
      {editingUser && (
        <UserForm
          user={editingUser}
          onSubmit={(data) => updateMutation.mutate({ id: editingUser.id, data })}
          onCancel={() => setEditingUser(null)}
          isLoading={updateMutation.isPending}
        />
      )}

      {/* Users Table */}
      <div className="card overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50 dark:bg-gray-700">
            <tr>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Username</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Email</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Role</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Created</th>
              <th className="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
            {users?.data?.map((user) => (
              <tr key={user.id}>
                <td className="px-4 py-3 font-medium">{user.username}</td>
                <td className="px-4 py-3 text-gray-500">{user.email}</td>
                <td className="px-4 py-3">
                  <span className={`px-2 py-1 text-xs rounded-full ${
                    user.role === 'admin'
                      ? 'bg-purple-100 text-purple-700 dark:bg-purple-900 dark:text-purple-300'
                      : 'bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-300'
                  }`}>
                    {user.role}
                  </span>
                </td>
                <td className="px-4 py-3 text-sm text-gray-500">
                  {new Date(user.created_at).toLocaleDateString()}
                </td>
                <td className="px-4 py-3 text-right">
                  <button
                    onClick={() => setEditingUser(user)}
                    className="p-1 text-gray-500 hover:text-gray-700 mr-2"
                  >
                    <Edit2 className="w-4 h-4" />
                  </button>
                  <button
                    onClick={() => {
                      if (confirm(`Delete user ${user.username}?`)) {
                        deleteMutation.mutate(user.id)
                      }
                    }}
                    className="p-1 text-red-500 hover:text-red-700"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function UserForm({
  user,
  onSubmit,
  onCancel,
  isLoading,
}: {
  user?: AdminUser
  onSubmit: (data: { username?: string; email?: string; password?: string; role?: string }) => void
  onCancel: () => void
  isLoading: boolean
}) {
  const [username, setUsername] = useState(user?.username || '')
  const [email, setEmail] = useState(user?.email || '')
  const [password, setPassword] = useState('')
  const [role, setRole] = useState(user?.role || 'user')

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    const data: { username?: string; email?: string; password?: string; role?: string } = {}
    if (user) {
      // Update - only include changed fields
      if (username !== user.username) data.username = username
      if (email !== user.email) data.email = email
      if (password) data.password = password
      if (role !== user.role) data.role = role
    } else {
      // Create - include all fields
      data.username = username
      data.email = email
      data.password = password
      data.role = role
    }
    onSubmit(data)
  }

  return (
    <form onSubmit={handleSubmit} className="card p-6 space-y-4">
      <h3 className="text-lg font-medium">{user ? 'Edit User' : 'Create User'}</h3>

      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium mb-1">Username</label>
          <input
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            className="input"
            required={!user}
            minLength={3}
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Email</label>
          <input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className="input"
            required={!user}
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">
            Password {user && '(leave blank to keep current)'}
          </label>
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="input"
            required={!user}
            minLength={8}
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Role</label>
          <select
            value={role}
            onChange={(e) => setRole(e.target.value)}
            className="input"
          >
            <option value="user">User</option>
            <option value="admin">Admin</option>
          </select>
        </div>
      </div>

      <div className="flex gap-2">
        <button type="submit" disabled={isLoading} className="btn btn-primary">
          {isLoading ? 'Saving...' : user ? 'Update User' : 'Create User'}
        </button>
        <button type="button" onClick={onCancel} className="btn btn-secondary">
          Cancel
        </button>
      </div>
    </form>
  )
}

function ScannerTab() {
  const { data: pathData } = useQuery({
    queryKey: ['admin', 'model-path'],
    queryFn: () => adminApi.getModelPath(),
  })

  const { data: scanStatus, refetch: refetchStatus } = useQuery({
    queryKey: ['admin', 'scan-status'],
    queryFn: () => adminApi.getScanStatus(),
    refetchInterval: (query) => (query?.state?.data?.data?.running ? 2000 : false),
  })

  const startScanMutation = useMutation({
    mutationFn: () => adminApi.startScan(),
    onSuccess: () => refetchStatus(),
  })

  const stopScanMutation = useMutation({
    mutationFn: () => adminApi.stopScan(),
    onSuccess: () => refetchStatus(),
  })

  return (
    <div className="space-y-6">
      <div className="card p-6">
        <div className="flex items-center justify-between mb-6">
          <div>
            <h2 className="text-lg font-medium">File System Scanner</h2>
            <p className="text-sm text-gray-500">
              Scan the model directory to index new 3D files
            </p>
            <p className="text-sm text-gray-400 mt-1">
              Path: <code className="bg-gray-100 dark:bg-gray-700 px-2 py-0.5 rounded">{pathData?.data?.path || 'Not set'}</code>
            </p>
          </div>
          <div className="flex gap-2">
            {scanStatus?.data?.running ? (
              <button
                onClick={() => stopScanMutation.mutate()}
                className="btn btn-secondary flex items-center gap-2"
              >
                <Square className="w-4 h-4" />
                Stop Scan
              </button>
            ) : (
              <button
                onClick={() => startScanMutation.mutate()}
                className="btn btn-primary flex items-center gap-2"
              >
                <Play className="w-4 h-4" />
                Start Scan
              </button>
            )}
            <button onClick={() => refetchStatus()} className="btn btn-secondary">
              <RefreshCw className="w-4 h-4" />
            </button>
          </div>
        </div>

        {scanStatus?.data && <ScanStatusDisplay status={scanStatus.data} />}
      </div>
    </div>
  )
}

function DatabaseTab() {
  const queryClient = useQueryClient()
  const [confirmClearModels, setConfirmClearModels] = useState('')
  const [confirmClearAll, setConfirmClearAll] = useState('')

  const clearModelsMutation = useMutation({
    mutationFn: () => adminApi.clearModels('CLEAR_MODELS'),
    onSuccess: () => {
      queryClient.invalidateQueries()
      setConfirmClearModels('')
      alert('Models cleared successfully')
    },
  })

  const clearDatabaseMutation = useMutation({
    mutationFn: () => adminApi.clearDatabase('CLEAR_ALL_DATA'),
    onSuccess: () => {
      queryClient.invalidateQueries()
      setConfirmClearAll('')
      alert('Database cleared successfully')
    },
  })

  return (
    <div className="space-y-6">
      {/* Clear Models */}
      <div className="card p-6 border-yellow-200 bg-yellow-50 dark:bg-yellow-900/20 dark:border-yellow-800">
        <div className="flex items-start gap-4">
          <AlertTriangle className="w-6 h-6 text-yellow-600 flex-shrink-0 mt-1" />
          <div className="flex-1">
            <h3 className="font-medium text-yellow-800 dark:text-yellow-200">
              Clear All Models
            </h3>
            <p className="text-sm text-yellow-700 dark:text-yellow-300 mt-1">
              This will remove all indexed models, favorites, collections, and duplicate groups.
              Categories and tags will be preserved. Files on disk are not affected.
            </p>
            <div className="mt-4 flex gap-2">
              <input
                type="text"
                value={confirmClearModels}
                onChange={(e) => setConfirmClearModels(e.target.value)}
                placeholder="Type CLEAR_MODELS to confirm"
                className="input flex-1"
              />
              <button
                onClick={() => clearModelsMutation.mutate()}
                disabled={confirmClearModels !== 'CLEAR_MODELS' || clearModelsMutation.isPending}
                className="btn bg-yellow-600 text-white hover:bg-yellow-700 disabled:opacity-50"
              >
                Clear Models
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Clear Everything */}
      <div className="card p-6 border-red-200 bg-red-50 dark:bg-red-900/20 dark:border-red-800">
        <div className="flex items-start gap-4">
          <AlertTriangle className="w-6 h-6 text-red-600 flex-shrink-0 mt-1" />
          <div className="flex-1">
            <h3 className="font-medium text-red-800 dark:text-red-200">
              Clear Entire Database
            </h3>
            <p className="text-sm text-red-700 dark:text-red-300 mt-1">
              This will remove ALL data including models, categories, tags, collections, and favorites.
              User accounts will be preserved. This action cannot be undone.
            </p>
            <div className="mt-4 flex gap-2">
              <input
                type="text"
                value={confirmClearAll}
                onChange={(e) => setConfirmClearAll(e.target.value)}
                placeholder="Type CLEAR_ALL_DATA to confirm"
                className="input flex-1"
              />
              <button
                onClick={() => clearDatabaseMutation.mutate()}
                disabled={confirmClearAll !== 'CLEAR_ALL_DATA' || clearDatabaseMutation.isPending}
                className="btn bg-red-600 text-white hover:bg-red-700 disabled:opacity-50"
              >
                Clear Database
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

function StatCard({
  icon,
  label,
  value,
}: {
  icon: React.ReactNode
  label: string
  value: number
}) {
  return (
    <div className="card p-4">
      <div className="flex items-center gap-3">
        <div className="p-2 bg-primary-100 dark:bg-primary-900 text-primary-600 rounded-lg">
          {icon}
        </div>
        <div>
          <p className="text-2xl font-bold text-gray-900 dark:text-white">
            {value.toLocaleString()}
          </p>
          <p className="text-sm text-gray-500">{label}</p>
        </div>
      </div>
    </div>
  )
}

function ScanStatusDisplay({ status }: { status: ScanStatus }) {
  const progress = status.files_scanned > 0
    ? Math.round(((status.files_indexed + status.files_skipped) / status.files_scanned) * 100)
    : 0

  return (
    <div className="space-y-4">
      {/* Status indicator */}
      <div className="flex items-center gap-2">
        <div
          className={`w-3 h-3 rounded-full ${
            status.running ? 'bg-green-500 animate-pulse' : 'bg-gray-400'
          }`}
        />
        <span className="text-sm font-medium">
          {status.running ? 'Scanning...' : 'Idle'}
        </span>
      </div>

      {/* Progress bar */}
      {status.running && (
        <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
          <div
            className="bg-primary-600 h-2 rounded-full transition-all"
            style={{ width: `${progress}%` }}
          />
        </div>
      )}

      {/* Stats grid */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
        <div className="p-3 bg-gray-50 dark:bg-gray-700 rounded">
          <p className="text-gray-500">Files Scanned</p>
          <p className="text-lg font-medium">{status.files_scanned.toLocaleString()}</p>
        </div>
        <div className="p-3 bg-gray-50 dark:bg-gray-700 rounded">
          <p className="text-gray-500">Files Indexed</p>
          <p className="text-lg font-medium text-green-600">
            {status.files_indexed.toLocaleString()}
          </p>
        </div>
        <div className="p-3 bg-gray-50 dark:bg-gray-700 rounded">
          <p className="text-gray-500">Files Skipped</p>
          <p className="text-lg font-medium">{status.files_skipped.toLocaleString()}</p>
        </div>
        <div className="p-3 bg-gray-50 dark:bg-gray-700 rounded">
          <p className="text-gray-500">Errors</p>
          <p className={`text-lg font-medium ${status.errors > 0 ? 'text-red-600' : ''}`}>
            {status.errors}
          </p>
        </div>
      </div>

      {/* Current path */}
      {status.running && status.current_path && (
        <div className="text-sm">
          <p className="text-gray-500">Current path:</p>
          <code className="text-xs bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded block truncate">
            {status.current_path}
          </code>
        </div>
      )}

      {/* Last error */}
      {status.last_error && (
        <div className="p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded text-sm">
          <p className="text-red-700 dark:text-red-300 font-medium">Last Error:</p>
          <p className="text-red-600 dark:text-red-400">{status.last_error}</p>
        </div>
      )}

      {/* Timestamps */}
      <div className="flex gap-6 text-xs text-gray-500">
        {status.started_at && (
          <span>Started: {new Date(status.started_at).toLocaleString()}</span>
        )}
        {status.completed_at && (
          <span>Completed: {new Date(status.completed_at).toLocaleString()}</span>
        )}
      </div>
    </div>
  )
}
