#!/bin/bash

# Playpulse Panel - Pre-deployment Verification
# Created by hhexlorddev

echo "🔍 Playpulse Panel - Pre-deployment Verification"
echo "================================================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

checks_passed=0
total_checks=0

check_item() {
    ((total_checks++))
    if [ "$1" = true ]; then
        echo -e "${GREEN}✓${NC} $2"
        ((checks_passed++))
    else
        echo -e "${RED}✗${NC} $2"
    fi
}

info_item() {
    echo -e "${BLUE}ℹ${NC} $1"
}

echo "🐳 Docker Environment"
echo "---------------------"
check_item "$(command -v docker &> /dev/null && echo true || echo false)" "Docker is installed"
check_item "$(command -v docker-compose &> /dev/null && echo true || echo false)" "Docker Compose is installed"
check_item "$(docker info &> /dev/null && echo true || echo false)" "Docker daemon is running"

echo ""
echo "📁 Project Structure"
echo "--------------------"
check_item "$([ -f "backend/main.go" ] && echo true || echo false)" "Backend main.go exists"
check_item "$([ -f "backend/go.mod" ] && echo true || echo false)" "Backend go.mod exists"
check_item "$([ -f "backend/Dockerfile" ] && echo true || echo false)" "Backend Dockerfile exists"
check_item "$([ -f "frontend/package.json" ] && echo true || echo false)" "Frontend package.json exists"
check_item "$([ -f "frontend/Dockerfile" ] && echo true || echo false)" "Frontend Dockerfile exists"
check_item "$([ -f "docker-compose.yml" ] && echo true || echo false)" "Docker Compose file exists"

echo ""
echo "⚙️ Configuration"
echo "----------------"
check_item "$([ -f "backend/.env.example" ] && echo true || echo false)" "Backend .env.example exists"
check_item "$([ -x "scripts/setup.sh" ] && echo true || echo false)" "Setup script is executable"

echo ""
echo "🔧 Core Files"
echo "-------------"
check_item "$([ -f "backend/config/config.go" ] && echo true || echo false)" "Configuration system"
check_item "$([ -f "backend/database/database.go" ] && echo true || echo false)" "Database layer"
check_item "$([ -f "backend/models/models.go" ] && echo true || echo false)" "Data models"
check_item "$([ -f "backend/services/server_manager.go" ] && echo true || echo false)" "Server management service"
check_item "$([ -f "backend/services/websocket.go" ] && echo true || echo false)" "WebSocket service"
check_item "$([ -f "frontend/src/App.tsx" ] && echo true || echo false)" "Frontend App component"
check_item "$([ -f "frontend/src/types/index.ts" ] && echo true || echo false)" "TypeScript definitions"

echo ""
echo "📊 Summary"
echo "----------"
echo "Checks passed: $checks_passed/$total_checks"

if [ $checks_passed -eq $total_checks ]; then
    echo -e "${GREEN}🎉 All checks passed! Ready for deployment.${NC}"
    echo ""
    echo "🚀 Next Steps:"
    echo "1. Run: ./scripts/setup.sh"
    echo "2. Access: http://localhost:3000"
    echo "3. Login: admin@playpulse.dev / admin123"
    exit 0
else
    echo -e "${RED}❌ Some checks failed. Please review the missing components.${NC}"
    exit 1
fi