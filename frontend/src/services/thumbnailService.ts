import * as THREE from 'three'
import { STLLoader } from 'three/examples/jsm/loaders/STLLoader.js'

interface ThumbnailRequest {
  url: string
  resolve: (dataUrl: string | null) => void
}

class ThumbnailService {
  private renderer: THREE.WebGLRenderer | null = null
  private scene: THREE.Scene | null = null
  private camera: THREE.PerspectiveCamera | null = null
  private cache: Map<string, string> = new Map()
  private queue: ThumbnailRequest[] = []
  private processing = false
  private readonly size = 256

  private initialize() {
    if (this.renderer) return

    // Create offscreen canvas
    const canvas = document.createElement('canvas')
    canvas.width = this.size
    canvas.height = this.size

    // Renderer
    this.renderer = new THREE.WebGLRenderer({
      canvas,
      antialias: true,
      preserveDrawingBuffer: true,
    })
    this.renderer.setSize(this.size, this.size)
    this.renderer.setPixelRatio(1)

    // Scene
    this.scene = new THREE.Scene()
    this.scene.background = new THREE.Color(0x1f2937)

    // Camera
    this.camera = new THREE.PerspectiveCamera(45, 1, 0.1, 1000)
    this.camera.position.set(3, 2, 3)
    this.camera.lookAt(0, 0, 0)

    // Lights
    const ambientLight = new THREE.AmbientLight(0xffffff, 0.6)
    this.scene.add(ambientLight)

    const directionalLight = new THREE.DirectionalLight(0xffffff, 0.8)
    directionalLight.position.set(1, 1, 1)
    this.scene.add(directionalLight)

    const directionalLight2 = new THREE.DirectionalLight(0xffffff, 0.3)
    directionalLight2.position.set(-1, -1, -1)
    this.scene.add(directionalLight2)
  }

  async getThumbnail(url: string): Promise<string | null> {
    // Check cache first
    if (this.cache.has(url)) {
      return this.cache.get(url)!
    }

    // Add to queue
    return new Promise((resolve) => {
      this.queue.push({ url, resolve })
      this.processQueue()
    })
  }

  private async processQueue() {
    if (this.processing || this.queue.length === 0) return

    this.processing = true
    this.initialize()

    while (this.queue.length > 0) {
      const request = this.queue.shift()!

      // Check cache again (might have been processed while waiting)
      if (this.cache.has(request.url)) {
        request.resolve(this.cache.get(request.url)!)
        continue
      }

      try {
        const dataUrl = await this.renderThumbnail(request.url)
        if (dataUrl) {
          this.cache.set(request.url, dataUrl)
        }
        request.resolve(dataUrl)
      } catch {
        request.resolve(null)
      }

      // Small delay to prevent blocking
      await new Promise((r) => setTimeout(r, 10))
    }

    this.processing = false
  }

  private renderThumbnail(url: string): Promise<string | null> {
    return new Promise((resolve) => {
      if (!this.renderer || !this.scene || !this.camera) {
        resolve(null)
        return
      }

      const loader = new STLLoader()
      loader.load(
        url,
        (geometry) => {
          // Clear previous mesh
          const toRemove: THREE.Object3D[] = []
          this.scene!.traverse((child) => {
            if (child instanceof THREE.Mesh) {
              toRemove.push(child)
            }
          })
          toRemove.forEach((obj) => this.scene!.remove(obj))

          // Process geometry
          geometry.computeVertexNormals()
          geometry.center()
          geometry.rotateX(-Math.PI / 2)

          geometry.computeBoundingBox()
          const boundingBox = geometry.boundingBox!
          const size = new THREE.Vector3()
          boundingBox.getSize(size)

          // Scale to fit
          const maxDim = Math.max(size.x, size.y, size.z)
          const scale = 2 / maxDim

          const material = new THREE.MeshPhongMaterial({
            color: 0x3b82f6,
            specular: 0x111111,
            shininess: 100,
          })

          const mesh = new THREE.Mesh(geometry, material)
          mesh.scale.setScalar(scale)
          mesh.rotation.y = Math.PI / 6

          this.scene!.add(mesh)

          // Render
          this.renderer!.render(this.scene!, this.camera!)

          // Get data URL
          const dataUrl = this.renderer!.domElement.toDataURL('image/png')

          // Cleanup geometry
          geometry.dispose()
          material.dispose()

          resolve(dataUrl)
        },
        undefined,
        () => {
          resolve(null)
        }
      )
    })
  }

  clearCache() {
    this.cache.clear()
  }

  dispose() {
    if (this.renderer) {
      this.renderer.dispose()
      this.renderer = null
    }
    this.scene = null
    this.camera = null
    this.cache.clear()
  }
}

// Singleton instance
export const thumbnailService = new ThumbnailService()
