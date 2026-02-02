import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { modelsApi, categoriesApi } from '../api/client'
import ModelGrid from '../components/ModelGrid'
import { ArrowRight, Box, FolderTree, FileBox } from 'lucide-react'

export default function Home() {
  const { data: recentModels, isLoading: modelsLoading } = useQuery({
    queryKey: ['models', 'recent'],
    queryFn: () => modelsApi.list({ limit: 8 }),
  })

  const { data: categories } = useQuery({
    queryKey: ['categories'],
    queryFn: () => categoriesApi.list(),
  })

  return (
    <div className="space-y-12">
      {/* Hero */}
      <div className="text-center py-12">
        <div className="flex justify-center mb-6">
          <Box className="w-20 h-20 text-primary-600" />
        </div>
        <h1 className="text-4xl font-bold text-gray-900 dark:text-white mb-4">
          Optics 3D Model Library
        </h1>
        <p className="text-xl text-gray-600 dark:text-gray-300 max-w-2xl mx-auto">
          Browse, organize, and manage your collection of 3D models including STL, SCAD, and more.
        </p>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <StatCard
          icon={<FileBox className="w-8 h-8" />}
          label="Total Models"
          value={recentModels?.data.total || 0}
        />
        <StatCard
          icon={<FolderTree className="w-8 h-8" />}
          label="Categories"
          value={categories?.data.length || 0}
        />
        <StatCard
          icon={<Box className="w-8 h-8" />}
          label="File Types"
          value="STL, SCAD, OBJ, 3MF"
          isText
        />
      </div>

      {/* Recent Models */}
      <section>
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-2xl font-bold text-gray-900 dark:text-white">
            Recent Models
          </h2>
          <Link
            to="/browse"
            className="flex items-center gap-1 text-primary-600 hover:text-primary-700"
          >
            View all
            <ArrowRight className="w-4 h-4" />
          </Link>
        </div>
        <ModelGrid
          models={(recentModels?.data.models || []).filter(m => !['scad', 'dat'].includes(m.file_type.toLowerCase()))}
          loading={modelsLoading}
        />
      </section>

      {/* Categories */}
      {categories?.data && categories.data.length > 0 && (
        <section>
          <div className="flex items-center justify-between mb-6">
            <h2 className="text-2xl font-bold text-gray-900 dark:text-white">
              Categories
            </h2>
            <Link
              to="/categories"
              className="flex items-center gap-1 text-primary-600 hover:text-primary-700"
            >
              View all
              <ArrowRight className="w-4 h-4" />
            </Link>
          </div>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            {categories.data.slice(0, 8).map((category) => (
              <Link
                key={category.id}
                to={`/browse?category=${category.id}`}
                className="card p-4 hover:shadow-md transition-shadow"
              >
                <FolderTree className="w-8 h-8 text-primary-500 mb-2" />
                <h3 className="font-medium text-gray-900 dark:text-white">
                  {category.name}
                </h3>
                {category.model_count !== undefined && (
                  <p className="text-sm text-gray-500">
                    {category.model_count} models
                  </p>
                )}
              </Link>
            ))}
          </div>
        </section>
      )}
    </div>
  )
}

function StatCard({
  icon,
  label,
  value,
  isText = false,
}: {
  icon: React.ReactNode
  label: string
  value: number | string
  isText?: boolean
}) {
  return (
    <div className="card p-6">
      <div className="flex items-center gap-4">
        <div className="text-primary-600">{icon}</div>
        <div>
          <p className="text-sm text-gray-500 dark:text-gray-400">{label}</p>
          <p className={`${isText ? 'text-lg' : 'text-2xl'} font-bold text-gray-900 dark:text-white`}>
            {typeof value === 'number' ? value.toLocaleString() : value}
          </p>
        </div>
      </div>
    </div>
  )
}
