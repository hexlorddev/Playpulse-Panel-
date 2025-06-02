import axios from 'axios'
import toast from 'react-hot-toast'

import { 
  ApiResponse, 
  LoginRequest, 
  RegisterRequest, 
  AuthResponse, 
  User,
  Server,
  CreateServerRequest,
  UpdateServerRequest,
  ServerStats,
  Plugin,
  Backup,
  Schedule,
  ServerFile,
  Notification,
  AuditLog,
  DashboardStats,
} from '@/types'

// Create axios instance
const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || '/api/v1',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor to add auth token
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('playpulse-auth')
    if (token) {
      try {
        const authData = JSON.parse(token)
        if (authData.state.token) {
          config.headers.Authorization = `Bearer ${authData.state.token}`
        }
      } catch (error) {
        console.error('Error parsing auth token:', error)
      }
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Response interceptor to handle errors and token refresh
api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config

    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true

      try {
        // Try to refresh token
        const authData = localStorage.getItem('playpulse-auth')
        if (authData) {
          const parsed = JSON.parse(authData)
          const refreshToken = parsed.state.refreshToken

          if (refreshToken) {
            const response = await axios.post('/api/v1/auth/refresh', {
              refresh_token: refreshToken,
            })

            const { access_token } = response.data
            
            // Update stored token
            parsed.state.token = access_token
            localStorage.setItem('playpulse-auth', JSON.stringify(parsed))

            // Retry original request
            originalRequest.headers.Authorization = `Bearer ${access_token}`
            return api(originalRequest)
          }
        }
      } catch (refreshError) {
        // Refresh failed, clear auth and redirect to login
        localStorage.removeItem('playpulse-auth')
        window.location.href = '/login'
        return Promise.reject(refreshError)
      }
    }

    // Handle other error responses
    if (error.response?.status >= 500) {
      toast.error('Server error. Please try again later.')
    } else if (error.response?.status === 404) {
      toast.error('Resource not found.')
    } else if (error.response?.status === 403) {
      toast.error('Access denied.')
    }

    return Promise.reject(error)
  }
)

// Auth API
export const authApi = {
  login: (credentials: LoginRequest) => 
    api.post<AuthResponse>('/auth/login', credentials),
  
  register: (data: RegisterRequest) => 
    api.post<ApiResponse>('/auth/register', data),
  
  logout: () => 
    api.post<ApiResponse>('/auth/logout'),
  
  refreshToken: (data: { refresh_token: string }) => 
    api.post<ApiResponse>('/auth/refresh', data),
  
  me: () => 
    api.get<User>('/auth/me'),
  
  updateProfile: (data: Partial<User>) => 
    api.put<User>('/auth/profile', data),
  
  changePassword: (data: { current_password: string; new_password: string }) => 
    api.put<ApiResponse>('/auth/password', data),
}

// Server API
export const serverApi = {
  getServers: () => 
    api.get<Server[]>('/servers'),
  
  getServer: (serverId: string) => 
    api.get<Server>(`/servers/${serverId}`),
  
  createServer: (data: CreateServerRequest) => 
    api.post<Server>('/servers', data),
  
  updateServer: (serverId: string, data: UpdateServerRequest) => 
    api.put<Server>(`/servers/${serverId}`, data),
  
  deleteServer: (serverId: string, deleteFiles = false) => 
    api.delete<ApiResponse>(`/servers/${serverId}?delete_files=${deleteFiles}`),
  
  startServer: (serverId: string) => 
    api.post<ApiResponse>(`/servers/${serverId}/start`),
  
  stopServer: (serverId: string) => 
    api.post<ApiResponse>(`/servers/${serverId}/stop`),
  
  restartServer: (serverId: string) => 
    api.post<ApiResponse>(`/servers/${serverId}/restart`),
  
  sendCommand: (serverId: string, command: string) => 
    api.post<ApiResponse>(`/servers/${serverId}/command`, { command }),
  
  getLogs: (serverId: string, lines = 100) => 
    api.get<{ logs: string[]; lines: number }>(`/servers/${serverId}/logs?lines=${lines}`),
  
  getStats: (serverId: string) => 
    api.get<ServerStats>(`/servers/${serverId}/stats`),
}

// File API
export const fileApi = {
  getFiles: (serverId: string, path = '/') => 
    api.get<ServerFile[]>(`/servers/${serverId}/files?path=${encodeURIComponent(path)}`),
  
  getFile: (serverId: string, filePath: string) => 
    api.get<{ content: string }>(`/servers/${serverId}/files/content?path=${encodeURIComponent(filePath)}`),
  
  saveFile: (serverId: string, filePath: string, content: string) => 
    api.put<ApiResponse>(`/servers/${serverId}/files/content`, { path: filePath, content }),
  
  deleteFile: (serverId: string, filePath: string) => 
    api.delete<ApiResponse>(`/servers/${serverId}/files?path=${encodeURIComponent(filePath)}`),
  
  createFolder: (serverId: string, folderPath: string) => 
    api.post<ApiResponse>(`/servers/${serverId}/files/folder`, { path: folderPath }),
  
  uploadFile: (serverId: string, formData: FormData) => 
    api.post<ApiResponse>(`/servers/${serverId}/files/upload`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    }),
  
  downloadFile: (serverId: string, filePath: string) => 
    api.get(`/servers/${serverId}/files/download?path=${encodeURIComponent(filePath)}`, {
      responseType: 'blob',
    }),
}

