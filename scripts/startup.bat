@echo off
REM Atlas Flow - Complete Startup & Demo Script (Windows)

setlocal enabledelayedexpansion

if not defined DB_USER set DB_USER=atlasflow
if not defined DB_NAME set DB_NAME=atlasflow

echo.
echo ╔════════════════════════════════════════════════════════════════╗
echo ║        Atlas Flow - Distributed Workflow Orchestration        ║
echo ║                    Complete System Demo                        ║
echo ╚════════════════════════════════════════════════════════════════╝
echo.

REM Check if Docker is running
echo [1/6] Checking Docker...
docker ps >nul 2>&1
if errorlevel 1 (
    echo X Docker is not running. Please start Docker Desktop.
    exit /b 1
)
echo ✓ Docker is running
echo.

REM Stop existing containers
echo [2/6] Cleaning up existing containers...
docker-compose down 2>nul
timeout /t 2 /nobreak >nul
echo ✓ Cleanup complete
echo.

REM Build images once here instead of rebuilding during demo runs.
echo [3/6] Building service images...
docker-compose build auth-service workflow-service frontend worker-1 worker-2 worker-3
echo ✓ Service images built
echo.

REM Start infrastructure and apply schema once here.
echo [4/6] Starting infrastructure and applying migrations...
docker-compose up -d postgres redis nats
echo    Waiting for services to be healthy...
timeout /t 5 /nobreak >nul
docker-compose exec -T postgres psql -U %DB_USER% -d %DB_NAME% -f /docker-entrypoint-initdb.d/001_init_schema.sql
docker-compose exec -T postgres psql -U %DB_USER% -d %DB_NAME% -f /docker-entrypoint-initdb.d/002_phase2_runtime.sql
echo ✓ Database migrations applied
echo.

echo [5/6] Starting application services...
docker-compose up -d auth-service workflow-service frontend
echo    Waiting for auth service, orchestrator and frontend to be ready...
timeout /t 5 /nobreak >nul
echo ✓ Infrastructure started (PostgreSQL, Redis, NATS, Auth, Orchestrator, Frontend)
echo.

REM Start demo workers
echo [6/6] Starting demo workers...
docker-compose up -d worker-1 worker-2 worker-3
echo    Waiting for workers to register...
timeout /t 5 /nobreak >nul
echo ✓ All 3 demo workers started and registered
echo   • Worker 1: HTTP ^& Script tasks (capacity: 5)
echo   • Worker 2: Database ^& Echo tasks (capacity: 8)
echo   • Worker 3: All task types (capacity: 10)
echo.

REM Check running containers
echo [6/6] Verifying all services are running...
echo ✓ Services running:
docker-compose ps
echo.

REM Display access information
echo [6/6] System Ready!
echo.
echo ═══════════════════════════════════════════════════════════════
echo Atlas Flow is now running!
echo.
echo 🌐 Web Interfaces:
echo    • Frontend Dashboard: http://localhost:3000
echo.
echo 📍 API Endpoints:
echo    • Orchestrator API: http://localhost:8002
echo    • Health Check: curl http://localhost:8002/health
echo.
echo 📊 Monitoring:
echo    • View orchestrator logs: docker-compose logs -f workflow-service
echo    • View worker-1 logs: docker-compose logs -f worker-1
echo    • View worker-2 logs: docker-compose logs -f worker-2
echo    • View worker-3 logs: docker-compose logs -f worker-3
echo.
echo 🚀 Quick Start - Create ^& Execute Workflow:
echo.
echo    Terminal 1: Monitor orchestrator
echo    docker-compose logs -f workflow-service
echo.
echo    Terminal 2: Run demo workflow
echo    bash scripts/demo-workflows.sh
echo.
echo ═══════════════════════════════════════════════════════════════
echo.
echo 📚 Documentation:
echo    • Complete Usage Guide: USAGE_GUIDE.md
echo    • Real Worker System: REAL_WORKER_SYSTEM.md
echo    • Phase 2 Audit: PHASE_2_AUDIT.md
echo.
echo ⏹️  To stop everything:
echo    docker-compose down
echo.

endlocal
