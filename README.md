# Atlas Flow - Distributed Workflow Orchestration System

> **Production-grade distributed workflow orchestration engine with real NATS-based worker coordination and automatic load balancing.**

## 🚀 Quick Start (One Command)

### macOS/Linux:
```bash
bash scripts/startup.sh
```

### Windows (PowerShell):
```powershell
bash scripts/startup.bat
```

This single command:
- ✅ Starts PostgreSQL, Redis, NATS
- ✅ Starts orchestrator service
- ✅ Starts 3 demo workers with different capabilities
- ✅ Outputs all connection details

**All services run in Docker containers — no local installations required.**

---

## 📋 What Gets Started

When you run the startup script, you get:

| Service | Port | Purpose |
|---------|------|---------|
| **PostgreSQL** | 5432 | Workflow & task persistence |
| **Redis** | 6379 | Caching & session management |
| **NATS** | 4222 | Real-time pub/sub message bus |
| **Orchestrator** | 8002 | Task distribution & coordination |
| **Worker 1** | - | HTTP & Script task handler (capacity: 5) |
| **Worker 2** | - | Database & Echo task handler (capacity: 8) |
| **Worker 3** | - | All task types (capacity: 10) |

---

## 🎯 How It Works

### The Real Worker System

Workers are **real processes** that maintain persistent connections to the orchestrator via NATS:

```
┌─────────────────────────────────────────────────────────────┐
│                     Workflow Created                        │
│                   POST /workflows/create                    │
└────────────────────────────────┬──────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────┐
│              Orchestrator Detects Running Workflow          │
│          (Polls database every 1 second for running tasks)  │
└────────────────────────────────┬──────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────┐
│          Identifies Tasks with Satisfied Dependencies       │
│                (Ready to execute right now)                 │
└────────────────────────────────┬──────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────┐
│      Selects Best Worker (least loaded with capability)     │
│           (Queries WorkerConnectionManager)                 │
└────────────────────────────────┬──────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────┐
│          Sends Task via NATS to Worker                      │
│      (Publishes to workers.{worker-id}.tasks channel)      │
└────────────────────────────────┬──────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────┐
│        Worker Receives Task (listening on NATS)             │
│           Executes based on task type (HTTP, Script, DB)    │
└────────────────────────────────┬──────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────┐
│    Worker Sends Result back via NATS                        │
│  (Publishes to tasks.{task-id}.result channel)            │
└────────────────────────────────┬──────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────┐
│  Orchestrator Receives Result & Updates Workflow            │
│   (Database updated, next ready tasks identified)           │
└────────────────────────────────┬──────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────┐
│   Workflow Continues Until Completion                       │
│        (All tasks executed, status = completed)             │
└─────────────────────────────────────────────────────────────┘
```

### Worker Heartbeat System

Workers send heartbeats every 10 seconds:

```json
{
  "worker_id": "worker-1",
  "status": "connected",
  "capabilities": ["http_request", "script", "echo"],
  "capacity": 5,
  "running_tasks": 2,
  "completed_tasks": 45,
  "failed_tasks": 2
}
```

The orchestrator:
- ✅ Detects worker liveness in real-time
- ✅ Tracks worker capacity and load
- ✅ Knows exactly which tasks each worker can execute
- ✅ Auto-reassigns tasks from dead workers after 30 seconds

---

## 🔨 Create Your First Workflow

### Via cURL (API):

```bash
# 1. Create workflow
curl -X POST http://localhost:8002/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My First Workflow",
    "definition": {
      "tasks": [
        {
          "id": "greeting",
          "type": "echo",
          "payload": {"message": "Hello from Atlas Flow!"}
        },
        {
          "id": "fetch",
          "type": "http_request",
          "payload": {
            "url": "https://jsonplaceholder.typicode.com/posts/1",
            "method": "GET"
          },
          "depends_on": ["greeting"]
        }
      ]
    }
  }'
```

Response:
```json
{
  "id": "wf-abc123",
  "name": "My First Workflow",
  "status": "created"
}
```

### 2. Execute the workflow:

```bash
curl -X POST http://localhost:8002/workflows/wf-abc123/execute
```

