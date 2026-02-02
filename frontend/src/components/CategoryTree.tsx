import { useState } from 'react'
import { Category } from '../api/client'
import { ChevronRight, ChevronDown, Folder, FolderOpen } from 'lucide-react'

interface CategoryTreeProps {
  categories: Category[]
  selectedId?: string
  onSelect: (category: Category) => void
}

export default function CategoryTree({ categories, selectedId, onSelect }: CategoryTreeProps) {
  return (
    <div className="space-y-1">
      {categories.map((category) => (
        <CategoryNode
          key={category.id}
          category={category}
          selectedId={selectedId}
          onSelect={onSelect}
          level={0}
        />
      ))}
    </div>
  )
}

interface CategoryNodeProps {
  category: Category
  selectedId?: string
  onSelect: (category: Category) => void
  level: number
}

function CategoryNode({ category, selectedId, onSelect, level }: CategoryNodeProps) {
  const [expanded, setExpanded] = useState(false)
  const hasChildren = category.children && category.children.length > 0
  const isSelected = category.id === selectedId

  return (
    <div>
      <div
        className={`flex items-center gap-2 px-2 py-1.5 rounded-lg cursor-pointer transition-colors ${
          isSelected
            ? 'bg-primary-100 dark:bg-primary-900 text-primary-700 dark:text-primary-300'
            : 'hover:bg-gray-100 dark:hover:bg-gray-700'
        }`}
        style={{ paddingLeft: `${level * 16 + 8}px` }}
        onClick={() => onSelect(category)}
      >
        {hasChildren ? (
          <button
            onClick={(e) => {
              e.stopPropagation()
              setExpanded(!expanded)
            }}
            className="p-0.5 hover:bg-gray-200 dark:hover:bg-gray-600 rounded"
          >
            {expanded ? (
              <ChevronDown className="w-4 h-4" />
            ) : (
              <ChevronRight className="w-4 h-4" />
            )}
          </button>
        ) : (
          <div className="w-5" />
        )}

        {expanded ? (
          <FolderOpen className="w-4 h-4 text-primary-500" />
        ) : (
          <Folder className="w-4 h-4 text-gray-400" />
        )}

        <span className="flex-1 truncate">{category.name}</span>

        {category.model_count !== undefined && category.model_count > 0 && (
          <span className="text-xs text-gray-400">{category.model_count}</span>
        )}
      </div>

      {hasChildren && expanded && (
        <div>
          {category.children!.map((child) => (
            <CategoryNode
              key={child.id}
              category={child}
              selectedId={selectedId}
              onSelect={onSelect}
              level={level + 1}
            />
          ))}
        </div>
      )}
    </div>
  )
}
