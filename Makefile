.PHONY: help build up down logs clean test fmt lint docker-build docker-up docker-down

help:
	@echo "AtlasFlow - Distributed Workflow Orchestration Engine"
	@echo ""
	@echo "Available commands:"
	@echo "  make build              Build all backend services"
	@echo "  make docker-build       Build Docker images"
	@echo "  make docker-up          Start Docker Compose stack"
	@echo "  make docker-down        Stop Docker Compose stack"
	@echo "  make docker-logs        View Docker logs"
	@echo "  make db-migrate         Run database migrations"
	@echo "  make test               Run tests"
	@echo "  make fmt                Format code"
	@echo "  make lint               Lint code"
	@echo "  make clean              Clean build artifacts"

# Build commands
build:
	@echo "Building backend services..."
	@cd backend/auth-service && go build -o ../../bin/auth-service ./cmd/main.go
	@cd backend/workflow-service && go build -o ../../bin/workflow-service ./cmd/main.go
	@cd backend/worker-service && go build -o ../../bin/worker-service ./cmd/main.go
	@cd backend/event-service && go build -o ../../bin/event-service ./cmd/main.go
	@cd backend/gateway-service && go build -o ../../bin/gateway-service ./cmd/main.go

# Docker commands
docker-build:
	@echo "Building Docker images..."
	docker-compose build

docker-up:
	@echo "Starting Docker Compose stack..."
	docker-compose up -d
	@echo "Services started!"
	@echo "API Gateway: http://localhost:8000"
	@echo "Frontend: http://localhost:3000"

docker-down:
	@echo "Stopping Docker Compose stack..."
	docker-compose down

docker-logs:
	docker-compose logs -f

docker-clean:
	docker-compose down -v
	@echo "Containers and volumes removed"

# Database commands
db-migrate:
	@echo "Running database migrations..."
	PGPASSWORD=atlasflow_dev psql -h localhost -U atlasflow -d atlasflow -f infra/migrations/001_init_schema.sql
	PGPASSWORD=atlasflow_dev psql -h localhost -U atlasflow -d atlasflow -f infra/migrations/002_phase2_runtime.sql
	@echo "Migrations completed!"

# Development commands
test:
	@echo "Running tests..."
	@cd backend/auth-service && go test ./...
	@cd backend/workflow-service && go test ./...
	@cd backend/worker-service && go test ./...
	@cd backend/event-service && go test ./...
	@cd backend/gateway-service && go test ./...

fmt:
	@echo "Formatting code..."
	@cd backend && go fmt ./...

lint:
	@echo "Linting code..."
	@cd backend && golangci-lint run ./...

clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	docker-compose down -v
	@echo "Cleanup complete!"

# Utility commands
status:
	@echo "Service Status:"
	docker-compose ps

shell-postgres:
	PGPASSWORD=atlasflow_dev psql -h localhost -U atlasflow -d atlasflow

shell-redis:
	redis-cli -h localhost -p 6379

nats-pub:
	nats pub -s nats://localhost:4222

# Quick start
init: docker-build docker-up db-migrate
	@echo ""
	@echo "✓ AtlasFlow initialized!"
	@echo ""
	@echo "Services running:"
	@echo "  - API Gateway: http://localhost:8000"
	@echo "  - Frontend: http://localhost:3000"
	@echo "  - PostgreSQL: localhost:5432"
	@echo "  - Redis: localhost:6379"
	@echo "  - NATS: nats://localhost:4222"
