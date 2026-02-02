import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { categoriesApi, Category } from '../api/client'
import { useAuthStore } from '../store/auth'
import CategoryTree from '../components/CategoryTree'
import { FolderPlus, Folder, Edit2, Trash2, X, Check } from 'lucide-react'

export default function Categories() {
  const { isAuthenticated, user } = useAuthStore()
  const queryClient = useQueryClient()
  const [showCreateForm, setShowCreateForm] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [selectedCategory, setSelectedCategory] = useState<Category | null>(null)

  const isAdmin = isAuthenticated && user?.role === 'admin'

  const { data: categories, isLoading } = useQuery({
    queryKey: ['categories'],
    queryFn: () => categoriesApi.list(),
  })

  const createMutation = useMutation({
    mutationFn: (data: { name: string; parent_id?: string; description?: string }) =>
      categoriesApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['categories'] })
      setShowCreateForm(false)
    },
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Category> }) =>
      categoriesApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['categories'] })
      setEditingId(null)
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => categoriesApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['categories'] })
      setSelectedCategory(null)
    },
  })

  if (isLoading) {
    return <div className="text-center py-12 text-gray-500">Loading categories...</div>
  }

  return (
    <div className="max-w-4xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Categories</h1>
        {isAdmin && (
          <button
            onClick={() => setShowCreateForm(true)}
            className="btn btn-primary flex items-center gap-2"
          >
            <FolderPlus className="w-5 h-5" />
            New Category
          </button>
        )}
      </div>

      {/* Create Form */}
      {showCreateForm && (
        <CategoryForm
          onSubmit={(data) => createMutation.mutate(data)}
          onCancel={() => setShowCreateForm(false)}
          categories={categories?.data || []}
        />
      )}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Category Tree */}
        <div className="card p-4">
          <h2 className="text-lg font-medium text-gray-900 dark:text-white mb-4">
            Category Tree
          </h2>
          {categories?.data && categories.data.length > 0 ? (
            <CategoryTree
              categories={categories.data}
              selectedId={selectedCategory?.id}
              onSelect={setSelectedCategory}
            />
          ) : (
            <p className="text-gray-500 text-center py-8">No categories yet</p>
          )}
        </div>

        {/* Selected Category Details */}
        <div className="card p-4">
          {selectedCategory ? (
            editingId === selectedCategory.id ? (
              <CategoryEditForm
                category={selectedCategory}
                onSubmit={(data) =>
                  updateMutation.mutate({ id: selectedCategory.id, data })
                }
                onCancel={() => setEditingId(null)}
              />
            ) : (
              <div>
                <div className="flex items-start justify-between mb-4">
                  <div>
                    <h2 className="text-xl font-bold text-gray-900 dark:text-white">
                      {selectedCategory.name}
                    </h2>
                    {selectedCategory.description && (
                      <p className="text-gray-600 dark:text-gray-300 mt-1">
                        {selectedCategory.description}
                      </p>
                    )}
                  </div>
                  {isAdmin && (
                    <div className="flex gap-2">
                      <button
                        onClick={() => setEditingId(selectedCategory.id)}
                        className="p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded"
                      >
                        <Edit2 className="w-4 h-4" />
                      </button>
                      <button
                        onClick={() => {
                          if (confirm('Delete this category?')) {
                            deleteMutation.mutate(selectedCategory.id)
                          }
                        }}
                        className="p-2 text-red-500 hover:text-red-700 hover:bg-red-50 rounded"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  )}
                </div>

                <dl className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <dt className="text-gray-500">Models</dt>
                    <dd className="font-medium">{selectedCategory.model_count || 0}</dd>
                  </div>
                  <div className="flex justify-between">
                    <dt className="text-gray-500">Slug</dt>
                    <dd className="font-mono text-xs">{selectedCategory.slug}</dd>
                  </div>
                  {selectedCategory.children && selectedCategory.children.length > 0 && (
                    <div className="flex justify-between">
                      <dt className="text-gray-500">Subcategories</dt>
                      <dd className="font-medium">{selectedCategory.children.length}</dd>
                    </div>
                  )}
                </dl>

                <div className="mt-6">
                  <Link
                    to={`/browse?category=${selectedCategory.id}`}
                    className="btn btn-primary w-full"
                  >
                    Browse Models
                  </Link>
                </div>
              </div>
            )
          ) : (
            <div className="flex flex-col items-center justify-center py-12 text-gray-500">
              <Folder className="w-12 h-12 mb-4" />
              <p>Select a category to view details</p>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

function CategoryForm({
  onSubmit,
  onCancel,
  categories,
}: {
  onSubmit: (data: { name: string; parent_id?: string; description?: string }) => void
  onCancel: () => void
  categories: Category[]
}) {
  const [name, setName] = useState('')
  const [parentId, setParentId] = useState('')
  const [description, setDescription] = useState('')

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    onSubmit({
      name,
      parent_id: parentId || undefined,
      description: description || undefined,
    })
  }

  return (
    <form onSubmit={handleSubmit} className="card p-4 mb-6">
      <h3 className="text-lg font-medium mb-4">Create Category</h3>
      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium mb-1">Name</label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="input"
            required
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Parent Category</label>
          <select
            value={parentId}
            onChange={(e) => setParentId(e.target.value)}
            className="input"
          >
            <option value="">None (Top Level)</option>
            {flattenCategories(categories).map((cat) => (
              <option key={cat.id} value={cat.id}>
                {cat.name}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Description</label>
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            className="input"
            rows={3}
          />
        </div>
        <div className="flex gap-2">
          <button type="submit" className="btn btn-primary">
            <Check className="w-4 h-4 mr-1" />
            Create
          </button>
          <button type="button" onClick={onCancel} className="btn btn-secondary">
            <X className="w-4 h-4 mr-1" />
            Cancel
          </button>
        </div>
      </div>
    </form>
  )
}

function CategoryEditForm({
  category,
  onSubmit,
  onCancel,
}: {
  category: Category
  onSubmit: (data: Partial<Category>) => void
  onCancel: () => void
}) {
  const [name, setName] = useState(category.name)
  const [description, setDescription] = useState(category.description || '')

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    onSubmit({ name, description: description || undefined })
  }

  return (
    <form onSubmit={handleSubmit}>
      <h3 className="text-lg font-medium mb-4">Edit Category</h3>
      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium mb-1">Name</label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="input"
            required
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Description</label>
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            className="input"
            rows={3}
          />
        </div>
        <div className="flex gap-2">
          <button type="submit" className="btn btn-primary">
            <Check className="w-4 h-4 mr-1" />
            Save
          </button>
          <button type="button" onClick={onCancel} className="btn btn-secondary">
            <X className="w-4 h-4 mr-1" />
            Cancel
          </button>
        </div>
      </div>
    </form>
  )
}

function flattenCategories(categories: Category[], result: Category[] = []): Category[] {
  for (const cat of categories) {
    result.push(cat)
    if (cat.children) {
      flattenCategories(cat.children, result)
    }
  }
  return result
}
