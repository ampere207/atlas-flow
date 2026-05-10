#!/bin/bash
# Atlas Flow - Demo Workflows Script
# This script creates and executes demo workflows to showcase the system

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

API_URL="http://localhost:8002"
DELAY=2

echo ""
echo -e "${CYAN}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║     Atlas Flow - Demo Workflows (Real Worker Orchestration)    ║${NC}"
echo -e "${CYAN}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Function to check if API is ready
check_api_ready() {
    echo -e "${BLUE}Checking if orchestrator API is ready...${NC}"
    for i in {1..30}; do
        if curl -s "$API_URL/health" > /dev/null 2>&1; then
            echo -e "${GREEN}✓ API is ready${NC}"
            return 0
        fi
        echo -n "."
        sleep 1
    done
    echo -e "${RED}✗ API is not responding${NC}"
    return 1
}

# Function to create and execute a workflow
run_demo_workflow() {
    local name=$1
    local json=$2
    
    echo ""
    echo -e "${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}Demo: $name${NC}"
    echo -e "${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    echo ""
    echo -e "${BLUE}1. Creating workflow...${NC}"
    RESPONSE=$(curl -s -X POST "$API_URL/workflows" \
        -H "Content-Type: application/json" \
        -d "$json")
    
    WORKFLOW_ID=$(echo "$RESPONSE" | grep -o '"id":"[^"]*' | cut -d'"' -f4)
    
    if [ -z "$WORKFLOW_ID" ]; then
        echo -e "${RED}✗ Failed to create workflow${NC}"
        echo "Response: $RESPONSE"
        return 1
    fi
    
    echo -e "${GREEN}✓ Workflow created: $WORKFLOW_ID${NC}"
    
    echo ""
    echo -e "${BLUE}2. Executing workflow...${NC}"
    curl -s -X POST "$API_URL/workflows/$WORKFLOW_ID/execute" \
        -H "Content-Type: application/json" > /dev/null
    echo -e "${GREEN}✓ Execution started${NC}"
    
    echo ""
    echo -e "${BLUE}3. Monitoring execution...${NC}"
    for i in {1..30}; do
        STATUS=$(curl -s "$API_URL/workflows/$WORKFLOW_ID" | grep -o '"status":"[^"]*' | cut -d'"' -f4)
        
        if [ "$STATUS" = "completed" ]; then
            echo -e "${GREEN}✓ Workflow completed!${NC}"
            
            # Show task results
            echo ""
            echo -e "${BLUE}Task Results:${NC}"
            curl -s "$API_URL/workflows/$WORKFLOW_ID" | jq '.tasks[] | {id: .id, state: .state, result: .result}' 2>/dev/null || echo "  Check orchestrator logs for details"
            return 0
            
        elif [ "$STATUS" = "failed" ]; then
            echo -e "${RED}✗ Workflow failed${NC}"
            return 1
        else
            echo -n "."
            sleep 1
        fi
    done
    
    echo -e "${YELLOW}⏳ Still executing (check logs for details)${NC}"
}

# Check API is ready
check_api_ready || exit 1

echo ""
echo -e "${BLUE}Starting demo workflows...${NC}"
echo ""

# Demo 1: Simple Echo Pipeline
echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}DEMO 1: Simple Echo Pipeline (Sequential)${NC}"
echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"

DEMO1='{
  "name": "Echo Pipeline",
  "definition": {
    "tasks": [
      {
        "id": "greeting",
        "type": "echo",
        "payload": {
          "message": "Starting Atlas Flow demo..."
        }
      },
      {
        "id": "status",
        "type": "echo",
        "payload": {
          "message": "All systems operational"
        },
        "depends_on": ["greeting"]
      }
    ]
  }
}'

run_demo_workflow "Echo Pipeline" "$DEMO1"
sleep $DELAY

# Demo 2: Multi-Task Workflow
echo ""
echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}DEMO 2: Multi-Task Workflow (Mixed Task Types)${NC}"
echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"

