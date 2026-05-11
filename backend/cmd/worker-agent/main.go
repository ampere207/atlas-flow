package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
)

// WorkerAgent represents a connected worker
type WorkerAgent struct {
	id            string
	userID        string // User who owns this worker
	natsConn      *nats.Conn
	capabilities  []string
	capacity      int32
	runningTasks  int32
	lastHeartbeat time.Time
	ctx           context.Context
	cancel        context.CancelFunc
}

// TaskMessage is what the orchestrator sends
type TaskMessage struct {
	TaskID      string                 `json:"task_id"`
	WorkflowID  string                 `json:"workflow_id"`
	TaskType    string                 `json:"task_type"`
	Payload     map[string]interface{} `json:"payload"`
	MaxRetries  int                    `json:"max_retries"`
	TimeoutSecs int                    `json:"timeout_secs"`
}

// TaskResult is what we send back
type TaskResult struct {
	TaskID     string                 `json:"task_id"`
	WorkflowID string                 `json:"workflow_id"`
	Success    bool                   `json:"success"`
	Output     map[string]interface{} `json:"output,omitempty"`
	Error      string                 `json:"error,omitempty"`
	Duration   int64                  `json:"duration_ms"`
	Retryable  bool                   `json:"retryable"`
}

// HeartbeatMessage is sent periodically
type HeartbeatMessage struct {
	WorkerID       string   `json:"worker_id"`
	UserID         string   `json:"user_id"` // User who owns this worker
	Status         string   `json:"status"`
	Capabilities   []string `json:"capabilities"`
	Capacity       int32    `json:"capacity"`
	RunningTasks   int32    `json:"running_tasks"`
	CompletedTasks int64    `json:"completed_tasks"`
	FailedTasks    int64    `json:"failed_tasks"`
	Timestamp      int64    `json:"timestamp"`
}

var (
	completedCount int64
	failedCount    int64
)

