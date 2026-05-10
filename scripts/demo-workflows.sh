#!/bin/bash
# Atlas Flow - Demo Workflows Script with Authentication
# This script creates and executes demo workflows with proper user authentication

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
AUTH_URL="http://localhost:8001/auth"
DELAY=2
LOG_PREFIX="[demo-workflows]"

# Load local environment variables if present.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [ -f "$SCRIPT_DIR/../.env.local" ]; then
  set -a
  # shellcheck disable=SC1090
  . <(tr -d '\r' < "$SCRIPT_DIR/../.env.local")
  set +a
elif [ -f "$SCRIPT_DIR/../.env" ]; then
  set -a
  # shellcheck disable=SC1090
  . <(tr -d '\r' < "$SCRIPT_DIR/../.env")
  set +a
fi

# Authentication is required from your own account. Provide either:
# - ATLASFLOW_ACCESS_TOKEN, or
# - ATLASFLOW_AUTH_EMAIL and ATLASFLOW_AUTH_PASSWORD

echo ""
echo -e "${CYAN}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║    Atlas Flow - Demo Workflows (With Authentication)          ║${NC}"
echo -e "${CYAN}║            Real Worker Orchestration & User Isolation         ║${NC}"
echo -e "${CYAN}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Function to check if API is ready
check_api_ready() {
    echo -e "${BLUE}Checking if services are ready...${NC}"
    for i in {1..30}; do
        if curl -s "$API_URL/health" > /dev/null 2>&1; then
            echo -e "${GREEN}✓ Orchestrator API is ready${NC}"
            return 0
        fi
        echo -n "."
        sleep 1
    done
    echo -e "${RED}✗ API is not responding${NC}"
    return 1
}

# Function to authenticate and get token
authenticate_user() {
    echo -e "${BLUE}Step 1: Authenticating user...${NC}"

    if [ -n "${ATLASFLOW_ACCESS_TOKEN:-}" ]; then
        ACCESS_TOKEN="$ATLASFLOW_ACCESS_TOKEN"
        echo -e "${GREEN}✓ Using access token from ATLASFLOW_ACCESS_TOKEN${NC}"
        echo ""
        return 0
    fi

    if [ -z "${ATLASFLOW_AUTH_EMAIL:-}" ] || [ -z "${ATLASFLOW_AUTH_PASSWORD:-}" ]; then
        echo -e "${RED}✗ Missing credentials${NC}"
        echo -e "${YELLOW}Set ATLASFLOW_ACCESS_TOKEN or ATLASFLOW_AUTH_EMAIL and ATLASFLOW_AUTH_PASSWORD.${NC}"
        return 1
    fi
    
    # Login only; never create or default to another account.
    LOGIN_RESPONSE=$(curl -s -X POST "$AUTH_URL/login" \
        -H "Content-Type: application/json" \
        -d "{\"email\": \"$ATLASFLOW_AUTH_EMAIL\", \"password\": \"$ATLASFLOW_AUTH_PASSWORD\"}")
    
    # Check if login was successful
    if echo "$LOGIN_RESPONSE" | grep -q '"access_token"'; then
        ACCESS_TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"access_token":"[^"]*' | cut -d'"' -f4)
        echo -e "${GREEN}✓ Login successful${NC}"
        echo "  Token: ${ACCESS_TOKEN:0:20}...${ACCESS_TOKEN: -10}"
    else
        echo -e "${RED}✗ Failed to authenticate${NC}"
        echo "Response: $LOGIN_RESPONSE"
        return 1
    fi
    
    echo ""
}

