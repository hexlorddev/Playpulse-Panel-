#!/bin/bash

# Playpulse Panel - Ultimate Beast Setup Script
# The most advanced game server control panel deployment script
# Created by hhexlorddev

set -euo pipefail

# Script metadata
SCRIPT_VERSION="2.0.0"
PANEL_VERSION="1.0.0"
REQUIRED_DOCKER_VERSION="20.10.0"
REQUIRED_COMPOSE_VERSION="2.0.0"

# Colors and styling
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
GRAY='\033[0;90m'
NC='\033[0m' # No Color

# Configuration
INSTALL_DIR="/opt/playpulse"
CONFIG_DIR="$INSTALL_DIR/config"
DATA_DIR="$INSTALL_DIR/data"
LOGS_DIR="$INSTALL_DIR/logs"
BACKUP_DIR="$INSTALL_DIR/backups"

# Feature flags
ENABLE_MONITORING=true
ENABLE_ANALYTICS=true
ENABLE_AI_FEATURES=true
ENABLE_SECURITY=true
ENABLE_MULTI_NODE=true
ENABLE_MARKETPLACE=true

# Default passwords (will be auto-generated)
DB_PASSWORD=""
REDIS_PASSWORD=""
RABBITMQ_PASSWORD=""
GRAFANA_PASSWORD=""
INFLUXDB_PASSWORD=""
MINIO_PASSWORD=""
JWT_SECRET=""
JWT_REFRESH_SECRET=""

# Function definitions
print_banner() {
    clear
    echo -e "${PURPLE}"
    cat << "EOF"
    ██████╗ ██╗      █████╗ ██╗   ██╗██████╗ ██╗   ██╗██╗     ███████╗███████╗
    ██╔══██╗██║     ██╔══██╗╚██╗ ██╔╝██╔══██╗██║   ██║██║     ██╔════╝██╔════╝
    ██████╔╝██║     ███████║ ╚████╔╝ ██████╔╝██║   ██║██║     ███████╗█████╗  
    ██╔═══╝ ██║     ██╔══██║  ╚██╔╝  ██╔═══╝ ██║   ██║██║     ╚════██║██╔══╝  
    ██║     ███████╗██║  ██║   ██║   ██║     ╚██████╔╝███████╗███████║███████╗
    ╚═╝     ╚══════╝╚═╝  ╚═╝   ╚═╝   ╚═╝      ╚═════╝ ╚══════╝╚══════╝╚══════╝
                                                                               
    ██████╗  █████╗ ███╗   ██╗███████╗██╗                                    
    ██╔══██╗██╔══██╗████╗  ██║██╔════╝██║                                    
    ██████╔╝███████║██╔██╗ ██║█████╗  ██║                                    
    ██╔═══╝ ██╔══██║██║╚██╗██║██╔══╝  ██║                                    
    ██║     ██║  ██║██║ ╚████║███████╗███████╗                               
    ╚═╝     ╚═╝  ╚═╝╚═╝  ╚═══╝╚══════╝╚══════╝                               
EOF
    echo -e "${NC}"
    echo -e "${WHITE}The Ultimate Beast Gaming Control Panel${NC}"
    echo -e "${CYAN}Version: $PANEL_VERSION | Setup Script: $SCRIPT_VERSION${NC}"
    echo -e "${GRAY}Created by hhexlorddev - The Future of Server Management${NC}"
    echo ""
}

print_step() {
    echo -e "${BLUE}[$(date +'%H:%M:%S')] ➤${NC} $1"
}

print_success() {
    echo -e "${GREEN}[$(date +'%H:%M:%S')] ✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[$(date +'%H:%M:%S')] ⚠${NC} $1"
}

print_error() {
    echo -e "${RED}[$(date +'%H:%M:%S')] ✗${NC} $1"
}

print_info() {
    echo -e "${CYAN}[$(date +'%H:%M:%S')] ℹ${NC} $1"
}

check_root() {
    if [[ $EUID -eq 0 ]]; then
        print_warning "Running as root. It's recommended to use a non-root user with Docker permissions."
        read -p "Continue anyway? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi
}

