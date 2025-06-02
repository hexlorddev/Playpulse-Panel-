import React from 'react'
import { WebSocketMessage, ConsoleMessage, StatsMessage, StatusMessage } from '@/types'

type WebSocketEventHandler = (data: any) => void

interface WebSocketHandlers {
  [key: string]: WebSocketEventHandler[]
}

class WebSocketService {
  private ws: WebSocket | null = null
  private reconnectAttempts = 0
  private maxReconnectAttempts = 5
  private reconnectInterval = 5000
  private handlers: WebSocketHandlers = {}
  private url: string
  private isConnecting = false
  private heartbeatInterval: NodeJS.Timeout | null = null

  constructor() {
    this.url = import.meta.env.VITE_WS_URL || 'ws://localhost:8080/ws'
  }

  connect() {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      return
    }

    if (this.isConnecting) {
      return
    }

    this.isConnecting = true

    try {
      this.ws = new WebSocket(this.url)

      this.ws.onopen = () => {
        console.log('WebSocket connected')
        this.isConnecting = false
        this.reconnectAttempts = 0
        this.startHeartbeat()
        this.emit('connected', null)
      }

      this.ws.onmessage = (event) => {
        try {
          const message: WebSocketMessage = JSON.parse(event.data)
          this.handleMessage(message)
        } catch (error) {
          console.error('Error parsing WebSocket message:', error)
        }
      }

      this.ws.onclose = (event) => {
        console.log('WebSocket disconnected:', event.code, event.reason)
        this.isConnecting = false
        this.stopHeartbeat()
        this.emit('disconnected', { code: event.code, reason: event.reason })
        
        if (event.code !== 1000 && this.reconnectAttempts < this.maxReconnectAttempts) {
          this.scheduleReconnect()
        }
      }

      this.ws.onerror = (error) => {
        console.error('WebSocket error:', error)
        this.isConnecting = false
        this.emit('error', error)
      }
    } catch (error) {
      console.error('Error creating WebSocket connection:', error)
      this.isConnecting = false
      this.scheduleReconnect()
    }
  }

  disconnect() {
    if (this.ws) {
      this.ws.close(1000, 'Client disconnect')
      this.ws = null
    }
    this.stopHeartbeat()
    this.reconnectAttempts = this.maxReconnectAttempts // Prevent reconnection
  }

  send(message: any) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message))
    } else {
      console.warn('WebSocket is not connected')
    }
  }

  // Subscribe to server updates
  subscribeToServer(serverId: string) {
    this.send({
      type: 'subscribe_server',
      data: { server_id: serverId },
    })
  }

  // Unsubscribe from server updates
  unsubscribeFromServer(serverId: string) {
    this.send({
      type: 'unsubscribe_server',
      data: { server_id: serverId },
    })
  }

  // Send command to server
  sendServerCommand(serverId: string, command: string) {
    this.send({
      type: 'send_command',
      data: { server_id: serverId, command },
    })
  }

  // Send ping
  ping() {
    this.send({ type: 'ping', data: {} })
  }

  // Event handling
  on(event: string, handler: WebSocketEventHandler) {
    if (!this.handlers[event]) {
      this.handlers[event] = []
    }
    this.handlers[event].push(handler)
  }

  off(event: string, handler: WebSocketEventHandler) {
    if (this.handlers[event]) {
      this.handlers[event] = this.handlers[event].filter(h => h !== handler)
    }
  }

  private emit(event: string, data: any) {
    if (this.handlers[event]) {
      this.handlers[event].forEach(handler => handler(data))
    }
  }

  private handleMessage(message: WebSocketMessage) {
    switch (message.type) {
      case 'welcome':
        console.log('WebSocket welcome:', message.data)
        break

      case 'console_log':
        this.emit('console_log', {
          serverId: message.server_id,
          message: message.data as ConsoleMessage,
        })
        break

      case 'server_stats':
        this.emit('server_stats', {
          serverId: message.server_id,
          stats: message.data as StatsMessage,
        })
        break

      case 'server_status':
        this.emit('server_status', {
          serverId: message.server_id,
          status: message.data as StatusMessage,
        })
        break

      case 'command_sent':
        this.emit('command_sent', {
          serverId: message.server_id,
          data: message.data,
        })
        break

      case 'subscribed':
        this.emit('subscribed', {
          serverId: message.server_id,
          message: message.data,
        })
        break

      case 'unsubscribed':
        this.emit('unsubscribed', {
          serverId: message.server_id,
          message: message.data,
        })
        break

      case 'error':
        this.emit('ws_error', message.data)
        console.error('WebSocket error from server:', message.data)
        break

      case 'pong':
        // Handle pong response
        break

      default:
        console.warn('Unknown WebSocket message type:', message.type)
    }
  }

  private scheduleReconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('Max reconnection attempts reached')
      this.emit('max_reconnect_attempts', null)
      return
    }

    this.reconnectAttempts++
    const delay = this.reconnectInterval * Math.pow(2, this.reconnectAttempts - 1) // Exponential backoff
    
    console.log(`Attempting to reconnect in ${delay}ms (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`)
    
    setTimeout(() => {
      this.connect()
    }, delay)
  }

  private startHeartbeat() {
    this.heartbeatInterval = setInterval(() => {
      this.ping()
    }, 30000) // Ping every 30 seconds
  }

  private stopHeartbeat() {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval)
      this.heartbeatInterval = null
    }
  }

  // Getters
  get isConnected() {
    return this.ws && this.ws.readyState === WebSocket.OPEN
  }

  get connectionState() {
    if (!this.ws) return 'disconnected'
    
    switch (this.ws.readyState) {
      case WebSocket.CONNECTING:
        return 'connecting'
      case WebSocket.OPEN:
        return 'connected'
      case WebSocket.CLOSING:
        return 'closing'
      case WebSocket.CLOSED:
        return 'disconnected'
      default:
        return 'unknown'
    }
  }
}

// Create singleton instance
export const websocketService = new WebSocketService()

// React hook for WebSocket
export const useWebSocket = () => {
  const [isConnected, setIsConnected] = React.useState(websocketService.isConnected)
  const [connectionState, setConnectionState] = React.useState(websocketService.connectionState)

  React.useEffect(() => {
    const handleConnected = () => {
      setIsConnected(true)
      setConnectionState('connected')
    }

    const handleDisconnected = () => {
      setIsConnected(false)
      setConnectionState('disconnected')
    }

    const handleError = () => {
      setConnectionState('error')
    }

    websocketService.on('connected', handleConnected)
    websocketService.on('disconnected', handleDisconnected)
    websocketService.on('error', handleError)

    // Connect if not already connected
    if (!websocketService.isConnected) {
      websocketService.connect()
    }

    return () => {
      websocketService.off('connected', handleConnected)
      websocketService.off('disconnected', handleDisconnected)
      websocketService.off('error', handleError)
    }
  }, [])

  return {
    isConnected,
    connectionState,
    subscribe: websocketService.subscribeToServer.bind(websocketService),
    unsubscribe: websocketService.unsubscribeFromServer.bind(websocketService),
    sendCommand: websocketService.sendServerCommand.bind(websocketService),
    on: websocketService.on.bind(websocketService),
    off: websocketService.off.bind(websocketService),
  }
}

export default websocketService