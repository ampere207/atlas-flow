#!/bin/bash
# Atlas Flow - Complete Startup & Demo Script

set -e

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

# Start all services
echo -e "${BLUE}[3/6]${NC} Starting infrastructure and orchestrator..."
docker-compose up -d postgres redis nats
echo "   Waiting for services to be healthy..."
sleep 5
docker-compose up -d workflow-service
echo "   Waiting for orchestrator to be ready..."
sleep 3
echo -e "${GREEN}✓ Infrastructure started (PostgreSQL, Redis, NATS, Orchestrator)${NC}"
echo ""

# Start demo workers
echo -e "${BLUE}[4/6]${NC} Starting demo workers..."
docker-compose up -d worker-1 worker-2 worker-3
echo "   Waiting for workers to register..."
sleep 5
echo -e "${GREEN}✓ All 3 demo workers started and registered${NC}"
echo -e "  • Worker 1: HTTP & Script tasks (capacity: 5)"
echo -e "  • Worker 2: Database & Echo tasks (capacity: 8)"
echo -e "  • Worker 3: All task types (capacity: 10)"
echo ""

# Check running containers
echo -e "${BLUE}[5/6]${NC} Verifying all services are running..."
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
echo -e "📍 ${BLUE}API Endpoints:${NC}"
echo -e "   • Orchestrator API: ${YELLOW}http://localhost:8002${NC}"
echo -e "   • Health Check: ${YELLOW}curl http://localhost:8002/health${NC}"
echo ""
echo -e "📊 ${BLUE}Monitoring:${NC}"
echo -e "   • View orchestrator logs: ${YELLOW}docker-compose logs -f workflow-service${NC}"
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
