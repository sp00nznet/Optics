import { useEffect, useRef } from 'react'
import * as THREE from 'three'
import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls.js'
import { STLLoader } from 'three/examples/jsm/loaders/STLLoader.js'

interface ModelViewerProps {
  modelUrl: string
  className?: string
}

export default function ModelViewer({ modelUrl, className = '' }: ModelViewerProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const sceneRef = useRef<{
    scene: THREE.Scene
    camera: THREE.PerspectiveCamera
    renderer: THREE.WebGLRenderer
    controls: OrbitControls
    animationId: number
  } | null>(null)

  useEffect(() => {
    if (!containerRef.current) return

    const container = containerRef.current
    const width = container.clientWidth
    const height = container.clientHeight

    // Scene
    const scene = new THREE.Scene()
    scene.background = new THREE.Color(0x1f2937)

    // Camera
    const camera = new THREE.PerspectiveCamera(75, width / height, 0.1, 1000)
    camera.position.set(50, 50, 50)

    // Renderer
    const renderer = new THREE.WebGLRenderer({ antialias: true })
    renderer.setSize(width, height)
    renderer.setPixelRatio(window.devicePixelRatio)
    container.appendChild(renderer.domElement)

    // Controls
    const controls = new OrbitControls(camera, renderer.domElement)
    controls.enableDamping = true
    controls.dampingFactor = 0.05

    // Lights
    const ambientLight = new THREE.AmbientLight(0xffffff, 0.6)
    scene.add(ambientLight)

    const directionalLight = new THREE.DirectionalLight(0xffffff, 0.8)
    directionalLight.position.set(50, 50, 50)
    scene.add(directionalLight)

    const directionalLight2 = new THREE.DirectionalLight(0xffffff, 0.4)
    directionalLight2.position.set(-50, -50, -50)
    scene.add(directionalLight2)

    // Grid helper
    const gridHelper = new THREE.GridHelper(100, 20, 0x444444, 0x333333)
    scene.add(gridHelper)

    // Load STL
    const loader = new STLLoader()
    loader.load(
      modelUrl,
      (geometry) => {
        geometry.computeVertexNormals()

        // Center geometry at origin
        geometry.center()

        // Compute bounding box after centering
        geometry.computeBoundingBox()
        const boundingBox = geometry.boundingBox!
        const size = new THREE.Vector3()
        boundingBox.getSize(size)

        // Scale to fit in view (target size ~40 units)
        const maxDim = Math.max(size.x, size.y, size.z)
        const scale = 40 / maxDim

        const material = new THREE.MeshPhongMaterial({
          color: 0x3b82f6,
          specular: 0x111111,
          shininess: 200,
        })

        const mesh = new THREE.Mesh(geometry, material)

        // Rotate to correct orientation (STL files typically use Z-up, Three.js uses Y-up)
        mesh.rotation.x = -Math.PI / 2

        mesh.scale.setScalar(scale)

        // Recompute bounding box after rotation for proper positioning
        const box = new THREE.Box3().setFromObject(mesh)
        const meshSize = new THREE.Vector3()
        box.getSize(meshSize)

        // Position so bottom of model sits on grid (Y=0)
        mesh.position.y = -box.min.y

        scene.add(mesh)

        // Adjust camera to frame the model
        const distance = 60
        const centerY = meshSize.y / 2
        camera.position.set(distance, distance * 0.7, distance)
        camera.lookAt(0, centerY, 0)
        controls.target.set(0, centerY, 0)
        controls.update()
      },
      undefined,
      (error) => {
        console.error('Error loading STL:', error)
      }
    )

    // Animation loop
    const animate = () => {
      sceneRef.current!.animationId = requestAnimationFrame(animate)
      controls.update()
      renderer.render(scene, camera)
    }

    sceneRef.current = { scene, camera, renderer, controls, animationId: 0 }
    animate()

    // Handle resize
    const handleResize = () => {
      const width = container.clientWidth
      const height = container.clientHeight
      camera.aspect = width / height
      camera.updateProjectionMatrix()
      renderer.setSize(width, height)
    }
    window.addEventListener('resize', handleResize)

    // Cleanup
    return () => {
      window.removeEventListener('resize', handleResize)
      if (sceneRef.current) {
        cancelAnimationFrame(sceneRef.current.animationId)
        sceneRef.current.renderer.dispose()
        sceneRef.current.controls.dispose()
      }
      container.removeChild(renderer.domElement)
    }
  }, [modelUrl])

  return (
    <div
      ref={containerRef}
      className={`w-full h-96 rounded-lg overflow-hidden ${className}`}
    />
  )
}