### 3. Monitor execution:

```bash
# Watch orchestrator logs
docker-compose logs -f workflow-service

# Watch worker execution
docker-compose logs -f worker-1 worker-2 worker-3

# Or poll API
curl http://localhost:8002/workflows/wf-abc123
```

---

## 📊 Demo Workflows

Run comprehensive demos showing the system in action:

```bash
bash scripts/demo-workflows.sh
```

This runs:
1. **Echo Pipeline** - Simple sequential task execution
2. **Multi-Task Workflow** - Mixed HTTP, Script, Database tasks
3. **Parallel Execution** - Multiple tasks running simultaneously
4. **Load Balancing** - 3 concurrent workflows distributed across workers

---

## 🏗️ Architecture Overview

### Microservice-Based Design

Each component is isolated and independently deployable:

```
┌─────────────────────────────────────────────┐
│          Frontend (Next.js)                 │
│      (Workflow UI, Monitoring Dashboard)    │
└─────────────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────┐
│      Orchestrator Service (API on :8002)    │
│   - Creates workflows                        │
│   - Parses DAG                               │
│   - Sends tasks to workers via NATS         │
│   - Handles retries & failures              │
└─────────────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────┐
│      NATS Message Bus (:4222)               │
│   - workers.{id}.tasks (task assignment)    │
│   - workers.{id}.heartbeat (worker liveness)│
│   - tasks.{id}.result (task completion)     │
└─────────────────────────────────────────────┘
                        │
     ┌──────────────────┼──────────────────┐
     ▼                  ▼                  ▼
┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│  Worker 1   │  │  Worker 2   │  │  Worker 3   │
│ HTTP,Script │  │ Database    │  │    All      │
│ Capacity: 5 │  │ Capacity: 8 │  │ Capacity:10 │
└─────────────┘  └─────────────┘  └─────────────┘
```

### Technology Stack

- **Language**: Go 1.25
- **API Framework**: Gin (HTTP REST)
- **Message Bus**: NATS 2.10 (pub/sub)
- **Database**: PostgreSQL 16
- **Cache**: Redis 7
- **Container**: Docker + Docker Compose
- **Frontend**: Next.js 15, React 18, TailwindCSS

---

## 🔄 Task Types

### 1. Echo (Testing)
```json
{
  "type": "echo",
  "payload": {
    "message": "Hello World"
  }
}
```

### 2. HTTP Request
```json
{
  "type": "http_request",
  "payload": {
    "url": "https://api.example.com/data",
    "method": "GET",
    "headers": {"Authorization": "Bearer token"}
  }
}
```

### 3. Script Execution
```json
{
  "type": "script",
  "payload": {
    "script": "python transform.py",
    "timeout": 300
  }
}
```

### 4. Database Query
```json
{
  "type": "db_query",
  "payload": {
    "query": "SELECT * FROM users WHERE id = $1",
    "params": [123]
  }
}
```

---

## 📈 Scaling & Load Balancing

### Add More Workers

Edit `docker-compose.yml`:

```yaml
# Add to services section
worker-4:
  build:
    context: backend
    dockerfile: Dockerfile.worker
  environment:
    NATS_URL: nats://nats:4222
  command:
    - --nats=nats://nats:4222
    - --id=worker-4
    - --capabilities=http_request,script,db_query,echo
    - --capacity=15
  depends_on:
    - nats
    - workflow-service
  networks:
    - atlasflow
```

Then restart:
```bash
docker-compose up -d worker-4
```

New worker automatically registers and receives tasks!

### How Load Balancing Works

Orchestrator uses `WorkerConnectionManager` to find the best worker:

```go
// Selects least-loaded worker with required capability
worker := workerMgr.FindWorkerForTask(taskType)
// If worker-1 has 2/5 tasks and worker-2 has 1/8 tasks
// → Chooses worker-2 (more available capacity)
```

---

## 🔒 Failure Handling

### Automatic Retry
```json
{
  "id": "task-1",
  "type": "http_request",
  "payload": {...},
  "max_retries": 3
}
```