DEMO2='{
  "name": "Multi-Task Demo",
  "definition": {
    "tasks": [
      {
        "id": "start",
        "type": "echo",
        "payload": {
          "message": "Workflow started"
        }
      },
      {
        "id": "fetch-data",
        "type": "http_request",
        "payload": {
          "url": "https://jsonplaceholder.typicode.com/posts/1",
          "method": "GET"
        },
        "depends_on": ["start"]
      },
      {
        "id": "process",
        "type": "script",
        "payload": {
          "script": "echo Processing data...",
          "timeout": 30
        },
        "depends_on": ["fetch-data"]
      },
      {
        "id": "store",
        "type": "db_query",
        "payload": {
          "query": "INSERT INTO results (data) VALUES (json_data)"
        },
        "depends_on": ["process"]
      }
    ]
  }
}'

run_demo_workflow "Multi-Task Workflow" "$DEMO2"
sleep $DELAY

# Demo 3: Parallel Task Execution
echo ""
echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}DEMO 3: Parallel Task Execution${NC}"
echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"

DEMO3='{
  "name": "Parallel Tasks",
  "definition": {
    "tasks": [
      {
        "id": "init",
        "type": "echo",
        "payload": {
          "message": "Initializing parallel jobs"
        }
      },
      {
        "id": "job1",
        "type": "http_request",
        "payload": {
          "url": "https://api.example.com/job1",
          "method": "GET"
        },
        "depends_on": ["init"]
      },
      {
        "id": "job2",
        "type": "http_request",
        "payload": {
          "url": "https://api.example.com/job2",
          "method": "GET"
        },
        "depends_on": ["init"]
      },
      {
        "id": "job3",
        "type": "script",
        "payload": {
          "script": "echo Job 3 completed"
        },
        "depends_on": ["init"]
      },
      {
        "id": "aggregate",
        "type": "echo",
        "payload": {
          "message": "All jobs completed, aggregating results"
        },
        "depends_on": ["job1", "job2", "job3"]
      }
    ]
  }
}'

run_demo_workflow "Parallel Tasks" "$DEMO3"
sleep $DELAY

# Demo 4: Worker Load Balancing
echo ""
echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}DEMO 4: Worker Load Balancing (Multiple Parallel Workflows)${NC}"
echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"

echo -e "${BLUE}Creating 3 concurrent workflows to demonstrate load balancing...${NC}"
echo ""

for i in {1..3}; do
    WORKFLOW_JSON="{
      \"name\": \"Load Test Workflow $i\",
      \"definition\": {
        \"tasks\": [
          {
            \"id\": \"task-$i-1\",
            \"type\": \"echo\",
            \"payload\": {
              \"message\": \"Workflow $i started\"
            }
          },
          {
            \"id\": \"task-$i-2\",
            \"type\": \"http_request\",
            \"payload\": {
              \"url\": \"https://api.example.com/data-$i\",
              \"method\": \"GET\"
            },
            \"depends_on\": [\"task-$i-1\"]
          }
        ]
      }
    }"
    
    RESPONSE=$(curl -s -X POST "$API_URL/workflows" \
        -H "Content-Type: application/json" \
        -d "$WORKFLOW_JSON")
    
    WF_ID=$(echo "$RESPONSE" | grep -o '"id":"[^"]*' | cut -d'"' -f4)
    
    if [ ! -z "$WF_ID" ]; then
        echo -e "${YELLOW}Workflow $i: $WF_ID${NC}"
        curl -s -X POST "$API_URL/workflows/$WF_ID/execute" > /dev/null
        echo -e "${GREEN}✓ Execution started${NC}"
    fi
    
    sleep 1
done

echo ""
echo -e "${CYAN}Monitor load balancing in worker logs:${NC}"
echo -e "  ${YELLOW}docker-compose logs -f worker-1 worker-2 worker-3${NC}"
echo ""

# Summary
echo -e "${CYAN}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║                   Demo Complete!                              ║${NC}"
echo -e "${CYAN}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${BLUE}📊 Monitor System:${NC}"
echo -e "  • Orchestrator: ${YELLOW}docker-compose logs -f workflow-service${NC}"
echo -e "  • All Workers:  ${YELLOW}docker-compose logs -f worker-1 worker-2 worker-3${NC}"
echo -e "  • All Logs:     ${YELLOW}docker-compose logs -f${NC}"
echo ""
echo -e "${BLUE}🔍 Check Workflow Status:${NC}"
echo -e "  ${YELLOW}curl http://localhost:8002/workflows | jq .${NC}"
echo ""
echo -e "${BLUE}📚 Learn More:${NC}"
echo -e "  • See USAGE_GUIDE.md for complete API documentation"
echo -e "  • See REAL_WORKER_SYSTEM.md for architecture details"
echo ""