# Function to create and execute a workflow
run_demo_workflow() {
    local name=$1
    local json=$2
    local token=$3
    
    echo ""
    echo -e "${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}Demo: $name${NC}"
    echo -e "${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    echo ""
    echo -e "${BLUE}2. Creating workflow...${NC}"
    echo "$LOG_PREFIX creating workflow name=\"$name\" payload=$json"
    RESPONSE_FILE=$(mktemp)
    HTTP_STATUS=$(curl -sS -o "$RESPONSE_FILE" -w "%{http_code}" -X POST "$API_URL/workflows" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $token" \
        -d "$json" || true)
    RESPONSE=$(cat "$RESPONSE_FILE")
    rm -f "$RESPONSE_FILE"

    echo "$LOG_PREFIX workflow create response status=$HTTP_STATUS body=$RESPONSE"
    
    WORKFLOW_ID=$(echo "$RESPONSE" | grep -o '"id":"[^"]*' | cut -d'"' -f4 | head -1)
    
    if [ -z "$WORKFLOW_ID" ]; then
        echo -e "${RED}✗ Failed to create workflow${NC}"
        echo "HTTP Status: $HTTP_STATUS"
        echo "Response: $RESPONSE"
        return 1
    fi
    
    echo -e "${GREEN}✓ Workflow created: $WORKFLOW_ID${NC}"
    
    echo ""
    echo -e "${BLUE}3. Executing workflow...${NC}"
    echo "$LOG_PREFIX executing workflow id=$WORKFLOW_ID"
    EXEC_RESPONSE_FILE=$(mktemp)
    EXEC_HTTP_STATUS=$(curl -sS -o "$EXEC_RESPONSE_FILE" -w "%{http_code}" -X POST "$API_URL/workflows/$WORKFLOW_ID/execute" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $token" || true)
    EXEC_RESPONSE=$(cat "$EXEC_RESPONSE_FILE")
    rm -f "$EXEC_RESPONSE_FILE"

    echo "$LOG_PREFIX workflow execute response status=$EXEC_HTTP_STATUS body=$EXEC_RESPONSE"
    
    if echo "$EXEC_RESPONSE" | grep -q '"status"'; then
        echo -e "${GREEN}✓ Execution started${NC}"
    else
        echo -e "${RED}✗ Failed to execute workflow${NC}"
        echo "HTTP Status: $EXEC_HTTP_STATUS"
        echo "Response: $EXEC_RESPONSE"
    fi
    
    echo ""
    echo -e "${BLUE}4. Monitoring execution...${NC}"
    for i in {1..30}; do
        STATUS=$(curl -s "$API_URL/workflows/$WORKFLOW_ID" \
            -H "Authorization: Bearer $token" | grep -o '"status":"[^"]*' | cut -d'"' -f4)
        
        if [ "$STATUS" = "completed" ]; then
            echo -e "${GREEN}✓ Workflow completed!${NC}"
            
            # Show task results
            echo ""
            echo -e "${BLUE}Task Results:${NC}"
            curl -s "$API_URL/workflows/$WORKFLOW_ID" \
                -H "Authorization: Bearer $token" | jq '.tasks[] | {id: .id, state: .state}' 2>/dev/null || echo "  Check orchestrator logs for details"
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

# Authenticate and get token
authenticate_user
if [ -z "$ACCESS_TOKEN" ]; then
    echo -e "${RED}Failed to authenticate. Exiting.${NC}"
    exit 1
fi

echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}User Authentication & Isolation${NC}"
echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "${BLUE}Important:${NC}"
if [ -n "${ATLASFLOW_AUTH_EMAIL:-}" ]; then
  echo -e "  • Authenticated account: ${YELLOW}$ATLASFLOW_AUTH_EMAIL${NC}"
else
  echo -e "  • Authenticated account: ${YELLOW}(access token provided directly)${NC}"
fi
echo -e "  • Workers are visible only to the authenticated account"
echo -e "  • No demo account fallback is used"
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
        "id": "11111111-1111-1111-1111-111111111111",
        "type": "echo",
        "payload": {
          "message": "Starting Atlas Flow demo..."
        }
      },
      {
        "id": "22222222-2222-2222-2222-222222222222",
        "type": "echo",
        "payload": {
          "message": "All systems operational"
        },
        "depends_on": ["11111111-1111-1111-1111-111111111111"]
      }
    ]
  }
}'

run_demo_workflow "Echo Pipeline" "$DEMO1" "$ACCESS_TOKEN"
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
        "id": "33333333-3333-3333-3333-333333333333",
        "type": "echo",
        "payload": {
          "message": "Workflow started"
        }
      },
      {
        "id": "44444444-4444-4444-4444-444444444444",
        "type": "http_request",
        "payload": {
          "url": "https://jsonplaceholder.typicode.com/posts/1",
          "method": "GET"
        },
        "depends_on": ["33333333-3333-3333-3333-333333333333"]
      },
      {
        "id": "55555555-5555-5555-5555-555555555555",
        "type": "script",
        "payload": {
          "script": "echo Processing data...",
          "timeout": 30
        },
        "depends_on": ["44444444-4444-4444-4444-444444444444"]
      },
      {
        "id": "66666666-6666-6666-6666-666666666666",
        "type": "db_query",
        "payload": {
          "query": "INSERT INTO results (data) VALUES (json_data)"
        },
        "depends_on": ["55555555-5555-5555-5555-555555555555"]
      }
    ]
  }
}'