check_system_requirements() {
    print_step "Checking system requirements..."
    
    # Check OS
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        print_success "Linux detected"
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        print_success "macOS detected"
    else
        print_error "Unsupported operating system: $OSTYPE"
        exit 1
    fi
    
    # Check memory
    local memory_gb=$(free -g | awk '/^Mem:/{print $2}')
    if [[ $memory_gb -lt 4 ]]; then
        print_error "Minimum 4GB RAM required. Found: ${memory_gb}GB"
        exit 1
    else
        print_success "Memory check passed: ${memory_gb}GB"
    fi
    
    # Check disk space
    local disk_gb=$(df -BG / | awk 'NR==2{print $4}' | sed 's/G//')
    if [[ $disk_gb -lt 20 ]]; then
        print_error "Minimum 20GB free disk space required. Found: ${disk_gb}GB"
        exit 1
    else
        print_success "Disk space check passed: ${disk_gb}GB available"
    fi
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed. Please install Docker first."
        echo "Visit: https://docs.docker.com/get-docker/"
        exit 1
    fi
    
    # Check Docker version
    local docker_version=$(docker --version | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
    if ! version_compare "$docker_version" "$REQUIRED_DOCKER_VERSION"; then
        print_error "Docker version $REQUIRED_DOCKER_VERSION or higher required. Found: $docker_version"
        exit 1
    fi
    print_success "Docker version check passed: $docker_version"
    
    # Check Docker Compose
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        print_error "Docker Compose is not installed. Please install Docker Compose first."
        exit 1
    fi
    
    # Check if Docker daemon is running
    if ! docker info &> /dev/null; then
        print_error "Docker daemon is not running. Please start Docker first."
        exit 1
    fi
    print_success "Docker daemon is running"
    
    # Check ports
    check_ports
    
    print_success "System requirements check completed"
}

check_ports() {
    local ports=(80 443 3000 8080 8090 5432 6379 9090 3001 8086 9200 5601 8200 9000 9001)
    local occupied_ports=()
    
    for port in "${ports[@]}"; do
        if netstat -tuln 2>/dev/null | grep -q ":$port " || ss -tuln 2>/dev/null | grep -q ":$port "; then
            occupied_ports+=($port)
        fi
    done
    
    if [[ ${#occupied_ports[@]} -gt 0 ]]; then
        print_warning "The following ports are already in use: ${occupied_ports[*]}"
        print_info "You may need to stop other services or modify the configuration"
        read -p "Continue anyway? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi
}

version_compare() {
    local version1=$1
    local version2=$2
    
    if [[ "$(printf '%s\n' "$version1" "$version2" | sort -V | head -n1)" == "$version2" ]]; then
        return 0
    else
        return 1
    fi
}

select_deployment_mode() {
    print_step "Select deployment mode..."
    echo ""
    echo "1) 🚀 Full Beast Mode - All features enabled (Recommended)"
    echo "2) 🎮 Gaming Focus - Core gaming features only"
    echo "3) 📊 Analytics Focus - Core + monitoring + analytics"
    echo "4) 🔒 Security Focus - Core + security + monitoring"
    echo "5) 🌐 Multi-Node - Distributed deployment"
    echo "6) 💻 Development - All features + development tools"
    echo "7) 🎯 Custom - Choose specific features"
    echo ""
    
    while true; do
        read -p "Enter your choice (1-7): " choice
        case $choice in
            1)
                DEPLOYMENT_MODE="full"
                ENABLE_MONITORING=true
                ENABLE_ANALYTICS=true
                ENABLE_AI_FEATURES=true
                ENABLE_SECURITY=true
                ENABLE_MULTI_NODE=true
                ENABLE_MARKETPLACE=true
                break
                ;;
            2)
                DEPLOYMENT_MODE="gaming"
                ENABLE_MONITORING=false
                ENABLE_ANALYTICS=false
                ENABLE_AI_FEATURES=false
                ENABLE_SECURITY=false
                ENABLE_MULTI_NODE=false
                ENABLE_MARKETPLACE=true
                break
                ;;
            3)
                DEPLOYMENT_MODE="analytics"
                ENABLE_MONITORING=true
                ENABLE_ANALYTICS=true
                ENABLE_AI_FEATURES=false
                ENABLE_SECURITY=false
                ENABLE_MULTI_NODE=false
                ENABLE_MARKETPLACE=true
                break
                ;;
            4)
                DEPLOYMENT_MODE="security"
                ENABLE_MONITORING=true
                ENABLE_ANALYTICS=false
                ENABLE_AI_FEATURES=false
                ENABLE_SECURITY=true
                ENABLE_MULTI_NODE=false
                ENABLE_MARKETPLACE=true
                break
                ;;
            5)
                DEPLOYMENT_MODE="multinode"
                ENABLE_MONITORING=true
                ENABLE_ANALYTICS=true
                ENABLE_AI_FEATURES=true
                ENABLE_SECURITY=true
                ENABLE_MULTI_NODE=true
                ENABLE_MARKETPLACE=true
                break
                ;;
            6)
                DEPLOYMENT_MODE="development"
                ENABLE_MONITORING=true
                ENABLE_ANALYTICS=true
                ENABLE_AI_FEATURES=true
                ENABLE_SECURITY=true
                ENABLE_MULTI_NODE=true
                ENABLE_MARKETPLACE=true
                break
                ;;
            7)
                DEPLOYMENT_MODE="custom"
                select_custom_features
                break
                ;;
            *)
                print_error "Invalid choice. Please enter 1-7."
                ;;
        esac
    done
    
    print_success "Deployment mode selected: $DEPLOYMENT_MODE"
}

