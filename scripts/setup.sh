#!/bin/bash

# Playpulse Panel Setup Script
# Created by hhexlorddev

set -e

echo "🎮 Playpulse Panel Setup Script"
echo "================================"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Functions
print_step() {
    echo -e "${BLUE}➤${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_info() {
    echo -e "${CYAN}ℹ${NC} $1"
}

check_requirements() {
    print_step "Checking system requirements..."
    
    # Check if Docker is installed
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed. Please install Docker first."
        echo "Visit: https://docs.docker.com/get-docker/"
        exit 1
    fi
    
    # Check if Docker Compose is installed
    if ! command -v docker-compose &> /dev/null; then
        print_error "Docker Compose is not installed. Please install Docker Compose first."
        echo "Visit: https://docs.docker.com/compose/install/"
        exit 1
    fi
    
    # Check if ports are available
    if netstat -tuln | grep -q ':80 '; then
        print_warning "Port 80 is already in use. You may need to stop other services."
    fi
    
    if netstat -tuln | grep -q ':8080 '; then
        print_warning "Port 8080 is already in use. You may need to stop other services."
    fi
    
    print_success "System requirements check completed"
}

setup_environment() {
    print_step "Setting up environment configuration..."
    
    # Create .env file from example
    if [ ! -f "backend/.env" ]; then
        cp backend/.env.example backend/.env
        print_success "Created backend/.env from example"
    else
        print_info "Backend .env file already exists"
    fi
    
    # Generate secure secrets
    JWT_SECRET=$(openssl rand -base64 32)
    JWT_REFRESH_SECRET=$(openssl rand -base64 32)
    
    # Update .env with generated secrets
    sed -i "s/your-super-secret-jwt-key-change-this-in-production/$JWT_SECRET/g" backend/.env
    sed -i "s/your-super-secret-refresh-key-change-this-in-production/$JWT_REFRESH_SECRET/g" backend/.env
    
    print_success "Generated secure JWT secrets"
}

build_images() {
    print_step "Building Docker images..."
    
    # Build backend
    print_info "Building backend image..."
    docker build -t playpulse-backend ./backend
    
    # Build frontend
    print_info "Building frontend image..."
    docker build -t playpulse-frontend ./frontend
    
    print_success "Docker images built successfully"
}

start_services() {
    print_step "Starting services with Docker Compose..."
    
    # Start database first
    print_info "Starting PostgreSQL database..."
    docker-compose up -d postgres
    
    # Wait for database to be ready
    print_info "Waiting for database to be ready..."
    sleep 10
    
    # Start all services
    print_info "Starting all services..."
    docker-compose up -d
    
    print_success "All services started"
}

wait_for_services() {
    print_step "Waiting for services to be ready..."
    
    # Wait for backend
    print_info "Waiting for backend API..."
    timeout=60
    while [ $timeout -gt 0 ]; do
        if curl -s http://localhost:8080/health > /dev/null 2>&1; then
            break
        fi
        sleep 1
        ((timeout--))
    done
    
    if [ $timeout -eq 0 ]; then
        print_error "Backend API failed to start within 60 seconds"
        exit 1
    fi
    
    print_success "Backend API is ready"
    
    # Wait for frontend
    print_info "Waiting for frontend..."
    timeout=60
    while [ $timeout -gt 0 ]; do
        if curl -s http://localhost:3000 > /dev/null 2>&1; then
            break
        fi
        sleep 1
        ((timeout--))
    done
    
    if [ $timeout -eq 0 ]; then
        print_warning "Frontend may not be ready yet, but continuing..."
    else
        print_success "Frontend is ready"
    fi
}

show_completion_info() {
    echo ""
    echo -e "${GREEN}🎉 Playpulse Panel Setup Complete!${NC}"
    echo "=================================="
    echo ""
    echo -e "${CYAN}📱 Access your panel:${NC}"
    echo "   🌐 Web Interface: http://localhost:3000"
    echo "   🔗 API Endpoint:  http://localhost:8080"
    echo "   📊 API Docs:      http://localhost:8080/docs"
    echo ""
    echo -e "${CYAN}🔑 Default Admin Credentials:${NC}"
    echo "   📧 Email:    admin@playpulse.dev"
    echo "   🔒 Password: admin123"
    echo ""
    echo -e "${YELLOW}⚠️  Security Notice:${NC}"
    echo "   🔐 Please change the default admin password immediately!"
    echo "   🔑 Update JWT secrets in production environments"
    echo "   🛡️  Configure SSL/TLS for production use"
    echo ""
    echo -e "${CYAN}📖 Useful Commands:${NC}"
    echo "   🔄 Restart services:    docker-compose restart"
    echo "   📋 View logs:           docker-compose logs -f"
    echo "   🛑 Stop services:       docker-compose down"
    echo "   📊 Service status:      docker-compose ps"
    echo ""
    echo -e "${PURPLE}🚀 Created by hhexlorddev${NC}"
    echo "   🐙 GitHub: https://github.com/hhexlorddev"
    echo "   📧 Email:  contact@hhexlorddev.com"
    echo ""
}

cleanup_on_error() {
    print_error "Setup failed. Cleaning up..."
    docker-compose down 2>/dev/null || true
    exit 1
}

# Trap errors
trap cleanup_on_error ERR

# Main execution
main() {
    echo -e "${PURPLE}"
    cat << "EOF"
    ____  __                        __           
   / __ \/ /___ ___  ____  __  __/ /________ 
  / /_/ / / __ `/ / / / __ \/ / / / / ___/ _ \
 / ____/ / /_/ / /_/ / /_/ / /_/ / (__  )  __/
/_/   /_/\__,_/\__, / .___/\__,_/_/____/\___/ 
              /____/_/                        
   ____                  __
  / __ \____ _____  ___  / /
 / /_/ / __ `/ __ \/ _ \/ / 
/ ____/ /_/ / / / /  __/ /  
/_/    \__,_/_/ /_/\___/_/   
                            
EOF
    echo -e "${NC}"
    
    check_requirements
    setup_environment
    build_images
    start_services
    wait_for_services
    show_completion_info
}

# Check if running as root
if [ "$EUID" -eq 0 ]; then
    print_warning "Running as root. Consider using a non-root user with Docker permissions."
fi

# Run main function
main