Retryable errors (network timeout, 503):
- Retry with exponential backoff
- If max_retries exceeded → Fail permanently

Non-retryable errors (400, 404):
- Fail immediately, no retry

### Worker Failure Recovery

**Scenario**: Worker crashes while executing task

```
1. Task assigned to worker-1 (state: assigned)
2. Worker-1 crashes
3. Orchestrator detects no heartbeat for 30+ seconds
4. Marks worker as "dead"
5. Reassigns task: state = pending
6. Next cycle: Sends to healthy worker (worker-2, worker-3)
7. Task continues from beginning
```

**Result**: Zero task loss, automatic recovery

---

## 📚 API Reference

### Workflows

```
POST   /workflows                Create workflow
GET    /workflows                List workflows
GET    /workflows/:id            Get workflow details
POST   /workflows/:id/execute    Execute workflow
GET    /workflows/:id/status     Get execution status
```

### Tasks

```
GET    /workflows/:id/tasks      Get all tasks
GET    /workflows/:id/tasks/:taskId  Get task details
```

### System

```
GET    /health                   Health check
GET    /workers                  List connected workers
```

---

## 🛠️ Development

### Build Locally

```bash
cd backend
go build ./...
```

### Run Without Docker

```bash
# Terminal 1: Start NATS
docker run -p 4222:4222 nats:2.10-alpine

# Terminal 2: Start orchestrator
go run ./workflow-service/cmd/main.go

# Terminal 3: Start worker
go run ./cmd/worker-agent/main.go --id=worker-1

# Terminal 4: Create workflows
curl -X POST http://localhost:8002/workflows ...
```

---

## 📖 Full Documentation

- **[USAGE_GUIDE.md](./USAGE_GUIDE.md)** - Comprehensive usage examples and API docs
- **[REAL_WORKER_SYSTEM.md](./REAL_WORKER_SYSTEM.md)** - Deep dive into worker architecture
- **[PHASE_2_AUDIT.md](./PHASE_2_AUDIT.md)** - Complete feature audit

---

## 🎓 Learning Path

1. **Start**: Run `bash scripts/startup.sh`
2. **Observe**: Watch demo workflows with `bash scripts/demo-workflows.sh`
3. **Understand**: Read [REAL_WORKER_SYSTEM.md](./REAL_WORKER_SYSTEM.md)
4. **Create**: Build your own workflows using [USAGE_GUIDE.md](./USAGE_GUIDE.md)
5. **Scale**: Add more workers and tasks

---

## 🐛 Troubleshooting

### Workers not connecting?
```bash
# Check NATS is running
docker-compose ps nats

# View worker logs
docker-compose logs worker-1 worker-2 worker-3

# Restart workers
docker-compose restart worker-1 worker-2 worker-3
```

### Tasks stuck?
```bash
# View orchestrator logs
docker-compose logs -f workflow-service

# Check database
docker-compose exec postgres psql -U atlasflow -d atlasflow -c "SELECT * FROM tasks WHERE state='assigned';"

# Reset (development only)
docker-compose down -v  # Removes all data
docker-compose up -d    # Fresh start
```

### API not responding?
```bash
# Check orchestrator is running
docker-compose ps workflow-service

# Test health
curl http://localhost:8002/health

# View logs
docker-compose logs workflow-service
```

---

## 🌟 Key Features

✅ **Real Worker Coordination** - NATS pub/sub for true distributed execution
✅ **Automatic Load Balancing** - Smart worker selection based on capacity
✅ **Failure Recovery** - Automatic task reassignment on worker failure
✅ **Retry Logic** - Exponential backoff for transient failures
✅ **DAG Execution** - Complex workflows with dependencies
✅ **Parallel Execution** - Multiple tasks run simultaneously
✅ **Scalable** - Add workers on demand, no coordinator changes
✅ **Observable** - Heartbeats, logs, real-time task tracking
✅ **Production Ready** - Error handling, timeouts, health checks

---

## 📞 Support

For issues, questions, or contributions:
1. Check [USAGE_GUIDE.md](./USAGE_GUIDE.md) for common scenarios
2. Review logs: `docker-compose logs -f`
3. Create an issue with logs and reproduction steps

