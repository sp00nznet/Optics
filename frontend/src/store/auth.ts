import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { authApi, User } from '../api/client'

interface AuthState {
  user: User | null
  token: string | null
  isAuthenticated: boolean
  isLoading: boolean
  error: string | null
  login: (email: string, password: string) => Promise<void>
  register: (username: string, email: string, password: string) => Promise<void>
  logout: () => void
  checkAuth: () => Promise<void>
  clearError: () => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      token: null,
      isAuthenticated: false,
      isLoading: false,
      error: null,

      login: async (email, password) => {
        set({ isLoading: true, error: null })
        try {
          const response = await authApi.login({ email, password })
          const { token, user } = response.data
          localStorage.setItem('token', token)
          set({ user, token, isAuthenticated: true, isLoading: false })
        } catch (error: unknown) {
          const message = (error as { response?: { data?: { error?: string } } })?.response?.data?.error || 'Login failed'
          set({ error: message, isLoading: false })
          throw error
        }
      },

      register: async (username, email, password) => {
        set({ isLoading: true, error: null })
        try {
          const response = await authApi.register({ username, email, password })
          const { token, user } = response.data
          localStorage.setItem('token', token)
          set({ user, token, isAuthenticated: true, isLoading: false })
        } catch (error: unknown) {
          const message = (error as { response?: { data?: { error?: string } } })?.response?.data?.error || 'Registration failed'
          set({ error: message, isLoading: false })
          throw error
        }
      },

      logout: () => {
        localStorage.removeItem('token')
        set({ user: null, token: null, isAuthenticated: false })
      },

      checkAuth: async () => {
        const token = localStorage.getItem('token')
        if (!token) {
          set({ isAuthenticated: false })
          return
        }

        set({ isLoading: true })
        try {
          const response = await authApi.me()
          set({ user: response.data, token, isAuthenticated: true, isLoading: false })
        } catch {
          localStorage.removeItem('token')
          set({ user: null, token: null, isAuthenticated: false, isLoading: false })
        }
      },

      clearError: () => set({ error: null }),
    }),
    {
      name: 'auth-storage',
      // Persist the whole session, not just the token: without user/isAuthenticated
      // a refresh renders as logged out and bounces admins off protected routes.
      partialize: (state) => ({
        token: state.token,
        user: state.user,
        isAuthenticated: state.isAuthenticated,
      }),
    }
  )
)
