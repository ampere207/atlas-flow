package runtime

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

// WorkerConnection represents a connected worker's metadata
type WorkerConnection struct {
	WorkerID       string
	UserID         string // User who owns this worker
	Capabilities   []string
	Capacity       int32
	RunningTasks   int32
	CompletedTasks int64
	FailedTasks    int64
	LastHeartbeat  time.Time
	Status         string // connected, disconnected, dead
}

// HeartbeatData represents a worker heartbeat message
type HeartbeatData struct {
	WorkerID       string   `json:"worker_id"`
	UserID         string   `json:"user_id"` // User who owns this worker
	Status         string   `json:"status"`
	Capabilities   []string `json:"capabilities"`
	Capacity       int32    `json:"capacity"`
	RunningTasks   int32    `json:"running_tasks"`
	CompletedTasks int64    `json:"completed_tasks"`
	FailedTasks    int64    `json:"failed_tasks"`
}

// WorkerConnectionManager tracks connected workers via NATS heartbeats
type WorkerConnectionManager struct {
	natsConn    *nats.Conn
	connections map[string]*WorkerConnection
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewWorkerConnectionManager creates a manager for tracking worker connections
func NewWorkerConnectionManager(nc *nats.Conn) *WorkerConnectionManager {
	ctx, cancel := context.WithCancel(context.Background())

	wcm := &WorkerConnectionManager{
		natsConn:    nc,
		connections: make(map[string]*WorkerConnection),
		ctx:         ctx,
		cancel:      cancel,
	}

	// Start listening for heartbeats
	go wcm.listenForHeartbeats()

	// Start cleanup for dead workers
	go wcm.cleanupDeadWorkers()

	return wcm
}

// listenForHeartbeats subscribes to all worker heartbeats
func (wcm *WorkerConnectionManager) listenForHeartbeats() {
	// Subscribe to heartbeats from all workers: workers.*.heartbeat
	_, err := wcm.natsConn.Subscribe("workers.*.heartbeat", func(msg *nats.Msg) {
		var hb HeartbeatData
		err := json.Unmarshal(msg.Data, &hb)
		if err != nil {
			log.Printf("Failed to unmarshal heartbeat: %v", err)
			return
		}

		wcm.updateWorkerConnection(&hb)
	})

	if err != nil {
		log.Printf("Failed to subscribe to worker heartbeats: %v", err)
		return
	}

	log.Println("✓ Listening for worker heartbeats on: workers.*.heartbeat")

	<-wcm.ctx.Done()
}

// updateWorkerConnection updates or creates a worker connection record
func (wcm *WorkerConnectionManager) updateWorkerConnection(hb *HeartbeatData) {
	wcm.mu.Lock()
	defer wcm.mu.Unlock()

	conn, exists := wcm.connections[hb.WorkerID]

	if !exists {
		// New worker connected
		conn = &WorkerConnection{
			WorkerID: hb.WorkerID,
			UserID:   hb.UserID, // Capture user ownership from heartbeat
		}
		wcm.connections[hb.WorkerID] = conn
		log.Printf("✓ New worker connected: %s (user: %s, capabilities: %v)", hb.WorkerID, hb.UserID, hb.Capabilities)
	}

	// Update connection info
	conn.Status = "connected"
	conn.UserID = hb.UserID // Keep user ID in sync
	conn.Capabilities = hb.Capabilities
	conn.Capacity = hb.Capacity
	conn.RunningTasks = hb.RunningTasks
	conn.CompletedTasks = hb.CompletedTasks
	conn.FailedTasks = hb.FailedTasks
	conn.LastHeartbeat = time.Now()
}

// cleanupDeadWorkers periodically checks for workers that haven't heartbeated
func (wcm *WorkerConnectionManager) cleanupDeadWorkers() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-wcm.ctx.Done():
			return
		case <-ticker.C:
			wcm.detectDeadWorkers()
		}
	}
}

// detectDeadWorkers marks workers as dead if they haven't heartbeated
func (wcm *WorkerConnectionManager) detectDeadWorkers() {
	wcm.mu.Lock()
	defer wcm.mu.Unlock()

	now := time.Now()
	heartbeatTimeout := 30 * time.Second

	for workerID, conn := range wcm.connections {
		if conn.Status == "connected" && now.Sub(conn.LastHeartbeat) > heartbeatTimeout {
			conn.Status = "dead"
			log.Printf("! Worker detected as dead (no heartbeat): %s", workerID)
		}
	}
}

// GetAvailableWorkers returns all healthy connected workers
func (wcm *WorkerConnectionManager) GetAvailableWorkers() []*WorkerConnection {
	return wcm.GetAvailableWorkersByUser("")
}

// GetAvailableWorkersByUser returns workers owned by a specific user
func (wcm *WorkerConnectionManager) GetAvailableWorkersByUser(userID string) []*WorkerConnection {
	wcm.mu.RLock()
	defer wcm.mu.RUnlock()

	available := make([]*WorkerConnection, 0)
	for _, conn := range wcm.connections {
		if conn.Status == "connected" {
			// Filter by user if specified
			if userID != "" && conn.UserID != userID {
				continue
			}
			available = append(available, conn)
		}
	}

	return available
}

// GetWorkersByUser returns all workers owned by a user (including dead workers)
func (wcm *WorkerConnectionManager) GetWorkersByUser(userID string) []*WorkerConnection {
	wcm.mu.RLock()
	defer wcm.mu.RUnlock()

	workers := make([]*WorkerConnection, 0)
	for _, conn := range wcm.connections {
		if conn.UserID == userID {
			workers = append(workers, conn)
		}
	}

	return workers
}

// FindWorkerForTask finds an available worker that can handle the task type
func (wcm *WorkerConnectionManager) FindWorkerForTask(taskType string) *WorkerConnection {
	return wcm.FindWorkerForTaskByUser(taskType, "")
}

// FindWorkerForTaskByUser finds a worker for a specific task type owned by a user
func (wcm *WorkerConnectionManager) FindWorkerForTaskByUser(taskType, userID string) *WorkerConnection {
	wcm.mu.RLock()
	defer wcm.mu.RUnlock()

	var bestWorker *WorkerConnection
	minLoad := int32(999)

	for _, conn := range wcm.connections {
		if conn.Status != "connected" {
			continue
		}

		// Filter by user if specified (for multi-tenant isolation)
		if userID != "" && conn.UserID != userID {
			continue
		}

		// Check if worker can handle this task type
		canHandle := false
		for _, cap := range conn.Capabilities {
			if cap == taskType {
				canHandle = true
				break
			}
		}

		if !canHandle {
			continue
		}

		// Check if worker has capacity
		if conn.RunningTasks >= conn.Capacity {
			continue
		}

		// Pick least-loaded worker
		load := conn.RunningTasks
		if load < minLoad {
			minLoad = load
			bestWorker = conn
		}
	}

	return bestWorker
}

// GetWorkerStatus returns the connection status of a worker
func (wcm *WorkerConnectionManager) GetWorkerStatus(workerID string) *WorkerConnection {
	wcm.mu.RLock()
	defer wcm.mu.RUnlock()

	return wcm.connections[workerID]
}

// GetAllWorkers returns all known workers
func (wcm *WorkerConnectionManager) GetAllWorkers() []*WorkerConnection {
	wcm.mu.RLock()
	defer wcm.mu.RUnlock()

	workers := make([]*WorkerConnection, 0, len(wcm.connections))
	for _, conn := range wcm.connections {
		workers = append(workers, conn)
	}

	return workers
}

// Stop stops the connection manager
func (wcm *WorkerConnectionManager) Stop() {
	wcm.cancel()
}