---

**Built with Go, NATS, and PostgreSQL — Designed for Production Microservice Architectures**


---

## The Problem

In modern microservice architectures, coordinating complex business workflows across multiple services is notoriously difficult:

- **Transient failures** - Network timeouts, service crashes, and partial failures require sophisticated retry logic
- **State management** - Tracking workflow progress across service boundaries is error-prone and fragile
- **Distributed consistency** - Ensuring tasks execute exactly once, in the correct order, is challenging
- **Failure recovery** - When workers crash or become unavailable, their tasks risk permanent loss
- **Observability** - Understanding what's happening in long-running workflows is nearly impossible without proper infrastructure
- **Scalability** - Coordinating thousands of concurrent workflows requires careful thought around resource allocation

Most teams resort to:
- **Manual orchestration** via temporary databases and polling loops
- **Message queues** that lack workflow semantics
- **Ad-hoc solutions** built into business logic
- **Expensive commercial platforms** that lock you into their ecosystem

AtlasFlow changes this paradigm.

---

## The Solution

AtlasFlow is built on the principles of **durable execution systems** like Temporal and Cadence, but designed from the ground up for modern distributed infrastructure:

### Core Capabilities

**Durable Workflow Execution**
- Workflows persist their state to a database after every step
- If a system crashes mid-execution, it recovers from the exact point of failure
- No data loss, no duplicate execution, no manual intervention needed
- Your workflow definition is the source of truth

**Distributed Worker Coordination**
- Workers poll for tasks they can execute
- Tasks are claimed atomically using distributed locks
- No task executes twice, even with multiple workers
- Workers register, heartbeat, and report results automatically

**Smart Failure Recovery**
- Detects when workers crash or become unresponsive
- Automatically reassigns orphaned tasks to healthy workers
- Implements exponential backoff for transient failures
- Distinguishes between retryable and permanent failures

**Workflow DAG Orchestration**
- Define complex workflows as directed acyclic graphs (DAGs)
- Tasks execute only when their dependencies are satisfied
- Support sequential chains, parallel branches, and complex dependency networks
- DAG execution is automatically coordinated and optimized

**Real-Time Event Stream**
- Every state change in your workflow emits an event
- Subscribe to events for live monitoring and integration
- Build event-driven systems on top of AtlasFlow
- Complete execution history is preserved for auditing and debugging

**Multi-Tenant Architecture**
- Each user has complete isolation of their workflows and workers
- No user can see or access another user's data
- Built-in authorization and ownership validation
- Suitable for multi-customer deployments

---

## How It Works

### The Execution Model

```
User creates a Workflow
        ↓
     [DAG Defined]
        ↓
    Execute Workflow
        ↓
   [Tasks Scheduled]
        ↓
  Workers Poll Queue
        ↓
 [Task Claimed]
        ↓
  Worker Executes
        ↓
[Result Reported]
        ↓
  Orchestrator Updates State
        ↓
  Check Dependencies
        ↓
 [Next Tasks Ready]
        ↓
  Cycle Repeats...
        ↓
   Workflow Completes
        ↓
  [Event Stream Published]
```

### Failure Resilience

When a worker crashes:
1. Orchestrator detects missing heartbeat
2. Identifies tasks assigned to dead worker
3. Reassigns tasks to healthy workers
4. Workflow continues without data loss
5. No manual intervention required

When a task fails:
1. Orchestrator evaluates retry policy
2. Calculates exponential backoff
3. Reschedules task for later retry
4. After max attempts, marks as failed
5. Workflow continues or stops based on policy

### Real-Time Visibility

Every step of execution broadcasts events:
- Workflow started
- Task assigned to worker
- Task execution began
- Task completed/failed
- Workflow finished
- And many more...

Subscribe to these events to build:
- Real-time dashboards
- Monitoring and alerting systems
- Integration with external systems
- Custom business logic

---

## Architecture

### System Components

The system is organized as independent services that communicate via events:

**Orchestration Engine**
- Parses workflow definitions
- Determines which tasks are ready
- Coordinates task scheduling
- Handles state transitions
- Manages retries and recovery

