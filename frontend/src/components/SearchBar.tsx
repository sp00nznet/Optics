import { useState, useEffect, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { searchApi, Model } from '../api/client'
import { Search, FileBox, X } from 'lucide-react'
import { useDebounce } from '../hooks/useDebounce'

interface SearchBarProps {
  className?: string
}

export default function SearchBar({ className = '' }: SearchBarProps) {
  const navigate = useNavigate()
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<Model[]>([])
  const [loading, setLoading] = useState(false)
  const [showResults, setShowResults] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)

  const debouncedQuery = useDebounce(query, 300)

  useEffect(() => {
    if (!debouncedQuery.trim()) {
      setResults([])
      return
    }

    setLoading(true)
    searchApi
      .search(debouncedQuery)
      .then((response) => {
        setResults(response.data.results || [])
      })
      .catch(() => {
        setResults([])
      })
      .finally(() => {
        setLoading(false)
      })
  }, [debouncedQuery])

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setShowResults(false)
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (query.trim()) {
      navigate(`/browse?q=${encodeURIComponent(query)}`)
      setShowResults(false)
    }
  }

  const handleSelect = (model: Model) => {
    navigate(`/models/${model.id}`)
    setQuery('')
    setShowResults(false)
  }

  return (
    <div ref={containerRef} className={`relative ${className}`}>
      <form onSubmit={handleSubmit}>
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" />
          <input
            type="text"
            value={query}
            onChange={(e) => {
              setQuery(e.target.value)
              setShowResults(true)
            }}
            onFocus={() => setShowResults(true)}
            placeholder="Search models..."
            className="w-full pl-10 pr-10 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-gray-50 dark:bg-gray-700 focus:ring-2 focus:ring-primary-500 focus:border-transparent outline-none"
          />
          {query && (
            <button
              type="button"
              onClick={() => {
                setQuery('')
                setResults([])
              }}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
            >
              <X className="w-5 h-5" />
            </button>
          )}
        </div>
      </form>

      {/* Results dropdown */}
      {showResults && query.trim() && (
        <div className="absolute top-full left-0 right-0 mt-1 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 max-h-96 overflow-auto z-50">
          {loading ? (
            <div className="p-4 text-center text-gray-500">Searching...</div>
          ) : results.length > 0 ? (
            <ul>
              {results.slice(0, 10).map((model) => (
                <li key={model.id}>
                  <button
                    onClick={() => handleSelect(model)}
                    className="w-full flex items-center gap-3 px-4 py-3 hover:bg-gray-50 dark:hover:bg-gray-700 text-left"
                  >
                    <FileBox className="w-8 h-8 text-gray-400 flex-shrink-0" />
                    <div className="flex-1 min-w-0">
                      <p className="font-medium text-gray-900 dark:text-white truncate">
                        {model.name}
                      </p>
                      <p className="text-sm text-gray-500 truncate">
                        {model.file_type.toUpperCase()} • {model.source || 'Local'}
                      </p>
                    </div>
                  </button>
                </li>
              ))}
              {results.length > 10 && (
                <li className="px-4 py-2 text-sm text-gray-500 text-center border-t border-gray-200 dark:border-gray-700">
                  {results.length - 10} more results...
                </li>
              )}
            </ul>
          ) : (
            <div className="p-4 text-center text-gray-500">No results found</div>
          )}
        </div>
      )}
    </div>
  )
}
