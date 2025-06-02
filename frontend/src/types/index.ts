// API Response Types
export interface ApiResponse<T = any> {
  data?: T
  message?: string
  error?: string
  timestamp?: string
}

// User Types
export interface User {
  id: string
  username: string
  email: string
  first_name: string
  last_name: string
  avatar?: string
  role: UserRole
  is_active: boolean
  email_verified: boolean
  two_factor_enabled: boolean
  last_login?: string
  created_at: string
  updated_at: string
  servers?: Server[]
}

export type UserRole = 'admin' | 'moderator' | 'user' | 'viewer'

// Auth Types
export interface LoginRequest {
  email: string
  password: string
}

export interface RegisterRequest {
  username: string
  email: string
  password: string
  first_name?: string
  last_name?: string
}

export interface AuthResponse {
  user: User
  access_token: string
  refresh_token: string
  expires_at: string
}

// Server Types
export interface Server {
  id: string
  name: string
  description?: string
  type: ServerType
  version?: string
  status: ServerStatus
  port: number
  memory_limit: number
  disk_limit: number
  cpu_limit: number
  path: string
  java_path?: string
  java_args?: string
  server_jar?: string
  start_command?: string
  stop_command?: string
  auto_restart: boolean
  auto_start: boolean
  backup_enabled: boolean
  last_backup?: string
  pid: number
  created_at: string
  updated_at: string
  users?: User[]
  plugins?: Plugin[]
  schedules?: Schedule[]
  backups?: Backup[]
  metrics?: ServerMetric[]
}

export type ServerType = 
  | 'minecraft' 
  | 'paper' 
  | 'spigot' 
  | 'fabric' 
  | 'forge' 
  | 'vanilla' 
  | 'bedrock' 
  | 'proxy' 
  | 'other'

export type ServerStatus = 
  | 'stopped' 
  | 'starting' 
  | 'running' 
  | 'stopping' 
  | 'crashed' 
  | 'unknown'

export interface CreateServerRequest {
  name: string
  description?: string
  type: ServerType
  version?: string
  port: number
  memory_limit: number
  disk_limit: number
  cpu_limit?: number
  java_path?: string
  java_args?: string
  auto_restart?: boolean
  auto_start?: boolean
}

export interface UpdateServerRequest {
  name?: string
  description?: string
  version?: string
  memory_limit?: number
  disk_limit?: number
  cpu_limit?: number
  java_path?: string
  java_args?: string
  start_command?: string
  stop_command?: string
  auto_restart?: boolean
  auto_start?: boolean
}

// Plugin Types
export interface Plugin {
  id: string
  server_id: string
  name: string
  version?: string
  author?: string
  description?: string
  file_name: string
  file_path: string
  file_size: number
  source: PluginSource
  source_id?: string
  is_enabled: boolean
  dependencies?: string[]
  install_date: string
  update_date?: string
  created_at: string
  updated_at: string
}

export type PluginSource = 'curseforge' | 'modrinth' | 'manual' | 'github'

// Schedule Types
export interface Schedule {
  id: string
  server_id: string
  name: string
  action: ScheduleAction
  command?: string
  cron_pattern: string
  is_active: boolean
  last_run?: string
  next_run?: string
  run_count: number
  created_at: string
  updated_at: string
}

export type ScheduleAction = 'restart' | 'stop' | 'start' | 'command' | 'backup'

// Backup Types
export interface Backup {
  id: string
  server_id: string
  name: string
  description?: string
  path: string
  size: number
  type: BackupType
  status: BackupStatus
  created_at: string
  updated_at: string
}

export type BackupType = 'manual' | 'scheduled' | 'automatic'
export type BackupStatus = 'creating' | 'completed' | 'failed'

// Metrics Types
export interface ServerMetric {
  id: string
  server_id: string
  cpu_usage: number
  memory_usage: number
  disk_usage: number
  network_in: number
  network_out: number
  player_count: number
  tps: number
  mspt: number
  timestamp: string
}

export interface ServerStats {
  cpu_usage: number
  memory_usage: number
  memory_limit: number
  disk_usage: number
  disk_limit: number
  network_in: number
  network_out: number
  player_count: number
  tps: number
  mspt: number
  uptime: number
  is_online: boolean
}

// File Types
export interface ServerFile {
  id: string
  server_id: string
  path: string
  name: string
  size: number
  type: FileType
  mime_type: string
  is_hidden: boolean
  created_at: string
  updated_at: string
}

export type FileType = 'file' | 'directory' | 'symlink'

// WebSocket Types
export interface WebSocketMessage {
  type: string
  server_id?: string
  data: any
  timestamp: string
}

export interface ConsoleMessage {
  line: string
  timestamp: string
  type: 'info' | 'warn' | 'error'
}

export interface StatsMessage {
  server_id: string
  cpu_usage: number
  memory_usage: number
  disk_usage: number
  player_count: number
  tps: number
  status: string
}

export interface StatusMessage {
  server_id: string
  status: string
  message: string
}

// Notification Types
export interface Notification {
  id: string
  user_id: string
  server_id?: string
  title: string
  message: string
  type: NotificationType
  priority: NotificationPriority
  is_read: boolean
  created_at: string
  read_at?: string
}

export type NotificationType = 'info' | 'warning' | 'error' | 'success'
export type NotificationPriority = 'low' | 'medium' | 'high'

// System Settings
export interface SystemSetting {
  id: string
  key: string
  value: string
  type: string
  category: string
  created_at: string
  updated_at: string
}

// Audit Log Types
export interface AuditLog {
  id: string
  user_id: string
  server_id?: string
  action: string
  details: string
  ip_address: string
  user_agent: string
  created_at: string
  user?: User
  server?: Server
}

// Dashboard Types
export interface DashboardStats {
  total_servers: number
  running_servers: number
  total_users: number
  total_backups: number
  total_disk_usage: number
  total_memory_usage: number
  recent_activity: AuditLog[]
}

// Theme Types
export type Theme = 'light' | 'dark' | 'system'

// Navigation Types
export interface NavItem {
  name: string
  href: string
  icon: any
  current?: boolean
  children?: NavItem[]
}

// Form Types
export interface FormErrors {
  [key: string]: string | undefined
}

// Modal Types
export interface ModalProps {
  isOpen: boolean
  onClose: () => void
  title?: string
  children: React.ReactNode
}

// Loading States
export interface LoadingState {
  loading: boolean
  error?: string
}

// Pagination
export interface Pagination {
  page: number
  limit: number
  total: number
  total_pages: number
}

export interface PaginatedResponse<T> {
  data: T[]
  pagination: Pagination
}

// Error Types
export interface ApiError {
  error: string
  message: string
  timestamp?: string
  path?: string
  method?: string
}