select_custom_features() {
    print_step "Select custom features..."
    
    read -p "Enable monitoring (Prometheus, Grafana)? (Y/n): " -n 1 -r
    echo
    ENABLE_MONITORING=[[ $REPLY =~ ^[Yy]$ || -z $REPLY ]]
    
    read -p "Enable analytics (InfluxDB, advanced metrics)? (Y/n): " -n 1 -r
    echo
    ENABLE_ANALYTICS=[[ $REPLY =~ ^[Yy]$ || -z $REPLY ]]
    
    read -p "Enable AI features (TensorFlow, predictions)? (Y/n): " -n 1 -r
    echo
    ENABLE_AI_FEATURES=[[ $REPLY =~ ^[Yy]$ || -z $REPLY ]]
    
    read -p "Enable security features (Vault, enhanced auth)? (Y/n): " -n 1 -r
    echo
    ENABLE_SECURITY=[[ $REPLY =~ ^[Yy]$ || -z $REPLY ]]
    
    read -p "Enable multi-node support? (Y/n): " -n 1 -r
    echo
    ENABLE_MULTI_NODE=[[ $REPLY =~ ^[Yy]$ || -z $REPLY ]]
    
    read -p "Enable marketplace? (Y/n): " -n 1 -r
    echo
    ENABLE_MARKETPLACE=[[ $REPLY =~ ^[Yy]$ || -z $REPLY ]]
}

generate_secure_passwords() {
    print_step "Generating secure passwords and secrets..."
    
    DB_PASSWORD=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-25)
    REDIS_PASSWORD=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-25)
    RABBITMQ_PASSWORD=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-25)
    GRAFANA_PASSWORD=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-25)
    INFLUXDB_PASSWORD=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-25)
    MINIO_PASSWORD=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-25)
    JWT_SECRET=$(openssl rand -base64 64 | tr -d "=+/")
    JWT_REFRESH_SECRET=$(openssl rand -base64 64 | tr -d "=+/")
    
    print_success "Secure passwords generated"
}

setup_directories() {
    print_step "Setting up directory structure..."
    
    # Create main directories
    sudo mkdir -p "$INSTALL_DIR" "$CONFIG_DIR" "$DATA_DIR" "$LOGS_DIR" "$BACKUP_DIR"
    
    # Create data subdirectories
    sudo mkdir -p "$DATA_DIR"/{postgres,redis,servers,backups,uploads,prometheus,grafana,influxdb,elasticsearch,vault,minio}
    
    # Set permissions
    sudo chown -R $USER:$USER "$INSTALL_DIR"
    sudo chmod -R 755 "$INSTALL_DIR"
    
    print_success "Directory structure created"
}

