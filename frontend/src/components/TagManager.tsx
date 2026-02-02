import { useState } from 'react'
import { Tag } from '../api/client'
import { Plus, X } from 'lucide-react'

interface TagManagerProps {
  tags: Tag[]
  selectedTags: Tag[]
  onAddTag: (tag: Tag) => void
  onRemoveTag: (tag: Tag) => void
  onCreateTag?: (name: string, color: string) => void
  editable?: boolean
}

export default function TagManager({
  tags,
  selectedTags,
  onAddTag,
  onRemoveTag,
  onCreateTag,
  editable = true,
}: TagManagerProps) {
  const [showAdd, setShowAdd] = useState(false)
  const [newTagName, setNewTagName] = useState('')
  const [newTagColor, setNewTagColor] = useState('#6366f1')

  const availableTags = tags.filter(
    (tag) => !selectedTags.find((t) => t.id === tag.id)
  )

  const handleCreateTag = () => {
    if (newTagName.trim() && onCreateTag) {
      onCreateTag(newTagName.trim(), newTagColor)
      setNewTagName('')
      setShowAdd(false)
    }
  }

  return (
    <div className="space-y-3">
      {/* Selected Tags */}
      <div className="flex flex-wrap gap-2">
        {selectedTags.map((tag) => (
          <span
            key={tag.id}
            className="inline-flex items-center gap-1 px-2 py-1 text-sm rounded-full"
            style={{ backgroundColor: tag.color + '20', color: tag.color }}
          >
            {tag.name}
            {editable && (
              <button
                onClick={() => onRemoveTag(tag)}
                className="hover:bg-black/10 rounded-full p-0.5"
              >
                <X className="w-3 h-3" />
              </button>
            )}
          </span>
        ))}

        {editable && (
          <button
            onClick={() => setShowAdd(!showAdd)}
            className="inline-flex items-center gap-1 px-2 py-1 text-sm text-gray-500 border border-dashed border-gray-300 dark:border-gray-600 rounded-full hover:border-gray-400"
          >
            <Plus className="w-3 h-3" />
            Add tag
          </button>
        )}
      </div>

      {/* Add Tag Dropdown */}
      {showAdd && editable && (
        <div className="p-3 bg-gray-50 dark:bg-gray-800 rounded-lg space-y-3">
          {/* Available tags */}
          {availableTags.length > 0 && (
            <div>
              <p className="text-xs text-gray-500 mb-2">Available tags:</p>
              <div className="flex flex-wrap gap-1">
                {availableTags.map((tag) => (
                  <button
                    key={tag.id}
                    onClick={() => {
                      onAddTag(tag)
                      setShowAdd(false)
                    }}
                    className="px-2 py-1 text-sm rounded-full hover:opacity-80 transition-opacity"
                    style={{ backgroundColor: tag.color + '20', color: tag.color }}
                  >
                    {tag.name}
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* Create new tag */}
          {onCreateTag && (
            <div>
              <p className="text-xs text-gray-500 mb-2">Or create new:</p>
              <div className="flex gap-2">
                <input
                  type="text"
                  value={newTagName}
                  onChange={(e) => setNewTagName(e.target.value)}
                  placeholder="Tag name"
                  className="input flex-1 text-sm"
                />
                <input
                  type="color"
                  value={newTagColor}
                  onChange={(e) => setNewTagColor(e.target.value)}
                  className="w-10 h-10 rounded cursor-pointer"
                />
                <button
                  onClick={handleCreateTag}
                  disabled={!newTagName.trim()}
                  className="btn btn-primary text-sm"
                >
                  Create
                </button>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
