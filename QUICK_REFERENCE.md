# Atlas Flow - Quick Reference Card

## 🚀 Start Everything (One Command)

```bash
bash scripts/startup.sh
```

That's it! All services will start in Docker containers.

---

## 📊 What You Get

| Service | Status | Port | Purpose |
|---------|--------|------|---------|
| PostgreSQL | Running | 5432 | Database |
| Redis | Running | 6379 | Cache |
| NATS | Running | 4222 | Message Bus |
| Orchestrator | Running | 8002 | API + Coordination |
| Worker 1 | Running | - | HTTP & Script |
| Worker 2 | Running | - | Database & Echo |
| Worker 3 | Running | - | All Tasks |

---

## 🔨 Create & Execute First Workflow

### Create:
```bash
curl -X POST http://localhost:8002/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Workflow",
    "definition": {
      "tasks": [
        {
          "id": "echo-1",
          "type": "echo",
          "payload": {"message": "Hello!"}
        }
      ]
    }
  }'
```

### Execute:
```bash
# Replace WORKFLOW_ID with response from above
curl -X POST http://localhost:8002/workflows/WORKFLOW_ID/execute
```

### Monitor:
```bash
# View orchestrator
docker-compose logs -f workflow-service

# View workers
docker-compose logs -f worker-1 worker-2 worker-3
```

---

## 📊 Run Demos

```bash
bash scripts/demo-workflows.sh
```

Shows:
1. Sequential task execution
2. Multiple task types
3. Parallel execution
4. Load balancing

---

## 🔄 Task Types

### Echo
```json
{"type": "echo", "payload": {"message": "test"}}
```

### HTTP Request
```json
{
  "type": "http_request",
  "payload": {
    "url": "https://api.example.com/data",
    "method": "GET"
  }
}
```

### Script
```json
{
  "type": "script",
  "payload": {
    "script": "echo 'running script'",
    "timeout": 30
  }
}
```

### Database
```json
{
  "type": "db_query",
  "payload": {
    "query": "SELECT * FROM table"
  }
}
```

---

## 📡 API Quick Reference

```bash
# Create workflow
POST /workflows

# List workflows
GET /workflows

# Get workflow details
GET /workflows/{id}

# Execute workflow
POST /workflows/{id}/execute

# Get workflow status
GET /workflows/{id}

# Get tasks
GET /workflows/{id}/tasks

# System health
GET /health

# List workers
GET /workers
```

---

## 🔍 Monitor & Debug

```bash
# Check all services
docker-compose ps

# View all logs
docker-compose logs -f

# View specific service
docker-compose logs -f workflow-service

# View specific worker
docker-compose logs -f worker-1

# Access database
docker-compose exec postgres psql -U atlasflow -d atlasflow

# Access NATS
docker-compose logs -f nats
```

---

## ➕ Add More Workers

1. Edit `docker-compose.yml`
2. Add new worker service:
```yaml
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

3. Start it:
```bash
docker-compose up -d worker-4
```

---

## 🛑 Stop Everything

```bash
docker-compose down
```

### Stop and Reset (Delete All Data)
```bash
docker-compose down -v
```

---

## 💡 Common Tasks

### Check if everything is running
```bash
curl http://localhost:8002/health
```

### List all workflows
```bash
curl http://localhost:8002/workflows | jq .
```

### List connected workers
```bash
curl http://localhost:8002/workers | jq .
```

### View workflow execution
```bash
curl http://localhost:8002/workflows/WORKFLOW_ID | jq .
```

### Create workflow with dependencies
```json
{
  "name": "Complex Workflow",
  "definition": {
    "tasks": [
      {
        "id": "task-1",
        "type": "echo",
        "payload": {"message": "start"}
      },
      {
        "id": "task-2",
        "type": "http_request",
        "payload": {"url": "https://api.example.com/data"},
        "depends_on": ["task-1"]
      },
      {
        "id": "task-3",
        "type": "script",
        "payload": {"script": "echo done"},
        "depends_on": ["task-2"]
      }
    ]
  }
}
```

---

## 📚 Full Documentation

- **README.md** - Project overview & architecture
- **USAGE_GUIDE.md** - Complete usage guide
- **REAL_WORKER_SYSTEM.md** - Worker architecture
- **PHASE_2_AUDIT.md** - Feature audit
- **IMPLEMENTATION_SUMMARY.md** - Session summary

---

## 🐛 Troubleshooting

### Workers not connecting?
```bash
docker-compose logs worker-1 worker-2 worker-3
docker-compose restart worker-1 worker-2 worker-3
```

### API not responding?
```bash
docker-compose logs workflow-service
curl http://localhost:8002/health
```

### Tasks not executing?
```bash
docker-compose logs -f workflow-service
docker-compose logs -f worker-1 worker-2 worker-3
```

### Reset everything?
```bash
docker-compose down -v
bash scripts/startup.sh
```

---

## 🔗 Quick Links

- **Orchestrator API**: http://localhost:8002
- **Health Check**: http://localhost:8002/health
- **NATS Management**: http://localhost:8222
- **PostgreSQL**: localhost:5432
- **Redis**: localhost:6379

---

## 📋 Workflow Templates

### Simple Echo
```bash
curl -X POST http://localhost:8002/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Echo Test",
    "definition": {
      "tasks": [{"id": "t1", "type": "echo", "payload": {"message": "Test!"}}]
    }
  }'
```

### API Call
```bash
curl -X POST http://localhost:8002/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "API Call",
    "definition": {
      "tasks": [{
        "id": "api",
        "type": "http_request",
        "payload": {
          "url": "https://jsonplaceholder.typicode.com/posts/1",
          "method": "GET"
        }
      }]
    }
  }'
```

### Multi-Step
```bash
curl -X POST http://localhost:8002/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Multi-Step",
    "definition": {
      "tasks": [
        {"id": "t1", "type": "echo", "payload": {"message": "Start"}},
        {"id": "t2", "type": "echo", "payload": {"message": "Middle"}, "depends_on": ["t1"]},
        {"id": "t3", "type": "echo", "payload": {"message": "End"}, "depends_on": ["t2"]}
      ]
    }
  }'
```

---

**Need help?** Check the full docs or run `docker-compose logs -f` to see what's happening!