create_environment_file() {
    print_step "Creating environment configuration..."
    
    cat > .env << EOF
# Playpulse Panel Environment Configuration
# Generated on $(date)

# Deployment
DEPLOYMENT_MODE=$DEPLOYMENT_MODE
DATA_PATH=$DATA_DIR

# Database
DB_PASSWORD=$DB_PASSWORD
POSTGRES_DB=playpulse_panel
POSTGRES_USER=playpulse

# Redis
REDIS_PASSWORD=$REDIS_PASSWORD

# RabbitMQ
RABBITMQ_PASSWORD=$RABBITMQ_PASSWORD

# JWT Secrets
JWT_SECRET=$JWT_SECRET
JWT_REFRESH_SECRET=$JWT_REFRESH_SECRET

# API URLs
API_URL=http://localhost:8080/api/v1
WS_URL=ws://localhost:8080/ws
FRONTEND_URL=http://localhost:3000

# External API Keys (configure these after installation)
CURSEFORGE_API_KEY=
MODRINTH_API_KEY=
GITHUB_TOKEN=

# Monitoring
GRAFANA_PASSWORD=$GRAFANA_PASSWORD

# Analytics
INFLUXDB_PASSWORD=$INFLUXDB_PASSWORD
INFLUXDB_TOKEN=playpulse-super-secret-auth-token

# Security
VAULT_ROOT_TOKEN=playpulse-vault-token

# Object Storage
MINIO_ROOT_USER=playpulse
MINIO_ROOT_PASSWORD=$MINIO_PASSWORD

# Node Management
NODE_ID=primary-node
NODE_NAME=Primary Node
NODE_LOCATION=local
NODE_TOKEN=secure-node-token-$(openssl rand -hex 16)

# Development
JUPYTER_TOKEN=playpulse-jupyter-token

# Game Manager
GAME_MANAGER_TOKEN=secure-game-manager-token-$(openssl rand -hex 16)

# Feature Flags
ENABLE_MONITORING=$ENABLE_MONITORING
ENABLE_ANALYTICS=$ENABLE_ANALYTICS
ENABLE_AI_FEATURES=$ENABLE_AI_FEATURES
ENABLE_SECURITY=$ENABLE_SECURITY
ENABLE_MULTI_NODE=$ENABLE_MULTI_NODE
ENABLE_MARKETPLACE=$ENABLE_MARKETPLACE
EOF
    
    print_success "Environment file created"
}

build_docker_images() {
    print_step "Building Docker images..."
    
    # Backend
    print_info "Building backend image..."
    docker build -t playpulse-backend:latest ./backend --target production
    
    # Frontend
    print_info "Building frontend image..."
    docker build -t playpulse-frontend:latest ./frontend --target production
    
    # Node Agent
    if [[ $ENABLE_MULTI_NODE == true ]]; then
        print_info "Building node agent image..."
        docker build -t playpulse-node-agent:latest ./nodes/agent
    fi
    
    print_success "Docker images built successfully"
}

start_core_services() {
    print_step "Starting core services..."
    
    # Start database first
    print_info "Starting PostgreSQL..."
    docker-compose -f docker-compose-ultimate.yml up -d postgres
    
    # Wait for database
    print_info "Waiting for database to be ready..."
    while ! docker-compose -f docker-compose-ultimate.yml exec -T postgres pg_isready -U playpulse -d playpulse_panel &>/dev/null; do
        sleep 2
        echo -n "."
    done
    echo ""
    
    # Start Redis
    print_info "Starting Redis..."
    docker-compose -f docker-compose-ultimate.yml up -d redis
    
    # Start RabbitMQ
    print_info "Starting RabbitMQ..."
    docker-compose -f docker-compose-ultimate.yml up -d rabbitmq
    
    print_success "Core services started"
}

start_application_services() {
    print_step "Starting application services..."
    
    # Backend
    print_info "Starting backend API..."
    docker-compose -f docker-compose-ultimate.yml up -d backend
    
    # Wait for backend
    print_info "Waiting for backend to be ready..."
    local retries=0
    while ! curl -sf http://localhost:8080/health &>/dev/null; do
        if [[ $retries -gt 30 ]]; then
            print_error "Backend failed to start within 60 seconds"
            return 1
        fi
        sleep 2
        ((retries++))
        echo -n "."
    done
    echo ""
    
    # Frontend
    print_info "Starting frontend..."
    docker-compose -f docker-compose-ultimate.yml up -d frontend
    
    print_success "Application services started"
}