**Worker Runtime**
- Polls for available work
- Claims task ownership atomically
- Executes tasks with timeout protection
- Reports results back to system
- Sends continuous heartbeats

**Event Bus**
- Publishes all state changes as events
- Enables real-time observability
- Decouples services via async communication
- Preserves complete execution history

**User Interface**
- Create and manage workflows
- Register and monitor workers
- View real-time execution progress
- Track execution history
- Monitor cluster health

### Data Persistence

All execution state is persisted to maintain durability:
- **Workflow State** - Current execution status and metadata
- **Task State** - Individual task progress and assignments
- **Execution History** - Complete audit trail of all transitions
- **Worker State** - Heartbeats and availability
- **Event Log** - All orchestration events for replay and debugging

This ensures that if any component fails, recovery is possible from persistent state.

---

## Key Features

### Orchestration
- ✅ DAG-based workflow execution
- ✅ Dependency resolution
- ✅ Automatic task scheduling
- ✅ Parallel execution support
- ✅ Sequential and complex workflows

### Durability & Reliability
- ✅ Persistent execution state
- ✅ Durable task queues
- ✅ Exactly-once execution semantics
- ✅ Automatic failure recovery
- ✅ Worker crash resilience

### Retry Logic
- ✅ Exponential backoff strategies
- ✅ Configurable retry policies
- ✅ Max attempt limits
- ✅ Prevents retry storms
- ✅ Distinguishes transient vs permanent failures

### Observability
- ✅ Real-time event streaming
- ✅ Complete execution history
- ✅ Live workflow monitoring
- ✅ Worker health tracking
- ✅ Detailed execution timelines

### Multi-Tenancy
- ✅ User-scoped data isolation
- ✅ Independent resource quotas
- ✅ Ownership validation on all operations
- ✅ Secure inter-user boundaries

### Scalability
- ✅ Supports thousands of concurrent workflows
- ✅ Distributed worker pools
- ✅ Horizontal scaling
- ✅ Load balancing across workers
- ✅ Event-driven coordination

---

## Use Cases

AtlasFlow is ideal for:

- **Order Processing Pipelines** - Complex multi-step order fulfillment with retries and failure handling
- **Data Processing Workflows** - ETL jobs, data validation, transformation, and aggregation
- **Approval Workflows** - Multi-stage approval processes with human-in-the-loop decisions
- **Microservice Sagas** - Long-running distributed transactions across multiple services
- **Report Generation** - Scheduled or on-demand reports with complex dependencies
- **Batch Processing** - Large-scale batch operations with automatic failure recovery
- **Integration Workflows** - Coordinating data flow between multiple external systems

---

## Core Concepts

### Workflow
A high-level orchestration plan defining what work needs to happen and in what order. Workflows are defined once and can be executed multiple times.

### Task
An atomic unit of work within a workflow. Tasks have clear inputs, outputs, and can succeed or fail independently.

### Worker
A distributed execution node that claims and executes tasks. Workers continuously heartbeat to prove they're alive and healthy.

### DAG (Directed Acyclic Graph)
The dependency structure describing how tasks relate to each other. Tasks only execute when all their dependencies are complete.

### Event
A notification that something happened during workflow execution. Events are emitted for every significant state change.

### Orchestrator
The central intelligence that manages workflow execution, task scheduling, dependency resolution, and failure recovery.

---

## Philosophy

AtlasFlow is built on these principles:

**Durability First** - Every important action is persisted. Recovery is always possible.

**Simplicity** - Clean abstractions hide complexity. Define workflows intuitively.

**Observability** - Complete visibility into what your system is doing. Real-time events, execution history, live dashboards.

**Reliability** - Failures are expected and handled gracefully. Your workflows should survive anything.

**Scalability** - Built for growth. Support thousands of workflows and hundreds of workers without breaking a sweat.

**Multi-Tenant Safety** - One user's workflows cannot affect another's. Strict isolation at every level.

---

## For More Information

For detailed information about implementation, deployment, and advanced usage, see the project's internal documentation.
Authorization: Bearer {access_token}

