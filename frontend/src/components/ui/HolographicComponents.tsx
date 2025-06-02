import React, { useState, useEffect, useRef } from 'react'
import { motion, AnimatePresence, useAnimation } from 'framer-motion'
import { Canvas, useFrame } from '@react-three/fiber'
import { Sphere, MeshDistortMaterial, OrbitControls } from '@react-three/drei'

// Holographic Server Card Component
export const HolographicServerCard: React.FC<{
  server: any
  onClick?: () => void
}> = ({ server, onClick }) => {
  const [isHovered, setIsHovered] = useState(false)
  const controls = useAnimation()
  
  useEffect(() => {
    if (isHovered) {
      controls.start({
        rotateY: 5,
        rotateX: 5,
        scale: 1.05,
        boxShadow: '0 20px 60px rgba(59, 130, 246, 0.3)',
      })
    } else {
      controls.start({
        rotateY: 0,
        rotateX: 0,
        scale: 1,
        boxShadow: '0 8px 32px rgba(0, 0, 0, 0.1)',
      })
    }
  }, [isHovered, controls])

  return (
    <motion.div
      className="relative w-full h-64 rounded-2xl overflow-hidden cursor-pointer"
      style={{
        background: 'linear-gradient(135deg, rgba(255,255,255,0.1) 0%, rgba(255,255,255,0.05) 100%)',
        backdropFilter: 'blur(20px)',
        border: '1px solid rgba(255,255,255,0.2)',
      }}
      animate={controls}
      onHoverStart={() => setIsHovered(true)}
      onHoverEnd={() => setIsHovered(false)}
      onClick={onClick}
      whileTap={{ scale: 0.95 }}
    >
      {/* Holographic Background */}
      <div className="absolute inset-0">
        <div className="absolute inset-0 bg-gradient-to-r from-blue-500/20 via-purple-500/20 to-pink-500/20 animate-pulse" />
        <div className="absolute inset-0 bg-gradient-to-br from-cyan-400/10 to-blue-600/10" />
      </div>
      
      {/* Floating Particles */}
      <div className="absolute inset-0">
        {[...Array(20)].map((_, i) => (
          <motion.div
            key={i}
            className="absolute w-1 h-1 bg-blue-400 rounded-full"
            style={{
              left: `${Math.random() * 100}%`,
              top: `${Math.random() * 100}%`,
            }}
            animate={{
              y: [0, -20, 0],
              opacity: [0.3, 1, 0.3],
            }}
            transition={{
              duration: 2 + Math.random() * 2,
              repeat: Infinity,
              delay: Math.random() * 2,
            }}
          />
        ))}
      </div>

      {/* Content */}
      <div className="relative z-10 p-6 h-full flex flex-col justify-between">
        <div>
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-xl font-bold text-white">{server.name}</h3>
            <StatusOrb status={server.status} />
          </div>
          
          <p className="text-blue-100 text-sm mb-4">{server.description}</p>
          
          <div className="flex items-center space-x-4 text-sm">
            <div className="flex items-center">
              <div className="w-2 h-2 bg-green-400 rounded-full mr-2" />
              <span className="text-green-100">{server.players || 0} players</span>
            </div>
            <div className="flex items-center">
              <div className="w-2 h-2 bg-blue-400 rounded-full mr-2" />
              <span className="text-blue-100">{server.type}</span>
            </div>
          </div>
        </div>

        {/* Performance Bars */}
        <div className="space-y-2">
          <PerformanceBar label="CPU" value={server.cpu_usage || 0} color="blue" />
          <PerformanceBar label="RAM" value={server.memory_usage || 0} color="purple" />
          <PerformanceBar label="TPS" value={(server.tps || 20) / 20 * 100} color="green" />
        </div>
      </div>

      {/* Holographic Border Effect */}
      <motion.div
        className="absolute inset-0 rounded-2xl"
        style={{
          background: 'linear-gradient(45deg, transparent, rgba(59, 130, 246, 0.3), transparent)',
          mask: 'linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0)',
          maskComposite: 'xor',
          padding: '2px',
        }}
        animate={{
          rotate: isHovered ? 360 : 0,
        }}
        transition={{
          duration: 8,
          repeat: Infinity,
          ease: 'linear',
        }}
      />
    </motion.div>
  )
}