start_optional_services() {
    print_step "Starting optional services..."
    
    local profiles=()
    
    if [[ $ENABLE_MONITORING == true ]]; then
        profiles+=(monitoring)
        print_info "Starting monitoring services..."
    fi
    
    if [[ $ENABLE_ANALYTICS == true ]]; then
        profiles+=(analytics)
        print_info "Starting analytics services..."
    fi
    
    if [[ $ENABLE_AI_FEATURES == true ]]; then
        profiles+=(ai)
        print_info "Starting AI services..."
    fi
    
    if [[ $ENABLE_SECURITY == true ]]; then
        profiles+=(security)
        print_info "Starting security services..."
    fi
    
    if [[ $DEPLOYMENT_MODE == "development" ]]; then
        profiles+=(development)
        print_info "Starting development services..."
    fi
    
    # Start services with profiles
    for profile in "${profiles[@]}"; do
        docker-compose -f docker-compose-ultimate.yml --profile "$profile" up -d
    done
    
    # Start additional services
    if [[ $ENABLE_MULTI_NODE == true ]]; then
        print_info "Starting node agent..."
        docker-compose -f docker-compose-ultimate.yml up -d node-agent
    fi
    
    # MinIO (always needed for file storage)
    print_info "Starting object storage..."
    docker-compose -f docker-compose-ultimate.yml up -d minio
    
    print_success "Optional services started"
}

initialize_admin_user() {
    print_step "Initializing secure admin system..."
    
    # Make CLI tool executable
    chmod +x ./cli/admin/playpulse-admin
    
    # Copy CLI tool to system
    sudo cp ./cli/admin/playpulse-admin /usr/local/bin/
    
    # Initialize security
    print_info "Initializing security infrastructure..."
    sudo playpulse-admin init-security
    
    print_success "Security infrastructure initialized"
    print_warning "To create the master admin user, run:"
    print_info "sudo playpulse-admin create-master-user"
}

setup_ssl_certificates() {
    print_step "Setting up SSL certificates..."
    
    # Create SSL directory
    mkdir -p ./docker/ssl
    
    # Generate self-signed certificate for development
    if [[ ! -f ./docker/ssl/playpulse.crt ]]; then
        print_info "Generating self-signed SSL certificate..."
        openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
            -keyout ./docker/ssl/playpulse.key \
            -out ./docker/ssl/playpulse.crt \
            -subj "/C=US/ST=State/L=City/O=Playpulse/CN=localhost"
        
        print_success "Self-signed SSL certificate generated"
        print_warning "For production, replace with valid SSL certificates"
    fi
}

configure_firewall() {
    print_step "Configuring firewall (optional)..."
    
    if command -v ufw &> /dev/null; then
        read -p "Configure UFW firewall? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            print_info "Configuring UFW firewall..."
            
            # Allow SSH
            sudo ufw allow ssh
            
            # Allow HTTP/HTTPS
            sudo ufw allow 80/tcp
            sudo ufw allow 443/tcp
            
            # Allow panel ports
            sudo ufw allow 3000/tcp  # Frontend
            sudo ufw allow 8080/tcp  # Backend API
            
            # Enable firewall
            sudo ufw --force enable
            
            print_success "Firewall configured"
        fi
    fi
}

