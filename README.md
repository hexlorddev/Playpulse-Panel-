# Playpulse Panel 🎮

A modern, powerful, and user-friendly game server control panel designed to manage Minecraft and other game servers. Built as a complete alternative to PufferPanel and Pterodactyl, offering superior performance and features.

**Created by hhexlorddev**

## 🌟 Features

### 🎯 Core Server Management
- ✅ Start / Stop / Restart / Soft Reload servers
- ✅ Scheduled restarts with CRON UI
- ✅ Auto-restart on crash detection
- ✅ Real-time WebSocket terminal
- ✅ Resource monitoring (RAM, CPU, TPS)
- ✅ Lag alerts and optimization hints

### 🔌 Plugin & Mod Control
- ✅ Install plugins/mods via CurseForge & Modrinth API
- ✅ Plugin update checker and rollback system
- ✅ Visual plugin manager with status and logs
- ✅ Plugin sandbox system for security

### 📁 File System & Configuration
- ✅ Full file browser with drag & drop
- ✅ Built-in code editor with syntax highlighting
- ✅ Auto-backups before file modifications
- ✅ GUI-based config editors (server.properties, etc.)

### 👥 Player Management
- ✅ Live player list with UUID, ping, inventory
- ✅ OP/De-OP, ban, kick, whitelist management
- ✅ In-game chat monitoring
- ✅ Scheduled commands to players

### 🌍 World Management
- ✅ World import/export functionality
- ✅ Multi-world support
- ✅ World switcher UI
- ✅ Automatic world backups
- ✅ Map rendering integration

### 🛡️ Security & Access Control
- ✅ JWT Authentication with session management
- ✅ 2FA (Time-based OTP)
- ✅ Role-based permissions (Admin, Mod, Viewer)
- ✅ Server permission scoping
- ✅ Comprehensive audit log system

### 🛠️ Automation & Scheduling
- ✅ CRON-based task scheduler
- ✅ Automated backup system
- ✅ Alert triggers (Discord webhooks)
- ✅ MOTD changer & favicon uploader

### 📱 Modern UI/UX
- ✅ Mobile-friendly responsive design
- ✅ Dark/Light mode toggle
- ✅ Real-time dashboard updates
- ✅ Theme customization
- ✅ Notification system

## 🏗️ Architecture

### Backend Stack
- **Language**: Go 1.21+
- **Framework**: Fiber v2 (Express-like performance)
- **Database**: PostgreSQL with GORM
- **WebSocket**: Gorilla WebSocket
- **Authentication**: JWT with refresh tokens
- **API**: RESTful with WebSocket support

### Frontend Stack
- **Framework**: React 18 with TypeScript
- **Styling**: TailwindCSS with dark/light mode
- **UI Components**: ShadCN UI library
- **State Management**: Zustand
- **Real-time**: WebSocket client
- **Charts**: Chart.js for performance graphs

### Deployment
- **Containerization**: Docker & Docker Compose
- **Reverse Proxy**: Nginx configuration
- **CI/CD**: GitHub Actions
- **Service**: systemd integration

## 🚀 Quick Start

### Automated Installation (Recommended)

The easiest way to get Playpulse Panel running is using our automated setup script:

```bash
# Clone the repository
git clone https://github.com/hhexlorddev/playpulse-panel.git
cd playpulse-panel

# Make setup script executable
chmod +x scripts/setup.sh

# Run the automated setup
./scripts/setup.sh
```

The script will:
- ✅ Check system requirements
- ✅ Set up environment configuration  
- ✅ Build Docker images
- ✅ Start all services
- ✅ Initialize the database
- ✅ Create default admin user

### Manual Installation

#### Prerequisites
- Docker 20.10+ & Docker Compose 2.0+
- 4GB+ RAM recommended
- 10GB+ disk space

