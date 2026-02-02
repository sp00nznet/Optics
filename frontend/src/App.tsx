import { Routes, Route, Navigate } from 'react-router-dom'
import { useAuthStore } from './store/auth'
import Layout from './components/Layout'
import Home from './pages/Home'
import Browse from './pages/Browse'
import ModelDetail from './pages/ModelDetail'
import Categories from './pages/Categories'
import Duplicates from './pages/Duplicates'
import Admin from './pages/Admin'
import Login from './pages/Login'
import Register from './pages/Register'

function ProtectedRoute({ children, adminOnly = false }: { children: React.ReactNode, adminOnly?: boolean }) {
  const { user, isAuthenticated } = useAuthStore()

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }

  if (adminOnly && user?.role !== 'admin') {
    return <Navigate to="/" replace />
  }

  return <>{children}</>
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route path="/register" element={<Register />} />
      <Route element={<Layout />}>
        <Route path="/" element={<Home />} />
        <Route path="/browse" element={<Browse />} />
        <Route path="/models/:id" element={<ModelDetail />} />
        <Route path="/categories" element={<Categories />} />
        <Route
          path="/duplicates"
          element={
            <ProtectedRoute adminOnly>
              <Duplicates />
            </ProtectedRoute>
          }
        />
        <Route
          path="/admin"
          element={
            <ProtectedRoute adminOnly>
              <Admin />
            </ProtectedRoute>
          }
        />
      </Route>
    </Routes>
  )
}