run_health_checks() {
    print_step "Running health checks..."
    
    local failed_checks=()
    
    # Check core services
    print_info "Checking core services..."
    
    if ! curl -sf http://localhost:8080/health &>/dev/null; then
        failed_checks+=("Backend API")
    fi
    
    if ! curl -sf http://localhost:3000 &>/dev/null; then
        failed_checks+=("Frontend")
    fi
    
    if ! docker-compose -f docker-compose-ultimate.yml exec -T postgres pg_isready -U playpulse &>/dev/null; then
        failed_checks+=("PostgreSQL")
    fi
    
    if ! docker-compose -f docker-compose-ultimate.yml exec -T redis redis-cli ping &>/dev/null; then
        failed_checks+=("Redis")
    fi
    
    # Check optional services
    if [[ $ENABLE_MONITORING == true ]]; then
        if ! curl -sf http://localhost:9090 &>/dev/null; then
            failed_checks+=("Prometheus")
        fi
        if ! curl -sf http://localhost:3001 &>/dev/null; then
            failed_checks+=("Grafana")
        fi
    fi
    
    if [[ ${#failed_checks[@]} -eq 0 ]]; then
        print_success "All health checks passed"
        return 0
    else
        print_warning "Some services failed health checks: ${failed_checks[*]}"
        return 1
    fi
}

show_completion_info() {
    local all_healthy=$1
    
    echo ""
    echo -e "${GREEN}🎉 Playpulse Panel Installation Complete!${NC}"
    echo "=========================================="
    echo ""
    
    if [[ $all_healthy == true ]]; then
        echo -e "${GREEN}✅ All services are healthy and running${NC}"
    else
        echo -e "${YELLOW}⚠️  Some services may need attention${NC}"
    fi
    
    echo ""
    echo -e "${CYAN}🌐 Access Your Panel:${NC}"
    echo "   📱 Web Interface:    http://localhost:3000"
    echo "   🔗 API Endpoint:     http://localhost:8080"
    echo "   📊 API Health:       http://localhost:8080/health"
    
    if [[ $ENABLE_MONITORING == true ]]; then
        echo ""
        echo -e "${CYAN}📊 Monitoring Dashboard:${NC}"
        echo "   📈 Grafana:          http://localhost:3001"
        echo "   📉 Prometheus:       http://localhost:9090"
    fi
    
    if [[ $ENABLE_SECURITY == true ]]; then
        echo ""
        echo -e "${CYAN}🔒 Security Services:${NC}"
        echo "   🛡️  Vault:           http://localhost:8200"
    fi
    
    echo ""
    echo -e "${CYAN}💾 Object Storage:${NC}"
    echo "   📦 MinIO Console:    http://localhost:9001"
    echo "   🔑 Username:         playpulse"
    echo "   🔑 Password:         $MINIO_PASSWORD"
    
    if [[ $DEPLOYMENT_MODE == "development" ]]; then
        echo ""
        echo -e "${CYAN}🛠️  Development Tools:${NC}"
        echo "   🗄️  Adminer:         http://localhost:8083"
        echo "   📮 MailHog:          http://localhost:8025"
        echo "   📡 Redis Commander: http://localhost:8084"
    fi
    
    echo ""
    echo -e "${CYAN}🔐 Admin Account Setup:${NC}"
    echo "   Run: ${WHITE}sudo playpulse-admin create-master-user${NC}"
    echo "   This creates the first admin account (ONE TIME ONLY)"
    echo ""
    
    echo -e "${CYAN}📋 Useful Commands:${NC}"
    echo "   🔄 Restart all:      docker-compose -f docker-compose-ultimate.yml restart"
    echo "   📝 View logs:        docker-compose -f docker-compose-ultimate.yml logs -f"
    echo "   🛑 Stop all:         docker-compose -f docker-compose-ultimate.yml down"
    echo "   📊 Service status:   docker-compose -f docker-compose-ultimate.yml ps"
    echo "   👥 Admin management: sudo playpulse-admin --help"
    echo ""
    
    echo -e "${YELLOW}⚠️  Important Security Notes:${NC}"
    echo "   🔐 Change default passwords immediately!"
    echo "   🛡️  Configure SSL certificates for production"
    echo "   🔑 Set up your external API keys in .env file"
    echo "   🚪 Configure firewall rules for your environment"
    echo "   📁 Backup your configuration and data regularly"
    echo ""
    
    echo -e "${CYAN}📚 Next Steps:${NC}"
    echo "   1. Create master admin: sudo playpulse-admin create-master-user"
    echo "   2. Access web interface: http://localhost:3000"
    echo "   3. Configure external API keys in .env file"
    echo "   4. Set up SSL certificates for production"
    echo "   5. Create your first game server!"
    echo ""
    
    echo -e "${PURPLE}🚀 Created by hhexlorddev${NC}"
    echo "   🐙 GitHub: https://github.com/hhexlorddev"
    echo "   📧 Email:  contact@hhexlorddev.com"
    echo "   💬 Discord: https://discord.gg/playpulse"
    echo ""
    
    echo -e "${WHITE}Enjoy your beast-level game server control panel! 🎮${NC}"
}

cleanup_on_error() {
    print_error "Installation failed. Cleaning up..."
    
    # Stop any running containers
    docker-compose -f docker-compose-ultimate.yml down 2>/dev/null || true
    
    # Remove generated files
    rm -f .env 2>/dev/null || true
    
    print_info "Cleanup completed. Check the error messages above for details."
    exit 1
}

# Main execution
main() {
    # Trap errors
    trap cleanup_on_error ERR
    
    print_banner
    
    # Preflight checks
    check_root
    check_system_requirements
    
    # Configuration
    select_deployment_mode
    generate_secure_passwords
    setup_directories
    create_environment_file
    setup_ssl_certificates
    
    # Build and deploy
    build_docker_images
    start_core_services
    start_application_services
    start_optional_services
    
    # Post-installation
    initialize_admin_user
    configure_firewall
    
    # Verification
    local healthy=true
    if ! run_health_checks; then
        healthy=false
    fi
    
    # Completion
    show_completion_info $healthy
}

# Run main function
main "$@"