Response:
{
  "success": true,
  "data": [ ... ]
}
```

#### Get Workflow
```
GET /workflows/{id}
Authorization: Bearer {access_token}

Response:
{
  "success": true,
  "data": { ... }
}
```

#### Update Workflow Status
```
PUT /workflows/{id}/status
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "status": "running"
}

Response:
{
  "success": true,
  "message": "workflow updated successfully"
}
```

### Worker Endpoints

#### Register Worker
```
POST /workers
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "name": "Worker-1"
}

Response:
{
  "success": true,
  "data": {
    "id": "uuid",
    "user_id": "uuid",
    "name": "Worker-1",
    "status": "idle",
    "last_heartbeat": "2024-01-01T00:00:00Z",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

#### List Workers
```
GET /workers?limit=10&offset=0
Authorization: Bearer {access_token}

Response:
{
  "success": true,
  "data": [ ... ]
}
```

#### Record Heartbeat
```
POST /workers/{id}/heartbeat
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "status": "active"
}

Response:
{
  "success": true,
  "message": "heartbeat recorded"
}
```

---

## Database Schema

### Users Table
```sql
CREATE TABLE users (
  id UUID PRIMARY KEY,
  email VARCHAR(255) UNIQUE NOT NULL,
  full_name VARCHAR(255) NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  created_at TIMESTAMP,
  updated_at TIMESTAMP
);
```

### Workflows Table
```sql
CREATE TABLE workflows (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id),
  name VARCHAR(255) NOT NULL,
  status VARCHAR(50) NOT NULL,
  metadata JSONB,
  created_at TIMESTAMP,
  updated_at TIMESTAMP
);
```

### Workers Table
```sql
CREATE TABLE workers (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id),
  name VARCHAR(255) NOT NULL,
  status VARCHAR(50) NOT NULL,
  last_heartbeat TIMESTAMP,
  created_at TIMESTAMP,
  updated_at TIMESTAMP
);
```

### Workflow Events Table
```sql
CREATE TABLE workflow_events (
  id UUID PRIMARY KEY,
  workflow_id UUID NOT NULL REFERENCES workflows(id),
  event_type VARCHAR(100) NOT NULL,
  payload JSONB,
  created_at TIMESTAMP
);
```

### Refresh Tokens Table
```sql
CREATE TABLE refresh_tokens (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id),
  token TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMP,
  created_at TIMESTAMP
);
```

---

## Development

### Building Backend Services

```bash
# Build all services
make build

# Run specific service
cd backend/auth-service && go run ./cmd/main.go
```

### Running Tests

```bash
make test
```

### Code Formatting

```bash
make fmt
```

### Database Management

#### Access PostgreSQL
```bash
make shell-postgres
```

#### Access Redis
```bash
make redis-cli
```

#### View Logs
```bash
make docker-logs
```

---

## Security Considerations

1. **Authentication**
   - JWT-based authentication with access + refresh tokens
   - Bcrypt password hashing
   - Token expiration and refresh mechanism

2. **Data Isolation**
   - All queries scoped by user_id
   - No global data exposure
   - Multi-tenant architecture

3. **API Security**
   - CORS configuration
   - Request validation
   - Structured error handling

4. **Future Enhancements**
   - Rate limiting
   - API key authentication
   - Audit logging
   - Encryption at rest

---

## Environment Configuration

Default `.env` configuration for development. See `.env.example` for all available options.

```
DB_HOST=localhost
DB_PORT=5432
DB_USER=atlasflow
DB_PASSWORD=atlasflow_dev
DB_NAME=atlasflow

REDIS_HOST=localhost
REDIS_PORT=6379

NATS_URL=nats://localhost:4222

JWT_SECRET=your-super-secret-key-change-in-production-12345

NEXT_PUBLIC_API_URL=http://localhost:8000
```

---

## Next Steps (Phase 2)

- Workflow replay and recovery
- Retry logic and failure handling
- Rollback mechanisms
- Deterministic execution
- Advanced orchestration patterns
- Distributed transaction support
- Performance optimization

---

## License

Proprietary - AtlasFlow Infrastructure

---

## Support

For issues, questions, or contributions, please refer to the internal documentation.
