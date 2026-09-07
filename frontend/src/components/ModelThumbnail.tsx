import { useEffect, useState } from 'react'
import { FileBox } from 'lucide-react'
import { thumbnailService } from '../services/thumbnailService'

interface ModelThumbnailProps {
  modelUrl: string
  fileType: string
  className?: string
}

export default function ModelThumbnail({ modelUrl, fileType, className = '' }: ModelThumbnailProps) {
  const [thumbnail, setThumbnail] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)

  const isSTL = fileType?.toLowerCase() === 'stl'

  useEffect(() => {
    if (!isSTL) {
      setLoading(false)
      return
    }

    setLoading(true)
    setError(false)

    thumbnailService.getThumbnail(modelUrl).then((dataUrl) => {
      if (dataUrl) {
        setThumbnail(dataUrl)
      } else {
        setError(true)
      }
      setLoading(false)
    })
  }, [modelUrl, isSTL])

  // For non-STL files, show a placeholder
  if (!isSTL) {
    return (
      <div className={`bg-gray-700 flex items-center justify-center ${className}`}>
        <div className="text-center text-gray-400">
          <FileBox className="w-12 h-12 mx-auto mb-2" />
          <span className="text-xs uppercase">{fileType}</span>
        </div>
      </div>
    )
  }

  if (loading) {
    return (
      <div className={`bg-gray-800 flex items-center justify-center ${className}`}>
        <div className="animate-pulse bg-gray-700 w-full h-full" />
      </div>
    )
  }

  if (error || !thumbnail) {
    return (
      <div className={`bg-gray-700 flex items-center justify-center ${className}`}>
        <div className="text-center text-gray-400">
          <FileBox className="w-12 h-12 mx-auto mb-2" />
          <span className="text-xs">Preview unavailable</span>
        </div>
      </div>
    )
  }

  return (
    <div className={`bg-gray-800 ${className}`}>
      <img
        src={thumbnail}
        alt="Model preview"
        className="w-full h-full object-cover"
      />
    </div>
  )
}
