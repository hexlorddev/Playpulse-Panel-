import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import toast from 'react-hot-toast'

import { User, AuthResponse, LoginRequest, RegisterRequest } from '@/types'
import { authApi } from '@/services/api'

interface AuthState {
  user: User | null
  token: string | null
  refreshToken: string | null
  isLoading: boolean
  error: string | null
}

interface AuthActions {
  login: (credentials: LoginRequest) => Promise<void>
  register: (data: RegisterRequest) => Promise<void>
  logout: () => Promise<void>
  refreshAuth: () => Promise<void>
  updateProfile: (data: Partial<User>) => Promise<void>
  clearError: () => void
  initializeAuth: () => void
}

export const useAuthStore = create<AuthState & AuthActions>()(
  persist(
    (set, get) => ({
      // State
      user: null,
      token: null,
      refreshToken: null,
      isLoading: false,
      error: null,

      // Actions
      login: async (credentials: LoginRequest) => {
        try {
          set({ isLoading: true, error: null })
          
          const response = await authApi.login(credentials)
          const { user, access_token, refresh_token } = response.data as AuthResponse
          
          set({
            user,
            token: access_token,
            refreshToken: refresh_token,
            isLoading: false,
            error: null,
          })
          
          toast.success(`Welcome back, ${user.first_name || user.username}!`)
        } catch (error: any) {
          const errorMessage = error.response?.data?.message || error.message || 'Login failed'
          set({ isLoading: false, error: errorMessage })
          toast.error(errorMessage)
          throw error
        }
      },

      register: async (data: RegisterRequest) => {
        try {
          set({ isLoading: true, error: null })
          
          const response = await authApi.register(data)
          
          set({ isLoading: false, error: null })
          toast.success('Account created successfully! Please login.')
        } catch (error: any) {
          const errorMessage = error.response?.data?.message || error.message || 'Registration failed'
          set({ isLoading: false, error: errorMessage })
          toast.error(errorMessage)
          throw error
        }
      },

      logout: async () => {
        try {
          const { token } = get()
          if (token) {
            await authApi.logout()
          }
        } catch (error) {
          console.error('Logout error:', error)
        } finally {
          set({
            user: null,
            token: null,
            refreshToken: null,
            error: null,
          })
          toast.success('Logged out successfully')
        }
      },

      refreshAuth: async () => {
        try {
          const { refreshToken } = get()
          if (!refreshToken) {
            throw new Error('No refresh token available')
          }

          const response = await authApi.refreshToken({ refresh_token: refreshToken })
          const { access_token, user } = response.data
          
          set({
            token: access_token,
            user,
            error: null,
          })
        } catch (error: any) {
          console.error('Token refresh failed:', error)
          // Clear auth state on refresh failure
          set({
            user: null,
            token: null,
            refreshToken: null,
            error: 'Session expired. Please login again.',
          })
          throw error
        }
      },

      updateProfile: async (data: Partial<User>) => {
        try {
          set({ isLoading: true, error: null })
          
          const response = await authApi.updateProfile(data)
          const updatedUser = response.data as User
          
          set({
            user: updatedUser,
            isLoading: false,
            error: null,
          })
          
          toast.success('Profile updated successfully!')
        } catch (error: any) {
          const errorMessage = error.response?.data?.message || error.message || 'Profile update failed'
          set({ isLoading: false, error: errorMessage })
          toast.error(errorMessage)
          throw error
        }
      },

      clearError: () => {
        set({ error: null })
      },

      initializeAuth: () => {
        const { token, refreshToken } = get()
        
        if (token && refreshToken) {
          // Try to refresh token on app load
          get().refreshAuth().catch(() => {
            // If refresh fails, clear auth state
            set({
              user: null,
              token: null,
              refreshToken: null,
              error: null,
            })
          })
        }
        
        set({ isLoading: false })
      },
    }),
    {
      name: 'playpulse-auth',
      partialize: (state) => ({
        user: state.user,
        token: state.token,
        refreshToken: state.refreshToken,
      }),
    }
  )
)