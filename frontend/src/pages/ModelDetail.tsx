import { useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { modelsApi, tagsApi } from '../api/client'
import { useAuthStore } from '../store/auth'
import ModelViewer from '../components/ModelViewer'
import ScadViewer from '../components/ScadViewer'
import TagManager from '../components/TagManager'
import {
  Download,
  Heart,
  Folder,
  Calendar,
  HardDrive,
  User,
  FileBox,
  Edit2,
  Check,
  X,
} from 'lucide-react'

export default function ModelDetail() {
  const { id } = useParams<{ id: string }>()
  const { isAuthenticated, user } = useAuthStore()
  const queryClient = useQueryClient()
  const [isEditing, setIsEditing] = useState(false)
  const [editName, setEditName] = useState('')
  const [editDescription, setEditDescription] = useState('')

  const { data: model, isLoading } = useQuery({
    queryKey: ['model', id],
    queryFn: () => modelsApi.get(id!),
    enabled: !!id,
  })

  const { data: tags } = useQuery({
    queryKey: ['tags'],
    queryFn: () => tagsApi.list(),
  })

  const updateMutation = useMutation({
    mutationFn: (data: { name?: string; description?: string }) =>
      modelsApi.update(id!, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['model', id] })
      setIsEditing(false)
    },
  })

  const addTagMutation = useMutation({
    mutationFn: (tagId: string) => modelsApi.addTags(id!, [tagId]),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['model', id] }),
  })

  const removeTagMutation = useMutation({
    mutationFn: (tagId: string) => modelsApi.removeTag(id!, tagId),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['model', id] }),
  })

  const favoriteMutation = useMutation({
    mutationFn: () => modelsApi.toggleFavorite(id!),
  })

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-gray-500">Loading...</div>
      </div>
    )
  }

  if (!model?.data) {
    return (
      <div className="text-center py-12">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
          Model not found
        </h1>
        <Link to="/browse" className="text-primary-600 hover:underline mt-4 inline-block">
          Back to browse
        </Link>
      </div>
    )
  }

  const m = model.data
  const isSTL = m.file_type.toLowerCase() === 'stl'
  const isSCAD = m.file_type.toLowerCase() === 'scad'
  const canEdit = isAuthenticated && (user?.role === 'admin')

  const formatFileSize = (bytes?: number) => {
    if (!bytes) return 'Unknown'
    const units = ['B', 'KB', 'MB', 'GB']
    let size = bytes
    let unitIndex = 0
    while (size >= 1024 && unitIndex < units.length - 1) {
      size /= 1024
      unitIndex++
    }
    return `${size.toFixed(1)} ${units[unitIndex]}`
  }

  const startEditing = () => {
    setEditName(m.name)
    setEditDescription(m.description || '')
    setIsEditing(true)
  }

  const saveEdits = () => {
    updateMutation.mutate({
      name: editName,
      description: editDescription,
    })
  }

  return (
    <div className="max-w-6xl mx-auto">
      {/* Header */}
      <div className="mb-6">
        {isEditing ? (
          <div className="space-y-4">
            <input
              type="text"
              value={editName}
              onChange={(e) => setEditName(e.target.value)}
              className="input text-2xl font-bold"
            />
            <div className="flex gap-2">
              <button onClick={saveEdits} className="btn btn-primary">
                <Check className="w-4 h-4 mr-1" />
                Save
              </button>
              <button onClick={() => setIsEditing(false)} className="btn btn-secondary">
                <X className="w-4 h-4 mr-1" />
                Cancel
              </button>
            </div>
          </div>
        ) : (
          <div className="flex items-start justify-between">
            <div>
              <h1 className="text-3xl font-bold text-gray-900 dark:text-white">
                {m.name}
              </h1>
              {m.category && (
                <Link
                  to={`/browse?category=${m.category.id}`}
                  className="inline-flex items-center gap-1 mt-2 text-sm text-primary-600 hover:underline"
                >
                  <Folder className="w-4 h-4" />
                  {m.category.name}
                </Link>
              )}
            </div>
            <div className="flex items-center gap-2">
              {canEdit && (
                <button onClick={startEditing} className="btn btn-secondary">
                  <Edit2 className="w-4 h-4" />
                </button>
              )}
              {isAuthenticated && (
                <button
                  onClick={() => favoriteMutation.mutate()}
                  className="btn btn-secondary"
                >
                  <Heart className="w-5 h-5" />
                </button>
              )}
              <a
                href={modelsApi.getFileUrl(m.id)}
                download
                className="btn btn-primary flex items-center gap-2"
              >
                <Download className="w-5 h-5" />
                Download
              </a>
            </div>
          </div>
        )}
      </div>

      {/* Side-by-side Viewer and Code Editor */}
      <div className="grid grid-cols-1 xl:grid-cols-2 gap-4 mb-6">
        {/* 3D Viewer */}
        <div>
          <h2 className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-2">3D Preview</h2>
          {isSTL ? (
            <ModelViewer modelUrl={modelsApi.getFileUrl(m.id)} className="h-[500px]" />
          ) : (
            <div className="h-[500px] bg-gray-100 dark:bg-gray-700 rounded-lg flex items-center justify-center">
              <div className="text-center">
                <FileBox className="w-16 h-16 mx-auto text-gray-400 mb-4" />
                <p className="text-gray-500">
                  {isSCAD
                    ? 'SCAD preview requires OpenSCAD rendering'
                    : `Preview not available for ${m.file_type.toUpperCase()} files`
                  }
                </p>
              </div>
            </div>
          )}
        </div>

        {/* Code Editor */}
        <div>
          <h2 className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-2">OpenSCAD Code</h2>
          <ScadViewer
            fileUrl={modelsApi.getFileUrl(m.id)}
            fileName={m.name + '.' + m.file_type}
            fileType={m.file_type}
            className="h-[500px]"
          />
        </div>
      </div>

      {/* Details Section */}
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-4">
        {/* Info */}
        <div className="card p-4">
          <h2 className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-3">
            Details
          </h2>
          <dl className="space-y-2">
            <DetailItem
              icon={<FileBox className="w-4 h-4" />}
              label="Type"
              value={m.file_type.toUpperCase()}
            />
            <DetailItem
              icon={<HardDrive className="w-4 h-4" />}
              label="Size"
              value={formatFileSize(m.file_size)}
            />
            {m.author && (
              <DetailItem
                icon={<User className="w-4 h-4" />}
                label="Author"
                value={m.author}
              />
            )}
            {m.source && (
              <DetailItem
                icon={<Folder className="w-4 h-4" />}
                label="Source"
                value={m.source}
              />
            )}
            <DetailItem
              icon={<Calendar className="w-4 h-4" />}
              label="Indexed"
              value={new Date(m.indexed_at).toLocaleDateString()}
            />
          </dl>
        </div>

        {/* Description */}
        <div className="card p-4">
          <h2 className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-3">
            Description
          </h2>
          {isEditing ? (
            <textarea
              value={editDescription}
              onChange={(e) => setEditDescription(e.target.value)}
              className="input h-24 text-sm"
              placeholder="Add a description..."
            />
          ) : (
            <p className="text-sm text-gray-600 dark:text-gray-300">
              {m.description || 'No description available'}
            </p>
          )}
        </div>

        {/* Tags */}
        <div className="card p-4">
          <h2 className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-3">
            Tags
          </h2>
          <TagManager
            tags={tags?.data || []}
            selectedTags={m.tags || []}
            onAddTag={(tag) => addTagMutation.mutate(tag.id)}
            onRemoveTag={(tag) => removeTagMutation.mutate(tag.id)}
            editable={canEdit}
          />
        </div>

        {/* File Path */}
        <div className="card p-4">
          <h2 className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-3">
            File Location
          </h2>
          <code className="block p-2 bg-gray-100 dark:bg-gray-700 rounded text-xs break-all">
            {m.file_path}
          </code>
        </div>
      </div>
    </div>
  )
}

function DetailItem({
  icon,
  label,
  value,
}: {
  icon: React.ReactNode
  label: string
  value: string
}) {
  return (
    <div className="flex items-center gap-3">
      <span className="text-gray-400">{icon}</span>
      <dt className="text-sm text-gray-500">{label}:</dt>
      <dd className="text-sm font-medium text-gray-900 dark:text-white">{value}</dd>
    </div>
  )
}
