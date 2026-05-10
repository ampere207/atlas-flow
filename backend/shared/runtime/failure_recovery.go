package runtime

import (
	"context"
	"fmt"
	"time"
)

// FailureRecoveryManager detects and recovers from task failures and worker crashes.
type FailureRecoveryManager struct {
	taskTimeout      time.Duration // How long before a task is considered stuck
	heartbeatTimeout time.Duration // How long before a worker is considered dead
}

// NewFailureRecoveryManager creates a recovery manager.
func NewFailureRecoveryManager(taskTimeout, heartbeatTimeout time.Duration) *FailureRecoveryManager {
	if taskTimeout == 0 {
		taskTimeout = 10 * time.Minute
	}
	if heartbeatTimeout == 0 {
		heartbeatTimeout = 30 * time.Second
	}
	return &FailureRecoveryManager{
		taskTimeout:      taskTimeout,
		heartbeatTimeout: heartbeatTimeout,
	}
}

// RecoveredTask represents a task that needs recovery.
type RecoveredTask struct {
	TaskID            string
	PreviousWorker    string
	AssignedAt        time.Time
	LastUpdate        time.Time
	ReasonForRecovery string
}

// IsTaskStalled checks if a task has been running too long without updates.
func (frm *FailureRecoveryManager) IsTaskStalled(taskStartTime time.Time, lastUpdate time.Time) bool {
	if taskStartTime.IsZero() {
		return false
	}
	// Task is stalled if it's been running longer than timeout
	return time.Since(lastUpdate) > frm.taskTimeout
}

// IsWorkerDead checks if a worker hasn't heartbeated recently.
func (frm *FailureRecoveryManager) IsWorkerDead(lastHeartbeat time.Time) bool {
	if lastHeartbeat.IsZero() {
		return true
	}
	return time.Since(lastHeartbeat) > frm.heartbeatTimeout
}

// DetectOrphanedTasks finds tasks assigned to dead workers.
// This is critical for preventing permanent failures.
func (frm *FailureRecoveryManager) DetectOrphanedTasks(ctx context.Context,
	assignedTasks map[string]TaskAssignment,
	workerHealthStatus map[string]WorkerHealth) ([]RecoveredTask, error) {

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	orphaned := make([]RecoveredTask, 0)

	for taskID, assignment := range assignedTasks {
		workerID := assignment.AssignedWorkerID
		if workerID == "" {
			continue // Not assigned yet
		}

		health, exists := workerHealthStatus[workerID]
		if !exists || frm.IsWorkerDead(health.LastHeartbeat) {
			orphaned = append(orphaned, RecoveredTask{
				TaskID:            taskID,
				PreviousWorker:    workerID,
				AssignedAt:        assignment.AssignedAt,
				LastUpdate:        health.LastHeartbeat,
				ReasonForRecovery: fmt.Sprintf("worker %s heartbeat timeout", workerID),
			})
		}
	}

	return orphaned, nil
}

// DetectStalledTasks finds tasks stuck in running state.
func (frm *FailureRecoveryManager) DetectStalledTasks(ctx context.Context,
	runningTasks map[string]TaskAssignment) ([]RecoveredTask, error) {

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	stalled := make([]RecoveredTask, 0)
	now := time.Now()

	for taskID, assignment := range runningTasks {
		if assignment.State != TaskStateRunning {
			continue
		}

		// Check if task has been running too long
		if assignment.StartedAt.IsZero() {
			continue
		}

		if now.Sub(assignment.StartedAt) > frm.taskTimeout {
			stalled = append(stalled, RecoveredTask{
				TaskID:         taskID,
				PreviousWorker: assignment.AssignedWorkerID,
				AssignedAt:     assignment.AssignedAt,
				LastUpdate:     assignment.UpdatedAt,
				ReasonForRecovery: fmt.Sprintf("task running for %v (timeout: %v)",
					now.Sub(assignment.StartedAt), frm.taskTimeout),
			})
		}
	}

	return stalled, nil
}

// TaskAssignment tracks which worker is assigned to a task.
type TaskAssignment struct {
	TaskID           string
	WorkflowID       string
	AssignedWorkerID string
	State            TaskState
	AssignedAt       time.Time
	StartedAt        time.Time
	UpdatedAt        time.Time
	Attempt          int
	LeaseExpiry      time.Time
}

// IsLeaseExpired checks if the task assignment lease has expired.
func (ta *TaskAssignment) IsLeaseExpired() bool {
	return ta.LeaseExpiry.Before(time.Now())
}

// WorkerHealth tracks worker liveness.
type WorkerHealth struct {
	WorkerID      string
	Status        string
	LastHeartbeat time.Time
	Capacity      int
	AssignedTasks int
}

// IsHealthy checks if a worker is in good state.
func (wh *WorkerHealth) IsHealthy() bool {
	return wh.Status == "active" && !wh.IsUnresponsive()
}

// IsUnresponsive checks if a worker hasn't heartbeated.
func (wh *WorkerHealth) IsUnresponsive() bool {
	return time.Since(wh.LastHeartbeat) > 30*time.Second
}

// RecoveryAction represents an action to take to recover from a failure.
type RecoveryAction struct {
	ActionType string // "reassign", "fail", "retry"
	TaskID     string
	FromWorker string
	ToWorker   string
	Reason     string
	ExecutedAt time.Time
	Status     string // "pending", "completed", "failed"
}

// RecoveryPlan collects all recovery actions for a workflow.
type RecoveryPlan struct {
	WorkflowID string
	Actions    []RecoveryAction
	CreatedAt  time.Time
	ExecutedAt time.Time
	Status     string
}

// NewRecoveryPlan creates a recovery plan.
func NewRecoveryPlan(workflowID string) *RecoveryPlan {
	return &RecoveryPlan{
		WorkflowID: workflowID,
		Actions:    make([]RecoveryAction, 0),
		CreatedAt:  time.Now(),
		Status:     "pending",
	}
}

// AddAction adds a recovery action to the plan.
func (rp *RecoveryPlan) AddAction(action RecoveryAction) {
	rp.Actions = append(rp.Actions, action)
}

// MarkExecuted marks the plan as executed.
func (rp *RecoveryPlan) MarkExecuted() {
	rp.ExecutedAt = time.Now()
	rp.Status = "executed"
}
