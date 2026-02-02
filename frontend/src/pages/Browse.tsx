import { useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { modelsApi, categoriesApi, Category } from '../api/client'
import ModelGrid from '../components/ModelGrid'
import CategoryTree from '../components/CategoryTree'
import { Filter, X } from 'lucide-react'

export default function Browse() {
  const [searchParams, setSearchParams] = useSearchParams()
  const [showFilters, setShowFilters] = useState(true)

  const page = parseInt(searchParams.get('page') || '1')
  const categoryId = searchParams.get('category') || undefined
  const fileType = searchParams.get('type') || undefined
  const searchQuery = searchParams.get('q') || undefined

  const { data: modelsData, isLoading } = useQuery({
    queryKey: ['models', { page, category: categoryId, type: fileType, q: searchQuery }],
    queryFn: () =>
      modelsApi.list({
        page,
        limit: 20,
        category: categoryId,
        type: fileType,
        q: searchQuery,
      }),
  })

  const { data: categories } = useQuery({
    queryKey: ['categories'],
    queryFn: () => categoriesApi.list(),
  })

  const handleCategorySelect = (category: Category) => {
    const params = new URLSearchParams(searchParams)
    params.set('category', category.id)
    params.delete('page')
    setSearchParams(params)
  }

  const handleTypeFilter = (type: string | null) => {
    const params = new URLSearchParams(searchParams)
    if (type) {
      params.set('type', type)
    } else {
      params.delete('type')
    }
    params.delete('page')
    setSearchParams(params)
  }

  const clearFilters = () => {
    setSearchParams({})
  }

  const hasFilters = categoryId || fileType || searchQuery

  return (
    <div className="flex gap-6">
      {/* Sidebar Filters */}
      {showFilters && (
        <aside className="w-64 flex-shrink-0 space-y-6">
          {/* Categories */}
          <div className="card p-4">
            <h3 className="font-medium text-gray-900 dark:text-white mb-3">
              Categories
            </h3>
            {categories?.data && (
              <CategoryTree
                categories={categories.data}
                selectedId={categoryId}
                onSelect={handleCategorySelect}
              />
            )}
          </div>

          {/* File Type */}
          <div className="card p-4">
            <h3 className="font-medium text-gray-900 dark:text-white mb-3">
              File Type
            </h3>
            <div className="space-y-2">
              {['stl', 'scad', 'obj', '3mf', 'step'].map((type) => (
                <label key={type} className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="radio"
                    name="fileType"
                    checked={fileType === type}
                    onChange={() => handleTypeFilter(type)}
                    className="text-primary-600"
                  />
                  <span className="text-sm uppercase">{type}</span>
                </label>
              ))}
              {fileType && (
                <button
                  onClick={() => handleTypeFilter(null)}
                  className="text-sm text-primary-600 hover:underline"
                >
                  Clear
                </button>
              )}
            </div>
          </div>
        </aside>
      )}

      {/* Main Content */}
      <div className="flex-1">
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <div className="flex items-center gap-4">
            <button
              onClick={() => setShowFilters(!showFilters)}
              className="btn btn-secondary flex items-center gap-2"
            >
              <Filter className="w-4 h-4" />
              Filters
            </button>

            {hasFilters && (
              <button
                onClick={clearFilters}
                className="flex items-center gap-1 text-sm text-gray-500 hover:text-gray-700"
              >
                <X className="w-4 h-4" />
                Clear all filters
              </button>
            )}
          </div>

          <p className="text-sm text-gray-500">
            {modelsData?.data.total || 0} models found
          </p>
        </div>

        {/* Active Filters */}
        {hasFilters && (
          <div className="flex flex-wrap gap-2 mb-4">
            {searchQuery && (
              <FilterBadge
                label={`Search: ${searchQuery}`}
                onRemove={() => {
                  const params = new URLSearchParams(searchParams)
                  params.delete('q')
                  setSearchParams(params)
                }}
              />
            )}
            {categoryId && categories?.data && (
              <FilterBadge
                label={`Category: ${findCategoryName(categories.data, categoryId)}`}
                onRemove={() => {
                  const params = new URLSearchParams(searchParams)
                  params.delete('category')
                  setSearchParams(params)
                }}
              />
            )}
            {fileType && (
              <FilterBadge
                label={`Type: ${fileType.toUpperCase()}`}
                onRemove={() => handleTypeFilter(null)}
              />
            )}
          </div>
        )}

        {/* Grid - filter out SCAD and DAT files */}
        <ModelGrid
          models={(modelsData?.data.models || []).filter(m => !['scad', 'dat'].includes(m.file_type.toLowerCase()))}
          loading={isLoading}
        />

        {/* Pagination */}
        {modelsData?.data && modelsData.data.total_pages > 1 && (
          <div className="mt-8 flex justify-center gap-2">
            <button
              onClick={() => {
                const params = new URLSearchParams(searchParams)
                params.set('page', String(page - 1))
                setSearchParams(params)
              }}
              disabled={page <= 1}
              className="btn btn-secondary disabled:opacity-50"
            >
              Previous
            </button>
            <span className="flex items-center px-4 text-sm text-gray-500">
              Page {page} of {modelsData.data.total_pages}
            </span>
            <button
              onClick={() => {
                const params = new URLSearchParams(searchParams)
                params.set('page', String(page + 1))
                setSearchParams(params)
              }}
              disabled={page >= modelsData.data.total_pages}
              className="btn btn-secondary disabled:opacity-50"
            >
              Next
            </button>
          </div>
        )}
      </div>
    </div>
  )
}

function FilterBadge({ label, onRemove }: { label: string; onRemove: () => void }) {
  return (
    <span className="inline-flex items-center gap-1 px-2 py-1 text-sm bg-primary-100 dark:bg-primary-900 text-primary-700 dark:text-primary-300 rounded-full">
      {label}
      <button onClick={onRemove} className="hover:text-primary-900">
        <X className="w-3 h-3" />
      </button>
    </span>
  )
}

function findCategoryName(categories: Category[], id: string): string {
  for (const cat of categories) {
    if (cat.id === id) return cat.name
    if (cat.children) {
      const found = findCategoryName(cat.children, id)
      if (found) return found
    }
  }
  return 'Unknown'
}
