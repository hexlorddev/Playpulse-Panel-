import { create } from 'zustand'
import { persist } from 'zustand/middleware'

import { Theme } from '@/types'

interface ThemeState {
  theme: Theme
}

interface ThemeActions {
  setTheme: (theme: Theme) => void
  initializeTheme: () => void
}

export const useThemeStore = create<ThemeState & ThemeActions>()(
  persist(
    (set, get) => ({
      // State
      theme: 'dark',

      // Actions
      setTheme: (theme: Theme) => {
        set({ theme })
        
        // Apply theme immediately
        const root = document.documentElement
        root.classList.remove('light', 'dark')
        
        if (theme === 'system') {
          const systemTheme = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
          root.classList.add(systemTheme)
        } else {
          root.classList.add(theme)
        }
      },

      initializeTheme: () => {
        const { theme } = get()
        
        // Listen for system theme changes
        if (theme === 'system') {
          const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
          const handleChange = () => {
            const root = document.documentElement
            root.classList.remove('light', 'dark')
            root.classList.add(mediaQuery.matches ? 'dark' : 'light')
          }
          
          mediaQuery.addEventListener('change', handleChange)
          handleChange() // Apply initial theme
        }
      },
    }),
    {
      name: 'playpulse-theme',
    }
  )
)