// Status Orb with Pulsing Animation
const StatusOrb: React.FC<{ status: string }> = ({ status }) => {
  const statusColors = {
    running: 'bg-green-400 shadow-green-400/50',
    stopped: 'bg-red-400 shadow-red-400/50',
    starting: 'bg-yellow-400 shadow-yellow-400/50',
    crashed: 'bg-red-600 shadow-red-600/50',
  }

  return (
    <motion.div
      className={`w-4 h-4 rounded-full ${statusColors[status as keyof typeof statusColors] || statusColors.stopped}`}
      animate={{
        scale: [1, 1.2, 1],
        boxShadow: [
          '0 0 0 0 currentColor',
          '0 0 0 8px transparent',
          '0 0 0 0 transparent',
        ],
      }}
      transition={{
        duration: 2,
        repeat: Infinity,
      }}
    />
  )
}

// Performance Bar with Liquid Animation
const PerformanceBar: React.FC<{
  label: string
  value: number
  color: string
}> = ({ label, value, color }) => {
  const colorMap = {
    blue: 'from-blue-400 to-blue-600',
    purple: 'from-purple-400 to-purple-600',
    green: 'from-green-400 to-green-600',
    red: 'from-red-400 to-red-600',
  }

  return (
    <div className="flex items-center justify-between">
      <span className="text-xs text-white/80 w-8">{label}</span>
      <div className="flex-1 mx-2 h-2 bg-white/10 rounded-full overflow-hidden">
        <motion.div
          className={`h-full bg-gradient-to-r ${colorMap[color as keyof typeof colorMap]}`}
          initial={{ width: 0 }}
          animate={{ width: `${Math.min(value, 100)}%` }}
          transition={{ duration: 1, ease: 'easeOut' }}
        />
      </div>
      <span className="text-xs text-white/80 w-8 text-right">{Math.round(value)}%</span>
    </div>
  )
}

// Quantum Loading Animation
export const QuantumLoader: React.FC<{ size?: 'sm' | 'md' | 'lg' }> = ({ size = 'md' }) => {
  const sizes = {
    sm: 'w-8 h-8',
    md: 'w-12 h-12',
    lg: 'w-16 h-16',
  }

  return (
    <div className={`relative ${sizes[size]} mx-auto`}>
      {/* Outer Ring */}
      <motion.div
        className="absolute inset-0 rounded-full border-2 border-blue-500/30"
        animate={{
          rotate: 360,
          scale: [1, 1.1, 1],
        }}
        transition={{
          rotate: { duration: 2, repeat: Infinity, ease: 'linear' },
          scale: { duration: 1, repeat: Infinity },
        }}
      />
      
      {/* Inner Ring */}
      <motion.div
        className="absolute inset-2 rounded-full border-2 border-purple-500/50"
        animate={{
          rotate: -360,
          scale: [1, 0.9, 1],
        }}
        transition={{
          rotate: { duration: 1.5, repeat: Infinity, ease: 'linear' },
          scale: { duration: 1.5, repeat: Infinity },
        }}
      />
      
      {/* Core */}
      <motion.div
        className="absolute inset-4 rounded-full bg-gradient-to-r from-blue-400 to-purple-600"
        animate={{
          scale: [0.8, 1.2, 0.8],
          opacity: [0.5, 1, 0.5],
        }}
        transition={{
          duration: 1,
          repeat: Infinity,
        }}
      />
      
      {/* Quantum Particles */}
      {[...Array(8)].map((_, i) => (
        <motion.div
          key={i}
          className="absolute w-1 h-1 bg-cyan-400 rounded-full"
          style={{
            left: '50%',
            top: '50%',
            marginLeft: '-2px',
            marginTop: '-2px',
          }}
          animate={{
            x: [0, Math.cos(i * Math.PI / 4) * 30, 0],
            y: [0, Math.sin(i * Math.PI / 4) * 30, 0],
            opacity: [0, 1, 0],
          }}
          transition={{
            duration: 2,
            repeat: Infinity,
            delay: i * 0.25,
          }}
        />
      ))}
    </div>
  )
}

