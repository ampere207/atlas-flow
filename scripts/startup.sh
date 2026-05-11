#!/bin/bash
# Atlas Flow - Complete Startup & Demo Script

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [ -f "$SCRIPT_DIR/../.env.local" ]; then
    set -a
    . <(tr -d '\r' < "$SCRIPT_DIR/../.env.local")
    set +a
elif [ -f "$SCRIPT_DIR/../.env" ]; then
    set -a
    . <(tr -d '\r' < "$SCRIPT_DIR/../.env")
    set +a
fi

DB_USER="${DB_USER:-atlasflow}"
DB_NAME="${DB_NAME:-atlasflow}"

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║        Atlas Flow - Distributed Workflow Orchestration        ║"
echo "║                    Complete System Demo                        ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if Docker is running
echo -e "${BLUE}[1/6]${NC} Checking Docker..."
if ! docker ps > /dev/null 2>&1; then
    echo -e "${RED}✗ Docker is not running. Please start Docker Desktop.${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Docker is running${NC}"
echo ""

# Stop existing containers
echo -e "${BLUE}[2/6]${NC} Cleaning up existing containers..."
docker-compose down 2>/dev/null || true
sleep 2
echo -e "${GREEN}✓ Cleanup complete${NC}"
echo ""

# Build images once here instead of rebuilding during demo runs.
echo -e "${BLUE}[3/6]${NC} Building service images (with BuildKit caching)..."
DOCKER_BUILDKIT=1 docker-compose build auth-service workflow-service frontend worker-1 worker-2 worker-3
echo -e "${GREEN}✓ Service images built${NC}"
echo ""

# Start infrastructure and apply schema once here.
echo -e "${BLUE}[4/6]${NC} Starting infrastructure and applying migrations..."
docker-compose up -d postgres redis nats
echo "   Waiting for services to be healthy..."
sleep 5
docker-compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" -f /docker-entrypoint-initdb.d/001_init_schema.sql
docker-compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" -f /docker-entrypoint-initdb.d/002_phase2_runtime.sql
echo -e "${GREEN}✓ Database migrations applied${NC}"
echo ""

echo -e "${BLUE}[5/6]${NC} Starting application services..."
docker-compose up -d auth-service workflow-service frontend
echo "   Waiting for orchestrator, auth service, and frontend to be ready..."
sleep 5
echo -e "${GREEN}✓ Infrastructure started (PostgreSQL, Redis, NATS, Auth, Orchestrator, Frontend)${NC}"
echo ""

echo -e "${BLUE}[5/6]${NC} Resolving authenticated user identity for worker registration..."
AUTH_URL="http://localhost:8001/auth"
AUTH_EMAIL="${ATLASFLOW_AUTH_EMAIL:-aashrith@gmail.com}"
AUTH_PASSWORD="${ATLASFLOW_AUTH_PASSWORD:-aashrith}"

LOGIN_RESPONSE=$(curl -s -X POST "$AUTH_URL/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\": \"$AUTH_EMAIL\", \"password\": \"$AUTH_PASSWORD\"}")

if ! echo "$LOGIN_RESPONSE" | grep -q '"access_token"'; then
    curl -s -X POST "$AUTH_URL/signup" \
        -H "Content-Type: application/json" \
        -d "{\"email\": \"$AUTH_EMAIL\", \"full_name\": \"Atlas User\", \"password\": \"$AUTH_PASSWORD\"}" >/dev/null 2>&1 || true
    LOGIN_RESPONSE=$(curl -s -X POST "$AUTH_URL/login" \
        -H "Content-Type: application/json" \
        -d "{\"email\": \"$AUTH_EMAIL\", \"password\": \"$AUTH_PASSWORD\"}")
fi

ATLASFLOW_USER_ID=$(echo "$LOGIN_RESPONSE" | grep -o '"user":{"id":"[^"]*' | cut -d'"' -f6)
if [ -z "$ATLASFLOW_USER_ID" ]; then
    echo -e "${RED}✗ Failed to resolve authenticated user ID from auth service${NC}"
    echo "   Response: $LOGIN_RESPONSE"
    exit 1
fi

export ATLASFLOW_USER_ID
echo -e "${GREEN}✓ Authenticated user resolved: $AUTH_EMAIL -> $ATLASFLOW_USER_ID${NC}"
echo ""

WORKER_ENV_FILE="$SCRIPT_DIR/../.env.worker-runtime"
cat > "$WORKER_ENV_FILE" <<EOF
ATLASFLOW_USER_ID=$ATLASFLOW_USER_ID
EOF
echo -e "${GREEN}✓ Worker runtime env written to .env.worker-runtime${NC}"
echo ""

# Start demo workers
echo -e "${BLUE}[6/6]${NC} Starting demo workers..."
docker-compose up -d worker-1 worker-2 worker-3
echo "   Waiting for workers to register..."
sleep 5
echo -e "${GREEN}✓ All 3 demo workers started and registered${NC}"
echo -e "  • Worker 1: HTTP & Script tasks (capacity: 5)"
echo -e "  • Worker 2: Database & Echo tasks (capacity: 8)"
echo -e "  • Worker 3: All task types (capacity: 10)"
echo ""

# Check running containers
echo -e "${BLUE}[6/6]${NC} Verifying all services are running..."
RUNNING=$(docker-compose ps --services --filter "status=running" | wc -l)
echo -e "${GREEN}✓ Services running:${NC}"
docker-compose ps

echo ""

# Display access information
echo -e "${BLUE}[6/6]${NC} System Ready!"
echo ""
echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}Atlas Flow is now running!${NC}"
echo ""
echo -e "${BLUE}🌐 Web Interfaces:${NC}"
echo -e "   • Frontend Dashboard: ${YELLOW}http://localhost:3000${NC}"
echo ""
echo -e "${BLUE}📍 API Endpoints:${NC}"
echo -e "   • Orchestrator API: ${YELLOW}http://localhost:8002${NC}"
echo -e "   • Health Check: ${YELLOW}curl http://localhost:8002/health${NC}"
echo ""
echo -e "📊 ${BLUE}Monitoring:${NC}"
echo -e "   • View orchestrator logs: ${YELLOW}docker-compose logs -f workflow-service${NC}"
echo -e "   • View frontend logs: ${YELLOW}docker-compose logs -f frontend${NC}"
echo -e "   • View worker-1 logs: ${YELLOW}docker-compose logs -f worker-1${NC}"
echo -e "   • View worker-2 logs: ${YELLOW}docker-compose logs -f worker-2${NC}"
echo -e "   • View worker-3 logs: ${YELLOW}docker-compose logs -f worker-3${NC}"
echo ""
echo -e "🚀 ${BLUE}Quick Start - Create & Execute Workflow:${NC}"
echo ""
echo -e "   ${YELLOW}# Terminal 1: Monitor orchestrator${NC}"
echo -e "   docker-compose logs -f workflow-service"
echo ""
echo -e "   ${YELLOW}# Terminal 2: Run demo workflow${NC}"
echo -e "   bash scripts/demo-workflows.sh"
echo ""
echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "📚 ${BLUE}Documentation:${NC}"
echo -e "   • Complete Usage Guide: ${YELLOW}USAGE_GUIDE.md${NC}"
echo -e "   • Real Worker System: ${YELLOW}REAL_WORKER_SYSTEM.md${NC}"
echo -e "   • Phase 2 Audit: ${YELLOW}PHASE_2_AUDIT.md${NC}"
echo ""
echo -e "⏹️  ${BLUE}To stop everything:${NC}"
echo -e "   docker-compose down"
echo ""