// Plugin API
export const pluginApi = {
  getPlugins: (serverId: string) => 
    api.get<Plugin[]>(`/servers/${serverId}/plugins`),
  
  installPlugin: (serverId: string, data: { source: string; id: string; version?: string }) => 
    api.post<Plugin>(`/servers/${serverId}/plugins`, data),
  
  togglePlugin: (serverId: string, pluginId: string, enabled: boolean) => 
    api.patch<ApiResponse>(`/servers/${serverId}/plugins/${pluginId}`, { is_enabled: enabled }),
  
  deletePlugin: (serverId: string, pluginId: string) => 
    api.delete<ApiResponse>(`/servers/${serverId}/plugins/${pluginId}`),
  
  searchCurseForge: (query: string, gameVersion?: string) => 
    api.get<any[]>(`/plugins/search/curseforge?query=${encodeURIComponent(query)}&version=${gameVersion || ''}`),
  
  searchModrinth: (query: string, gameVersion?: string) => 
    api.get<any[]>(`/plugins/search/modrinth?query=${encodeURIComponent(query)}&version=${gameVersion || ''}`),
}

// Backup API
export const backupApi = {
  getBackups: (serverId: string) => 
    api.get<Backup[]>(`/servers/${serverId}/backups`),
  
  createBackup: (serverId: string, name: string, description?: string) => 
    api.post<Backup>(`/servers/${serverId}/backups`, { name, description }),
  
  restoreBackup: (serverId: string, backupId: string) => 
    api.post<ApiResponse>(`/servers/${serverId}/backups/${backupId}/restore`),
  
  deleteBackup: (serverId: string, backupId: string) => 
    api.delete<ApiResponse>(`/servers/${serverId}/backups/${backupId}`),
  
  downloadBackup: (serverId: string, backupId: string) => 
    api.get(`/servers/${serverId}/backups/${backupId}/download`, {
      responseType: 'blob',
    }),
}

// Schedule API
export const scheduleApi = {
  getSchedules: (serverId: string) => 
    api.get<Schedule[]>(`/servers/${serverId}/schedules`),
  
  createSchedule: (serverId: string, data: Omit<Schedule, 'id' | 'server_id' | 'created_at' | 'updated_at'>) => 
    api.post<Schedule>(`/servers/${serverId}/schedules`, data),
  
  updateSchedule: (serverId: string, scheduleId: string, data: Partial<Schedule>) => 
    api.put<Schedule>(`/servers/${serverId}/schedules/${scheduleId}`, data),
  
  deleteSchedule: (serverId: string, scheduleId: string) => 
    api.delete<ApiResponse>(`/servers/${serverId}/schedules/${scheduleId}`),
  
  toggleSchedule: (serverId: string, scheduleId: string, active: boolean) => 
    api.patch<ApiResponse>(`/servers/${serverId}/schedules/${scheduleId}`, { is_active: active }),
}

// Dashboard API
export const dashboardApi = {
  getStats: () => 
    api.get<DashboardStats>('/dashboard/stats'),
  
  getRecentActivity: (limit = 10) => 
    api.get<AuditLog[]>(`/dashboard/activity?limit=${limit}`),
  
  getServerMetrics: (timeRange = '24h') => 
    api.get<any>(`/dashboard/metrics?range=${timeRange}`),
}

// Notification API
export const notificationApi = {
  getNotifications: (limit = 50, unreadOnly = false) => 
    api.get<Notification[]>(`/notifications?limit=${limit}&unread_only=${unreadOnly}`),
  
  markAsRead: (notificationId: string) => 
    api.patch<ApiResponse>(`/notifications/${notificationId}/read`),
  
  markAllAsRead: () => 
    api.patch<ApiResponse>('/notifications/read-all'),
  
  deleteNotification: (notificationId: string) => 
    api.delete<ApiResponse>(`/notifications/${notificationId}`),
}

// Admin API (for admin users)
export const adminApi = {
  getUsers: () => 
    api.get<User[]>('/admin/users'),
  
  getAuditLogs: (limit = 100) => 
    api.get<AuditLog[]>(`/admin/audit?limit=${limit}`),
  
  getSystemStats: () => 
    api.get<any>('/admin/stats'),
}

// Export the main API instance
export default api