// Neural Network Background
export const NeuralBackground: React.FC = () => {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  
  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    
    const ctx = canvas.getContext('2d')
    if (!ctx) return
    
    canvas.width = window.innerWidth
    canvas.height = window.innerHeight
    
    const nodes: Array<{
      x: number
      y: number
      vx: number
      vy: number
      energy: number
    }> = []
    
    // Create nodes
    for (let i = 0; i < 100; i++) {
      nodes.push({
        x: Math.random() * canvas.width,
        y: Math.random() * canvas.height,
        vx: (Math.random() - 0.5) * 2,
        vy: (Math.random() - 0.5) * 2,
        energy: Math.random(),
      })
    }
    
    const animate = () => {
      ctx.clearRect(0, 0, canvas.width, canvas.height)
      
      // Update nodes
      nodes.forEach(node => {
        node.x += node.vx
        node.y += node.vy
        
        // Bounce off edges
        if (node.x < 0 || node.x > canvas.width) node.vx *= -1
        if (node.y < 0 || node.y > canvas.height) node.vy *= -1
        
        // Keep in bounds
        node.x = Math.max(0, Math.min(canvas.width, node.x))
        node.y = Math.max(0, Math.min(canvas.height, node.y))
        
        // Update energy
        node.energy += (Math.random() - 0.5) * 0.02
        node.energy = Math.max(0, Math.min(1, node.energy))
      })
      
      // Draw connections
      for (let i = 0; i < nodes.length; i++) {
        for (let j = i + 1; j < nodes.length; j++) {
          const dx = nodes[i].x - nodes[j].x
          const dy = nodes[i].y - nodes[j].y
          const distance = Math.sqrt(dx * dx + dy * dy)
          
          if (distance < 150) {
            const opacity = (1 - distance / 150) * 0.5
            const energy = (nodes[i].energy + nodes[j].energy) / 2
            
            ctx.strokeStyle = `rgba(59, 130, 246, ${opacity * energy})`
            ctx.lineWidth = 1
            ctx.beginPath()
            ctx.moveTo(nodes[i].x, nodes[i].y)
            ctx.lineTo(nodes[j].x, nodes[j].y)
            ctx.stroke()
          }
        }
      }
      
      // Draw nodes
      nodes.forEach(node => {
        const size = 2 + node.energy * 3
        const opacity = 0.6 + node.energy * 0.4
        
        ctx.fillStyle = `rgba(59, 130, 246, ${opacity})`
        ctx.beginPath()
        ctx.arc(node.x, node.y, size, 0, Math.PI * 2)
        ctx.fill()
      })
      
      requestAnimationFrame(animate)
    }
    
    animate()
    
    const handleResize = () => {
      canvas.width = window.innerWidth
      canvas.height = window.innerHeight
    }
    
    window.addEventListener('resize', handleResize)
    return () => window.removeEventListener('resize', handleResize)
  }, [])
  
  return (
    <canvas
      ref={canvasRef}
      className="fixed inset-0 pointer-events-none opacity-30 z-0"
    />
  )
}