#### Step 1: Clone and Configure
```bash
git clone https://github.com/hhexlorddev/playpulse-panel.git
cd playpulse-panel

# Copy environment configuration
cp backend/.env.example backend/.env

# Generate secure secrets (Linux/macOS)
JWT_SECRET=$(openssl rand -base64 32)
JWT_REFRESH_SECRET=$(openssl rand -base64 32)

# Update .env file with your secrets
sed -i "s/your-super-secret-jwt-key-change-this-in-production/$JWT_SECRET/g" backend/.env
sed -i "s/your-super-secret-refresh-key-change-this-in-production/$JWT_REFRESH_SECRET/g" backend/.env
```

#### Step 2: Start Services
```bash
# Start all services
docker-compose up -d

# Check service status
docker-compose ps

# View logs
docker-compose logs -f
```

#### Step 3: Access the Panel
- 🌐 **Web Interface**: http://localhost:3000
- 🔗 **API Endpoint**: http://localhost:8080
- 📊 **API Documentation**: http://localhost:8080/docs

**Default Admin Credentials:**
- 📧 Email: `admin@playpulse.dev`
- 🔒 Password: `admin123`

> ⚠️ **Security Warning**: Change the default password immediately after first login!

### Development Setup

For development with hot reloading:

```bash
# Start database only
docker-compose up -d postgres redis

# Backend development
cd backend
go mod download
go run main.go

# Frontend development (new terminal)  
cd frontend
npm install
npm run dev
```

### Advanced Configuration

#### Custom Ports
Edit `docker-compose.yml` to change default ports:
```yaml
services:
  frontend:
    ports:
      - "8081:80"  # Change from 3000:80
  backend:
    ports:
      - "8082:8080"  # Change from 8080:8080
```

#### SSL/TLS Setup
1. Place SSL certificates in `docker/nginx/ssl/`
2. Update `docker/nginx/nginx.conf` with SSL configuration
3. Change ports to 443 in `docker-compose.yml`

#### External Database
Update `backend/.env` to use external PostgreSQL:
```env
DB_HOST=your-postgres-host.com
DB_PORT=5432
DB_USER=your-username
DB_PASSWORD=your-password
DB_NAME=playpulse_panel
DB_SSL_MODE=require
```

#### Production Optimization
```bash
# Use production profile
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d

# Enable monitoring
docker-compose --profile monitoring up -d
```

### Monitoring & Maintenance

#### Service Management
```bash
# Restart all services
docker-compose restart

# Restart specific service
docker-compose restart backend

# Update images
docker-compose pull
docker-compose up -d

# Clean up unused data
docker system prune -a
```

#### Database Backup
```bash
# Create backup
docker-compose exec postgres pg_dump -U playpulse playpulse_panel > backup_$(date +%Y%m%d_%H%M%S).sql

# Restore backup
docker-compose exec -T postgres psql -U playpulse playpulse_panel < backup_file.sql
```

#### Log Management
```bash
# View specific service logs
docker-compose logs -f backend
docker-compose logs -f frontend

# View last 100 lines
docker-compose logs --tail=100 backend

# Export logs
docker-compose logs --no-color > playpulse-logs.txt
```

## 📖 Documentation

- [Installation Guide](docs/installation.md)
- [API Documentation](docs/api.md)
- [Configuration Guide](docs/configuration.md)
- [Plugin Development](docs/plugins.md)
- [Troubleshooting](docs/troubleshooting.md)

## 🔧 Configuration

### Environment Variables
```env
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=playpulse
DB_PASSWORD=your_password
DB_NAME=playpulse_panel

# JWT
JWT_SECRET=your-secret-key-here
JWT_REFRESH_SECRET=your-refresh-secret

# Server
PORT=8080
ENVIRONMENT=production

# External APIs
CURSEFORGE_API_KEY=your_curseforge_key
MODRINTH_API_KEY=your_modrinth_key
```

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 👨‍💻 Author

**hhexlorddev** - Creator and Lead Developer

- GitHub: [@hhexlorddev](https://github.com/hhexlorddev)
- Email: contact@hhexlorddev.com

## 🙏 Acknowledgments

- Inspired by PufferPanel and Pterodactyl
- Built for the gaming community
- Special thanks to all contributors

---

*Playpulse Panel - Powering game servers with modern technology* 🚀