run_demo_workflow "Multi-Task Workflow" "$DEMO2" "$ACCESS_TOKEN"
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
        "id": "77777777-7777-7777-7777-777777777777",
        "type": "echo",
        "payload": {
          "message": "Initializing parallel jobs"
        }
      },
      {
        "id": "88888888-8888-8888-8888-888888888888",
        "type": "http_request",
        "payload": {
          "url": "https://api.example.com/job1",
          "method": "GET"
        },
        "depends_on": ["77777777-7777-7777-7777-777777777777"]
      },
      {
        "id": "99999999-9999-9999-9999-999999999999",
        "type": "http_request",
        "payload": {
          "url": "https://api.example.com/job2",
          "method": "GET"
        },
        "depends_on": ["77777777-7777-7777-7777-777777777777"]
      },
      {
        "id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
        "type": "script",
        "payload": {
          "script": "echo Job 3 completed"
        },
        "depends_on": ["77777777-7777-7777-7777-777777777777"]
      },
      {
        "id": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
        "type": "echo",
        "payload": {
          "message": "All jobs completed, aggregating results"
        },
        "depends_on": [
          "88888888-8888-8888-8888-888888888888",
          "99999999-9999-9999-9999-999999999999",
          "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
        ]
      }
    ]
  }
}'

run_demo_workflow "Parallel Tasks" "$DEMO3" "$ACCESS_TOKEN"
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
            \"id\": \"cccccccc-cccc-cccc-cccc-ccccccccccc$i\",
            \"type\": \"echo\",
            \"payload\": {
              \"message\": \"Workflow $i started\"
            }
          },
          {
            \"id\": \"dddddddd-dddd-dddd-dddd-ddddddddddd$i\",
            \"type\": \"http_request\",
            \"payload\": {
              \"url\": \"https://api.example.com/data-$i\",
              \"method\": \"GET\"
            },
            \"depends_on\": [\"cccccccc-cccc-cccc-cccc-ccccccccccc$i\"]
          }
        ]
      }
    }"
    
    RESPONSE=$(curl -s -X POST "$API_URL/workflows" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $ACCESS_TOKEN" \
        -d "$WORKFLOW_JSON")
    
    WF_ID=$(echo "$RESPONSE" | grep -o '"id":"[^"]*' | cut -d'"' -f4)
    
    if [ ! -z "$WF_ID" ]; then
        echo -e "${YELLOW}Workflow $i: $WF_ID${NC}"
        curl -s -X POST "$API_URL/workflows/$WF_ID/execute" \
            -H "Authorization: Bearer $ACCESS_TOKEN" > /dev/null
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
echo -e "${CYAN}║                      Demo Complete!                           ║${NC}"
echo -e "${CYAN}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${BLUE}🔐 Authentication & User Isolation:${NC}"
if [ -n "${ATLASFLOW_AUTH_EMAIL:-}" ]; then
  echo -e "  • Logged in as: ${YELLOW}$ATLASFLOW_AUTH_EMAIL${NC}"
else
  echo -e "  • Logged in with supplied access token${NC}"
fi
echo -e "  • All workflows are scoped to your account"
echo -e "  • Other users cannot see your workflows or workers"
echo ""
echo -e "${BLUE}📊 Monitor System:${NC}"
echo -e "  • Orchestrator: ${YELLOW}docker-compose logs -f workflow-service${NC}"
echo -e "  • All Workers:  ${YELLOW}docker-compose logs -f worker-1 worker-2 worker-3${NC}"
echo -e "  • All Logs:     ${YELLOW}docker-compose logs -f${NC}"
echo ""
echo -e "${BLUE}🔍 Check Workflows:${NC}"
echo -e "  ${YELLOW}curl -H \"Authorization: Bearer $ACCESS_TOKEN\" http://localhost:8002/workflows | jq .${NC}"
echo ""
echo -e "${BLUE}🔍 Check Workers (Isolated to User):${NC}"
echo -e "  ${YELLOW}curl -H \"Authorization: Bearer $ACCESS_TOKEN\" http://localhost:8002/workers | jq .${NC}"
echo ""
echo -e "${BLUE}📚 Learn More:${NC}"
echo -e "  • See USAGE_GUIDE.md for complete API documentation"
echo -e "  • See REAL_WORKER_SYSTEM.md for architecture details"
echo ""
