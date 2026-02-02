import { Link } from 'react-router-dom'
import { Model, modelsApi } from '../api/client'
import { Tag } from 'lucide-react'
import ModelThumbnail from './ModelThumbnail'

interface ModelCardProps {
  model: Model
}

export default function ModelCard({ model }: ModelCardProps) {
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
    <Link
      to={`/models/${model.id}`}
      className="card overflow-hidden group hover:shadow-md transition-shadow"
    >
      {/* 3D Thumbnail */}
      <div className="aspect-square relative overflow-hidden">
        <ModelThumbnail
          modelUrl={modelsApi.getFileUrl(model.id)}
          fileType={model.file_type}
          className="w-full h-full"
        />
        {/* File type badge */}
        <span className="absolute top-2 right-2 px-2 py-1 bg-black/70 text-white text-xs rounded uppercase">
          {model.file_type}
        </span>
      </div>

      {/* Info */}
      <div className="p-4">
        <h3 className="font-medium text-gray-900 dark:text-white truncate group-hover:text-primary-600">
          {model.name}
        </h3>

        <div className="mt-2 flex items-center justify-between text-sm text-gray-500 dark:text-gray-400">
          <span>{formatFileSize(model.file_size)}</span>
          {model.source && (
            <span className="capitalize">{model.source}</span>
          )}
        </div>

        {/* Tags */}
        {model.tags && model.tags.length > 0 && (
          <div className="mt-3 flex flex-wrap gap-1">
            {model.tags.slice(0, 3).map((tag) => (
              <span
                key={tag.id}
                className="inline-flex items-center gap-1 px-2 py-0.5 text-xs rounded-full"
                style={{ backgroundColor: tag.color + '20', color: tag.color }}
              >
                <Tag className="w-3 h-3" />
                {tag.name}
              </span>
            ))}
            {model.tags.length > 3 && (
              <span className="text-xs text-gray-400">
                +{model.tags.length - 3} more
              </span>
            )}
          </div>
        )}
      </div>
    </Link>
  )
}
