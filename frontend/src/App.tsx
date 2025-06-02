import React from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import { motion, AnimatePresence } from 'framer-motion'

// Store
import { useAuthStore } from '@/stores/authStore'
import { useThemeStore } from '@/stores/themeStore'

// Components
import Layout from '@/components/Layout'
import LoadingSpinner from '@/components/LoadingSpinner'

// Pages
import LoginPage from '@/pages/auth/LoginPage'
import RegisterPage from '@/pages/auth/RegisterPage'
import DashboardPage from '@/pages/DashboardPage'
import ServersPage from '@/pages/servers/ServersPage'
import ServerDetailPage from '@/pages/servers/ServerDetailPage'
import ServerConsolePage from '@/pages/servers/ServerConsolePage'
import ServerFilesPage from '@/pages/servers/ServerFilesPage'
import ServerPluginsPage from '@/pages/servers/ServerPluginsPage'
import ServerBackupsPage from '@/pages/servers/ServerBackupsPage'
import ServerSchedulesPage from '@/pages/servers/ServerSchedulesPage'
import ServerSettingsPage from '@/pages/servers/ServerSettingsPage'
import ProfilePage from '@/pages/ProfilePage'
import SettingsPage from '@/pages/SettingsPage'
import NotFoundPage from '@/pages/NotFoundPage'

// Hooks
import { useEffect } from 'react'

function App() {
  const { user, isLoading, initializeAuth } = useAuthStore()
  const { theme, initializeTheme } = useThemeStore()

  useEffect(() => {
    initializeAuth()
    initializeTheme()
  }, [initializeAuth, initializeTheme])

  useEffect(() => {
    // Apply theme to document
    const root = document.documentElement
    root.classList.remove('light', 'dark')
    
    if (theme === 'system') {
      const systemTheme = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
      root.classList.add(systemTheme)
    } else {
      root.classList.add(theme)
    }
  }, [theme])

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-950">
        <div className="text-center">
          <LoadingSpinner size="lg" />
          <p className="mt-4 text-gray-600 dark:text-gray-400">Loading Playpulse Panel...</p>
        </div>
      </div>
    )
  }

  return (
    <AnimatePresence mode="wait">
      <div className="min-h-screen bg-gray-50 dark:bg-gray-950">
        {!user ? (
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route path="/register" element={<RegisterPage />} />
            <Route path="*" element={<Navigate to="/login" replace />} />
          </Routes>
        ) : (
          <Layout>
            <Routes>
              <Route path="/" element={<Navigate to="/dashboard" replace />} />
              <Route path="/dashboard" element={<DashboardPage />} />
              
              {/* Server Routes */}
              <Route path="/servers" element={<ServersPage />} />
              <Route path="/servers/:serverId" element={<ServerDetailPage />} />
              <Route path="/servers/:serverId/console" element={<ServerConsolePage />} />
              <Route path="/servers/:serverId/files" element={<ServerFilesPage />} />
              <Route path="/servers/:serverId/plugins" element={<ServerPluginsPage />} />
              <Route path="/servers/:serverId/backups" element={<ServerBackupsPage />} />
              <Route path="/servers/:serverId/schedules" element={<ServerSchedulesPage />} />
              <Route path="/servers/:serverId/settings" element={<ServerSettingsPage />} />
              
              {/* User Routes */}
              <Route path="/profile" element={<ProfilePage />} />
              <Route path="/settings" element={<SettingsPage />} />
              
              {/* Auth Routes */}
              <Route path="/login" element={<Navigate to="/dashboard" replace />} />
              <Route path="/register" element={<Navigate to="/dashboard" replace />} />
              
              {/* 404 */}
              <Route path="*" element={<NotFoundPage />} />
            </Routes>
          </Layout>
        )}
      </div>
    </AnimatePresence>
  )
}

export default App