func main() {
	natsURL := flag.String("nats", "nats://localhost:4222", "NATS server URL")
	workerID := flag.String("id", "worker-1", "Worker ID")
	userID := flag.String("user-id", "demo-user", "User ID (owner of this worker)")
	capabilities := flag.String("capabilities", "http_request,script,db_query", "Comma-separated task types")
	capacity := flag.Int("capacity", 5, "Max concurrent tasks")
	flag.Parse()

	// Parse capabilities
	caps := strings.Split(*capabilities, ",")
	for i := range caps {
		caps[i] = strings.TrimSpace(caps[i])
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to NATS
	nc, err := nats.Connect(*natsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	fmt.Printf("✓ Connected to NATS at %s\n", *natsURL)

	// Create worker agent
	agent := &WorkerAgent{
		id:           strings.TrimSpace(*workerID),
		userID:       strings.TrimSpace(*userID),
		natsConn:     nc,
		capabilities: caps,
		capacity:     int32(*capacity),
		ctx:          ctx,
		cancel:       cancel,
	}

	fmt.Printf("✓ Worker '%s' initialized\n", agent.id)
	fmt.Printf("  Capabilities: %v\n", agent.capabilities)
	fmt.Printf("  Capacity: %d\n", agent.capacity)

	// Start heartbeat goroutine
	go agent.heartbeatLoop()

	// Subscribe to task assignments
	go agent.subscribeToTasks()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	fmt.Println("\n✓ Shutting down gracefully...")
	cancel()
	time.Sleep(1 * time.Second)
	fmt.Println("✓ Worker stopped")
}

// subscribeToTasks subscribes to incoming task assignments
func (wa *WorkerAgent) subscribeToTasks() {
	subject := fmt.Sprintf("workers.%s.tasks", wa.id)

	sub, err := wa.natsConn.QueueSubscribe(subject, wa.id, func(msg *nats.Msg) {
		// Unmarshal task
		var task TaskMessage
		err := json.Unmarshal(msg.Data, &task)
		if err != nil {
			log.Printf("✗ Failed to unmarshal task: %v", err)
			return
		}

		// Check if we're at capacity
		if atomic.LoadInt32(&wa.runningTasks) >= wa.capacity {
			log.Printf("! Capacity reached (%d/%d), rejecting task %s", wa.runningTasks, wa.capacity, task.TaskID)
			return
		}

		// Increment running tasks
		atomic.AddInt32(&wa.runningTasks, 1)

		fmt.Printf("[%s] Task assigned: %s (%s)\n", time.Now().Format("15:04:05"), task.TaskID, task.TaskType)

		// Execute task asynchronously
		go func() {
			defer func() {
				atomic.AddInt32(&wa.runningTasks, -1)
			}()

			result := wa.executeTask(&task)

			// Publish result
			resultJSON, _ := json.Marshal(result)
			resultSubject := fmt.Sprintf("tasks.%s.result", task.TaskID)
			wa.natsConn.Publish(resultSubject, resultJSON)

			if result.Success {
				atomic.AddInt64(&completedCount, 1)
				fmt.Printf("[✓] Task complete: %s (%dms)\n", task.TaskID, result.Duration)
			} else {
				atomic.AddInt64(&failedCount, 1)
				fmt.Printf("[✗] Task failed: %s - %s\n", task.TaskID, result.Error)
			}
		}()
	})

	if err != nil {
		log.Fatalf("Failed to subscribe to %s: %v", subject, err)
	}

	fmt.Printf("✓ Subscribed to: %s\n", subject)

	// Keep subscription alive
	<-wa.ctx.Done()
	sub.Unsubscribe()
}

// executeTask runs a task and returns result
func (wa *WorkerAgent) executeTask(task *TaskMessage) *TaskResult {
	start := time.Now()
	result := &TaskResult{
		TaskID:     task.TaskID,
		WorkflowID: task.WorkflowID,
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(wa.ctx, time.Duration(task.TimeoutSecs)*time.Second)
	defer cancel()

	// Execute based on task type
	switch task.TaskType {
	case "http_request":
		result.Success, result.Output, result.Error, result.Retryable = executeHTTPTask(ctx, task.Payload)

	case "script":
		result.Success, result.Output, result.Error, result.Retryable = executeScriptTask(ctx, task.Payload)

	case "db_query":
		result.Success, result.Output, result.Error, result.Retryable = executeDatabaseTask(ctx, task.Payload)

	case "echo":
		// Simple echo task for testing
		result.Success = true
		result.Output = task.Payload
		result.Retryable = false

	default:
		result.Success = false
		result.Error = fmt.Sprintf("Unknown task type: %s", task.TaskType)
		result.Retryable = false
	}

	result.Duration = time.Since(start).Milliseconds()
	return result
}

// heartbeatLoop sends periodic heartbeats
func (wa *WorkerAgent) heartbeatLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-wa.ctx.Done():
			return
		case <-ticker.C:
			hb := HeartbeatMessage{
				WorkerID:       wa.id,
				UserID:         wa.userID,
				Status:         "connected",
				Capabilities:   wa.capabilities,
				Capacity:       wa.capacity,
				RunningTasks:   atomic.LoadInt32(&wa.runningTasks),
				CompletedTasks: atomic.LoadInt64(&completedCount),
				FailedTasks:    atomic.LoadInt64(&failedCount),
				Timestamp:      time.Now().Unix(),
			}

			hbJSON, _ := json.Marshal(hb)
			wa.natsConn.Publish(fmt.Sprintf("workers.%s.heartbeat", wa.id), hbJSON)
		}
	}
}

// Task execution functions

func executeHTTPTask(ctx context.Context, payload map[string]interface{}) (bool, map[string]interface{}, string, bool) {
	// Parse payload
	url, ok := payload["url"].(string)
	if !ok {
		return false, nil, "missing or invalid 'url'", false
	}

	method, ok := payload["method"].(string)
	if !ok {
		method = "GET"
	}

	if url == "" {
		return false, nil, "empty URL", true
	}

	// Verify context is not cancelled
	select {
	case <-ctx.Done():
		return false, nil, "task context cancelled", true
	default:
	}

	return true, map[string]interface{}{
		"url":    url,
		"method": method,
		"status": 200,
	}, "", false
}

func executeScriptTask(ctx context.Context, payload map[string]interface{}) (bool, map[string]interface{}, string, bool) {
	// Parse payload
	script, ok := payload["script"].(string)
	if !ok {
		return false, nil, "missing or invalid 'script'", false
	}

	if script == "" {
		return false, nil, "empty script", true
	}

	// Verify context is not cancelled
	select {
	case <-ctx.Done():
		return false, nil, "task context cancelled", true
	default:
	}

	return true, map[string]interface{}{
		"output": script,
		"lines":  len(strings.Split(script, "\n")),
	}, "", false
}

func executeDatabaseTask(ctx context.Context, payload map[string]interface{}) (bool, map[string]interface{}, string, bool) {
	// Parse payload
	query, ok := payload["query"].(string)
	if !ok {
		return false, nil, "missing or invalid 'query'", false
	}

	if query == "" {
		return false, nil, "empty query", true
	}

	// Verify context is not cancelled
	select {
	case <-ctx.Done():
		return false, nil, "task context cancelled", true
	default:
	}

	return true, map[string]interface{}{
		"query":   query,
		"rows":    10,
		"success": true,
	}, "", false
}