// 3D Holographic Terminal
export const HolographicTerminal: React.FC<{
  logs: string[]
  onCommand?: (command: string) => void
}> = ({ logs, onCommand }) => {
  const [command, setCommand] = useState('')
  const [isVisible, setIsVisible] = useState(true)
  const terminalRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (terminalRef.current) {
      terminalRef.current.scrollTop = terminalRef.current.scrollHeight
    }
  }, [logs])

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (command.trim() && onCommand) {
      onCommand(command.trim())
      setCommand('')
    }
  }

  return (
    <AnimatePresence>
      {isVisible && (
        <motion.div
          initial={{ opacity: 0, scale: 0.8, rotateX: -30 }}
          animate={{ opacity: 1, scale: 1, rotateX: 0 }}
          exit={{ opacity: 0, scale: 0.8, rotateX: 30 }}
          className="w-full h-full"
          style={{
            perspective: '1000px',
          }}
        >
          <div
            className="w-full h-full rounded-lg overflow-hidden"
            style={{
              background: 'linear-gradient(135deg, rgba(0,0,0,0.9) 0%, rgba(20,20,20,0.95) 100%)',
              backdropFilter: 'blur(10px)',
              border: '1px solid rgba(0,255,0,0.3)',
              boxShadow: '0 0 30px rgba(0,255,0,0.2), inset 0 0 30px rgba(0,255,0,0.1)',
            }}
          >
            {/* Terminal Header */}
            <div className="flex items-center justify-between p-3 border-b border-green-500/30">
              <div className="flex items-center space-x-2">
                <div className="w-3 h-3 bg-red-500 rounded-full"></div>
                <div className="w-3 h-3 bg-yellow-500 rounded-full"></div>
                <div className="w-3 h-3 bg-green-500 rounded-full"></div>
              </div>
              <div className="text-green-400 text-sm font-mono">Playpulse Terminal v2.0</div>
              <button
                onClick={() => setIsVisible(false)}
                className="text-green-400 hover:text-green-300"
              >
                ×
              </button>
            </div>

            {/* Terminal Content */}
            <div
              ref={terminalRef}
              className="h-64 p-4 overflow-y-auto font-mono text-sm text-green-400 custom-scrollbar"
            >
              {logs.map((log, index) => (
                <motion.div
                  key={index}
                  initial={{ opacity: 0, x: -20 }}
                  animate={{ opacity: 1, x: 0 }}
                  transition={{ delay: index * 0.05 }}
                  className="mb-1 leading-relaxed"
                >
                  <span className="text-green-600">[{new Date().toLocaleTimeString()}]</span>{' '}
                  <span className={getLogColor(log)}>{log}</span>
                </motion.div>
              ))}
            </div>

            {/* Command Input */}
            <form onSubmit={handleSubmit} className="p-4 border-t border-green-500/30">
              <div className="flex items-center">
                <span className="text-green-400 mr-2">$</span>
                <input
                  type="text"
                  value={command}
                  onChange={(e) => setCommand(e.target.value)}
                  className="flex-1 bg-transparent text-green-400 outline-none font-mono"
                  placeholder="Enter command..."
                  autoComplete="off"
                />
              </div>
            </form>
          </div>
        </motion.div>
      )}
    </AnimatePresence>
  )
}

