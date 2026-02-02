import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { adminApi, DuplicateGroup } from '../api/client'
import { Copy, Check, Trash2, FileBox } from 'lucide-react'
import { useState } from 'react'

export default function Duplicates() {
  const queryClient = useQueryClient()

  const { data: duplicates, isLoading } = useQuery({
    queryKey: ['duplicates'],
    queryFn: () => adminApi.listDuplicates(),
  })

  const detectMutation = useMutation({
    mutationFn: () => adminApi.detectDuplicates(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['duplicates'] })
    },
  })

  const resolveMutation = useMutation({
    mutationFn: ({
      groupId,
      primaryModelId,
      deleteOthers,
    }: {
      groupId: string
      primaryModelId: string
      deleteOthers: boolean
    }) => adminApi.resolveDuplicate(groupId, primaryModelId, deleteOthers),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['duplicates'] })
    },
  })

  if (isLoading) {
    return <div className="text-center py-12 text-gray-500">Loading...</div>
  }

  return (
    <div className="max-w-6xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
            Duplicate Detection
          </h1>
          <p className="text-gray-500 mt-1">
            Find and resolve duplicate 3D models in your library
          </p>
        </div>
        <button
          onClick={() => detectMutation.mutate()}
          disabled={detectMutation.isPending}
          className="btn btn-primary flex items-center gap-2"
        >
          <Copy className="w-5 h-5" />
          {detectMutation.isPending ? 'Detecting...' : 'Detect Duplicates'}
        </button>
      </div>

      {duplicates?.data && duplicates.data.length > 0 ? (
        <div className="space-y-6">
          {duplicates.data.map((group) => (
            <DuplicateGroupCard
              key={group.id}
              group={group}
              onResolve={(primaryId, deleteOthers) =>
                resolveMutation.mutate({
                  groupId: group.id,
                  primaryModelId: primaryId,
                  deleteOthers,
                })
              }
            />
          ))}
        </div>
      ) : (
        <div className="card p-12 text-center">
          <Copy className="w-16 h-16 mx-auto text-gray-300 mb-4" />
          <h2 className="text-xl font-medium text-gray-900 dark:text-white mb-2">
            No duplicates found
          </h2>
          <p className="text-gray-500">
            Click "Detect Duplicates" to scan your library for duplicate files
          </p>
        </div>
      )}
    </div>
  )
}

function DuplicateGroupCard({
  group,
  onResolve,
}: {
  group: DuplicateGroup
  onResolve: (primaryId: string, deleteOthers: boolean) => void
}) {
  const [selectedPrimary, setSelectedPrimary] = useState(group.primary_model_id)

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

  return (
    <div className="card">
      <div className="p-4 border-b border-gray-200 dark:border-gray-700">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Copy className="w-5 h-5 text-yellow-500" />
            <span className="font-medium">
              {group.models.length} duplicate files
            </span>
          </div>
          <div className="flex gap-2">
            <button
              onClick={() => onResolve(selectedPrimary, false)}
              className="btn btn-secondary text-sm"
            >
              <Check className="w-4 h-4 mr-1" />
              Keep All (Mark Resolved)
            </button>
            <button
              onClick={() => {
                if (confirm('This will remove duplicate entries from the database (files are not deleted). Continue?')) {
                  onResolve(selectedPrimary, true)
                }
              }}
              className="btn btn-primary text-sm"
            >
              <Trash2 className="w-4 h-4 mr-1" />
              Keep Selected Only
            </button>
          </div>
        </div>
      </div>

      <div className="divide-y divide-gray-200 dark:divide-gray-700">
        {group.models.map((model) => (
          <div
            key={model.id}
            className={`p-4 flex items-center gap-4 ${
              model.id === selectedPrimary ? 'bg-primary-50 dark:bg-primary-900/20' : ''
            }`}
          >
            <input
              type="radio"
              name={`primary-${group.id}`}
              checked={model.id === selectedPrimary}
              onChange={() => setSelectedPrimary(model.id)}
              className="w-4 h-4 text-primary-600"
            />

            <div className="w-16 h-16 bg-gray-100 dark:bg-gray-700 rounded flex items-center justify-center flex-shrink-0">
              <FileBox className="w-8 h-8 text-gray-400" />
            </div>

            <div className="flex-1 min-w-0">
              <h3 className="font-medium text-gray-900 dark:text-white truncate">
                {model.name}
              </h3>
              <p className="text-sm text-gray-500 truncate">{model.file_path}</p>
              <div className="flex gap-4 mt-1 text-xs text-gray-400">
                <span>{model.file_type.toUpperCase()}</span>
                <span>{formatFileSize(model.file_size)}</span>
                <span>{new Date(model.created_at).toLocaleDateString()}</span>
              </div>
            </div>

            {model.id === selectedPrimary && (
              <span className="px-2 py-1 bg-primary-100 text-primary-700 text-xs rounded-full">
                Primary
              </span>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