// Advanced Performance Chart with Holographic Effect
export const HolographicChart: React.FC<{
  data: Array<{ timestamp: string; value: number }>
  title: string
  color?: string
}> = ({ data, title, color = 'blue' }) => {
  const svgRef = useRef<SVGSVGElement>(null)
  const [dimensions, setDimensions] = useState({ width: 400, height: 200 })

  useEffect(() => {
    const updateDimensions = () => {
      if (svgRef.current) {
        const { width, height } = svgRef.current.getBoundingClientRect()
        setDimensions({ width, height })
      }
    }

    updateDimensions()
    window.addEventListener('resize', updateDimensions)
    return () => window.removeEventListener('resize', updateDimensions)
  }, [])

  const maxValue = Math.max(...data.map(d => d.value))
  const minValue = Math.min(...data.map(d => d.value))
  const range = maxValue - minValue

  const getPath = (offset = 0) => {
    if (data.length < 2) return ''

    const points = data.map((point, index) => {
      const x = (index / (data.length - 1)) * dimensions.width
      const y = dimensions.height - ((point.value - minValue) / range) * dimensions.height + offset
      return `${x},${y}`
    })

    return `M ${points.join(' L ')}`
  }

  return (
    <div className="w-full h-full relative">
      <div className="absolute top-4 left-4 text-white font-medium z-10">
        {title}
      </div>
      
      <svg
        ref={svgRef}
        className="w-full h-full"
        style={{
          background: 'linear-gradient(135deg, rgba(0,0,0,0.2) 0%, rgba(20,20,20,0.4) 100%)',
        }}
      >
        <defs>
          <linearGradient id={`gradient-${color}`} x1="0%" y1="0%" x2="0%" y2="100%">
            <stop offset="0%" stopColor={`var(--${color}-400)`} stopOpacity="0.8" />
            <stop offset="100%" stopColor={`var(--${color}-600)`} stopOpacity="0.1" />
          </linearGradient>
          
          <filter id="glow">
            <feGaussianBlur stdDeviation="3" result="coloredBlur"/>
            <feMerge> 
              <feMergeNode in="coloredBlur"/>
              <feMergeNode in="SourceGraphic"/>
            </feMerge>
          </filter>
        </defs>

        {/* Grid Lines */}
        {[...Array(5)].map((_, i) => (
          <line
            key={`grid-${i}`}
            x1="0"
            y1={(i / 4) * dimensions.height}
            x2={dimensions.width}
            y2={(i / 4) * dimensions.height}
            stroke="rgba(255,255,255,0.1)"
            strokeWidth="1"
          />
        ))}

        {/* Area Fill */}
        <path
          d={`${getPath()} L ${dimensions.width},${dimensions.height} L 0,${dimensions.height} Z`}
          fill={`url(#gradient-${color})`}
        />

        {/* Main Line */}
        <motion.path
          d={getPath()}
          fill="none"
          stroke={`var(--${color}-400)`}
          strokeWidth="2"
          filter="url(#glow)"
          initial={{ pathLength: 0 }}
          animate={{ pathLength: 1 }}
          transition={{ duration: 2, ease: 'easeOut' }}
        />

        {/* Data Points */}
        {data.map((point, index) => {
          const x = (index / (data.length - 1)) * dimensions.width
          const y = dimensions.height - ((point.value - minValue) / range) * dimensions.height

          return (
            <motion.circle
              key={index}
              cx={x}
              cy={y}
              r="3"
              fill={`var(--${color}-400)`}
              initial={{ scale: 0 }}
              animate={{ scale: 1 }}
              transition={{ delay: index * 0.1 }}
              whileHover={{ scale: 1.5 }}
            />
          )
        })}
      </svg>
    </div>
  )
}

// Helper function for log coloring
const getLogColor = (log: string): string => {
  if (log.includes('ERROR') || log.includes('error')) return 'text-red-400'
  if (log.includes('WARN') || log.includes('warn')) return 'text-yellow-400'
  if (log.includes('INFO') || log.includes('info')) return 'text-blue-400'
  if (log.includes('DEBUG') || log.includes('debug')) return 'text-purple-400'
  return 'text-green-400'
}

// Morphing Navigation Menu
export const MorphingMenu: React.FC<{
  items: Array<{ name: string; icon: any; href: string }>
  activeItem: string
  onItemClick: (href: string) => void
}> = ({ items, activeItem, onItemClick }) => {
  return (
    <div className="relative">
      <div className="flex space-x-1 p-1 bg-black/20 rounded-full backdrop-blur-sm">
        {items.map((item) => (
          <motion.button
            key={item.href}
            onClick={() => onItemClick(item.href)}
            className={`relative px-4 py-2 rounded-full text-sm font-medium transition-colors ${
              activeItem === item.href
                ? 'text-white'
                : 'text-gray-400 hover:text-white'
            }`}
            whileHover={{ scale: 1.05 }}
            whileTap={{ scale: 0.95 }}
          >
            {activeItem === item.href && (
              <motion.div
                layoutId="activeBackground"
                className="absolute inset-0 bg-gradient-to-r from-blue-500 to-purple-600 rounded-full"
                initial={false}
                transition={{
                  type: 'spring',
                  stiffness: 300,
                  damping: 30,
                }}
              />
            )}
            <div className="relative flex items-center space-x-2">
              <item.icon className="w-4 h-4" />
              <span>{item.name}</span>
            </div>
          </motion.button>
        ))}
      </div>
    </div>
  )
}