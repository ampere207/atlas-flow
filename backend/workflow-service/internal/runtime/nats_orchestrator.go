package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"atlasflow/backend/shared/models"
	sharedruntime "atlasflow/backend/shared/runtime"
	"atlasflow/backend/workflow-service/internal/repository"

	"github.com/nats-io/nats.go"
)

// NATSOrchestrator uses NATS to coordinate workflow execution across real workers
type NATSOrchestrator struct {
	repo              *repository.WorkflowRepository
	natsConn          *nats.Conn
	workerConnMgr     *sharedruntime.WorkerConnectionManager
	scheduler         *sharedruntime.TaskScheduler
	retryManager      *sharedruntime.RetryManager
	recoveryManager   *sharedruntime.FailureRecoveryManager
	eventBus          sharedruntime.EventPublisher
	resultListeners   map[string]chan *TaskResultMessage
	resultListenersMu sync.RWMutex
	ctx               context.Context
	cancel            context.CancelFunc
	orchestrationTick time.Duration
}

// TaskResultMessage is received from workers
type TaskResultMessage struct {
	TaskID     string                 `json:"task_id"`
	WorkflowID string                 `json:"workflow_id"`
	Success    bool                   `json:"success"`
	Output     map[string]interface{} `json:"output,omitempty"`
	Error      string                 `json:"error,omitempty"`
	Duration   int64                  `json:"duration_ms"`
	Retryable  bool                   `json:"retryable"`
}

// NewNATSOrchestrator creates a NATS-based orchestrator
func NewNATSOrchestrator(repo *repository.WorkflowRepository,
	nc *nats.Conn,
	workerConnMgr *sharedruntime.WorkerConnectionManager,
	eventBus sharedruntime.EventPublisher) *NATSOrchestrator {

	ctx, cancel := context.WithCancel(context.Background())

	orch := &NATSOrchestrator{
		repo:              repo,
		natsConn:          nc,
		workerConnMgr:     workerConnMgr,
		eventBus:          eventBus,
		resultListeners:   make(map[string]chan *TaskResultMessage),
		ctx:               ctx,
		cancel:            cancel,
		orchestrationTick: 1 * time.Second,
		retryManager: sharedruntime.NewRetryManager(sharedruntime.RetryPolicy{
			MaxAttempts:       3,
			InitialBackoff:    1 * time.Second,
			MaxBackoff:        5 * time.Minute,
			BackoffMultiplier: 2.0,
		}),
		recoveryManager: sharedruntime.NewFailureRecoveryManager(10*time.Minute, 30*time.Second),
	}

	// Start listening for task results
	go orch.listenForTaskResults()

	return orch
}

// Start begins the orchestration loop
func (no *NATSOrchestrator) Start(ctx context.Context) error {
	ticker := time.NewTicker(no.orchestrationTick)
	defer ticker.Stop()

	log.Println("✓ NATS Orchestrator started")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-no.ctx.Done():
			return nil
		case <-ticker.C:
			no.orchestrationCycle()
		}
	}
}

// orchestrationCycle runs the main orchestration logic
func (no *NATSOrchestrator) orchestrationCycle() {
	// 1. Get all running workflows
	workflows, err := no.repo.GetRunningWorkflows()
	if err != nil {
		log.Printf("✗ Failed to get running workflows: %v", err)
		return
	}

	for _, workflow := range workflows {
		no.processWorkflow(workflow)
	}

	// 2. Handle failure recovery (detect dead workers)
	no.handleFailureRecovery()
}

// processWorkflow orchestrates a single workflow
func (no *NATSOrchestrator) processWorkflow(workflow *models.Workflow) {
	// Parse workflow definition
	var definition sharedruntime.WorkflowDefinition
	err := json.Unmarshal([]byte(workflow.Definition), &definition)
	if err != nil {
		log.Printf("✗ Failed to parse workflow %s: %v", workflow.ID, err)
		return
	}

	// Get all tasks for this workflow
	tasks, err := no.repo.ListTasksByWorkflow(workflow.ID, workflow.UserID)
	if err != nil {
		log.Printf("✗ Failed to get tasks for workflow %s: %v", workflow.ID, err)
		return
	}

	// Check if workflow is complete
	if no.isWorkflowComplete(tasks) {
		log.Printf("✓ Workflow complete: %s", workflow.ID)
		no.repo.UpdateWorkflowStatus(workflow.ID, "completed")

		// Publish workflow_completed event
		if no.eventBus != nil {
			event := sharedruntime.NewEventBuilder(sharedruntime.EventWorkflowCompleted).
				WorkflowID(workflow.ID).
				UserID(workflow.UserID).
				Build()
			no.eventBus.PublishEvent(context.Background(), event)
		}
		return
	}

	// Determine ready tasks (no unmet dependencies)
	readyTasks := no.getReadyTasks(tasks)

	// Send ready tasks to available workers
	for _, task := range readyTasks {
		if task.State == string(sharedruntime.TaskStatePending) {
			no.sendTaskToWorker(task, workflow.UserID)
		}
	}

	// Check for and handle dead worker tasks
	no.handleOrphanedTasks(tasks)
}

// getReadyTasks returns tasks that are ready to execute (all dependencies met)
func (no *NATSOrchestrator) getReadyTasks(tasks []*models.Task) []*models.Task {
	// Build task state map
	taskStates := make(map[string]sharedruntime.TaskState)
	for _, task := range tasks {
		taskStates[task.ID] = sharedruntime.TaskState(task.State)
	}

	ready := make([]*models.Task, 0)

	for _, task := range tasks {
		// Skip if already assigned or running
		if task.State == string(sharedruntime.TaskStateAssigned) ||
			task.State == string(sharedruntime.TaskStateRunning) ||
			task.State == string(sharedruntime.TaskStateCompleted) {
			continue
		}

		// Check if all dependencies are met
		if task.DependsOn != "" {
			var deps []string
			json.Unmarshal([]byte(task.DependsOn), &deps)

			allMet := true
			for _, depID := range deps {
				// Find dependency task
				depTask := no.findTask(depID, tasks)
				if depTask == nil || depTask.State != string(sharedruntime.TaskStateCompleted) {
					allMet = false
					break
				}
			}

			if !allMet {
				continue
			}
		}

		ready = append(ready, task)
	}

	return ready
}

// sendTaskToWorker sends a task to an available worker via NATS
func (no *NATSOrchestrator) sendTaskToWorker(task *models.Task, userID string) {
	// Log available workers for debugging
	allWorkers := no.workerConnMgr.GetAllWorkers()
	log.Printf("[task dispatch] task=%s type=%s user=%s available_workers=%d", task.ID, task.TaskType, userID, len(allWorkers))
	for _, w := range allWorkers {
		log.Printf("  - worker=%s user=%s capabilities=%v status=%s", w.WorkerID, w.UserID, w.Capabilities, w.Status)
	}

	// Find an available worker that can handle this task (scoped to user for multi-tenant isolation)
	worker := no.workerConnMgr.FindWorkerForTaskByUser(task.TaskType, userID)

	if worker == nil {
		log.Printf("! No workers for user %s, falling back to any available worker for task %s (%s)", userID, task.ID, task.TaskType)
		// Fallback: use any available worker that can handle this task type (for demo/testing)
		worker = no.workerConnMgr.FindWorkerForTask(task.TaskType)
	}

	if worker == nil {
		log.Printf("! No available workers for task %s (%s) - even with fallback", task.ID, task.TaskType)
		return
	}

	log.Printf("[task dispatch] assigning task=%s to worker=%s user=%s", task.ID, worker.WorkerID, worker.UserID)

	// Prepare task message
	taskMsg := map[string]interface{}{
		"task_id":      task.ID,
		"workflow_id":  task.WorkflowID,
		"task_type":    task.TaskType,
		"payload":      task.Payload,
		"max_retries":  task.MaxRetries,
		"timeout_secs": 300, // 5 minute default
	}

	taskJSON, _ := json.Marshal(taskMsg)

	// Send task to worker via NATS
	subject := fmt.Sprintf("workers.%s.tasks", worker.WorkerID)
	err := no.natsConn.Publish(subject, taskJSON)

	if err != nil {
		log.Printf("✗ Failed to send task to worker: %v", err)
		return
	}

	// Mark task as assigned
	no.repo.UpdateTaskStateOrchestrator(task.ID, string(sharedruntime.TaskStateAssigned), worker.WorkerID)

	// Set up result listener for this task
	no.setupResultListener(task.ID)

	log.Printf("[→] Task %s sent to worker %s", task.ID, worker.WorkerID)

	// Publish task_assigned event
	if no.eventBus != nil {
		event := sharedruntime.NewEventBuilder(sharedruntime.EventTaskAssigned).
			WorkflowID(task.WorkflowID).
			TaskID(task.ID).
			UserID(userID).
			Data("worker_id", worker.WorkerID).
			Build()
		no.eventBus.PublishEvent(context.Background(), event)
	}
}

// listenForTaskResults subscribes to all task result publications
func (no *NATSOrchestrator) listenForTaskResults() {
	_, err := no.natsConn.Subscribe("tasks.*.result", func(msg *nats.Msg) {
		var result TaskResultMessage
		err := json.Unmarshal(msg.Data, &result)
		if err != nil {
			log.Printf("✗ Failed to unmarshal task result: %v", err)
			return
		}

		// Route to result listener
		no.resultListenersMu.RLock()
		listener, exists := no.resultListeners[result.TaskID]
		no.resultListenersMu.RUnlock()

		if exists {
			select {
			case listener <- &result:
			case <-time.After(1 * time.Second):
				log.Printf("! Result listener timeout for task %s", result.TaskID)
			}
		}
	})

	if err != nil {
		log.Printf("✗ Failed to subscribe to task results: %v", err)
		return
	}

	log.Println("✓ Listening for task results on: tasks.*.result")

	<-no.ctx.Done()
}

// setupResultListener creates a listener for a specific task result
func (no *NATSOrchestrator) setupResultListener(taskID string) {
	no.resultListenersMu.Lock()
	defer no.resultListenersMu.Unlock()

	listener := make(chan *TaskResultMessage, 1)
	no.resultListeners[taskID] = listener

	// Handle result in background
	go no.handleTaskResult(taskID, listener)
}

// handleTaskResult processes the result of a task execution
func (no *NATSOrchestrator) handleTaskResult(taskID string, resultChan chan *TaskResultMessage) {
	timeout := time.After(10 * time.Minute)

	select {
	case result := <-resultChan:
		no.processTaskResult(taskID, result)

	case <-timeout:
		log.Printf("! Task result timeout: %s", taskID)
		no.resultListenersMu.Lock()
		delete(no.resultListeners, taskID)
		no.resultListenersMu.Unlock()
	}
}

// processTaskResult handles a completed task
func (no *NATSOrchestrator) processTaskResult(taskID string, result *TaskResultMessage) {
	// Update task state
	if result.Success {
		no.repo.UpdateTaskStateOrchestrator(taskID, string(sharedruntime.TaskStateCompleted), "")
		log.Printf("[✓] Task completed: %s", taskID)

		// Publish task_completed event
		if no.eventBus != nil {
			event := sharedruntime.NewEventBuilder(sharedruntime.EventTaskCompleted).
				WorkflowID(result.WorkflowID).
				TaskID(taskID).
				Data("duration_ms", result.Duration).
				Build()
			no.eventBus.PublishEvent(context.Background(), event)
		}
	} else {
		// Task failed - decide whether to retry
		task, _ := no.repo.GetTask(taskID, result.WorkflowID)

		if task != nil && no.retryManager.CanRetry(task.RetryCount+1, result.Error) {
			// Schedule retry
			retryTime := no.retryManager.NextRetryTime(task.RetryCount + 1)
			no.repo.UpdateTaskStateOrchestrator(taskID, string(sharedruntime.TaskStateRetrying), "")
			no.repo.UpdateTaskRetryCount(taskID, task.RetryCount+1)

			log.Printf("[↻] Task will retry: %s (attempt %d)", taskID, task.RetryCount+2)

			// Publish task_retrying event
			if no.eventBus != nil {
				event := sharedruntime.NewEventBuilder(sharedruntime.EventTaskRetrying).
					WorkflowID(result.WorkflowID).
					TaskID(taskID).
					Data("retry_count", task.RetryCount+1).
					Data("next_retry", retryTime).
					Build()
				no.eventBus.PublishEvent(context.Background(), event)
			}
		} else {
			// Permanent failure
			no.repo.UpdateTaskStateOrchestrator(taskID, string(sharedruntime.TaskStateFailed), "")
			no.repo.UpdateTaskError(taskID, result.Error)

			log.Printf("[✗] Task failed permanently: %s - %s", taskID, result.Error)

			// Publish task_failed event
			if no.eventBus != nil {
				event := sharedruntime.NewEventBuilder(sharedruntime.EventTaskFailed).
					WorkflowID(result.WorkflowID).
					TaskID(taskID).
					Data("error", result.Error).
					Build()
				no.eventBus.PublishEvent(context.Background(), event)
			}
		}
	}

	// Clean up listener
	no.resultListenersMu.Lock()
	delete(no.resultListeners, taskID)
	no.resultListenersMu.Unlock()
}

// handleOrphanedTasks handles tasks from dead workers
func (no *NATSOrchestrator) handleOrphanedTasks(tasks []*models.Task) {
	for _, task := range tasks {
		if task.State != string(sharedruntime.TaskStateAssigned) &&
			task.State != string(sharedruntime.TaskStateRunning) {
			continue
		}

		if task.AssignedWorkerID == "" {
			continue
		}

		// Check if worker is still alive
		worker := no.workerConnMgr.GetWorkerStatus(task.AssignedWorkerID)
		if worker != nil && worker.Status == "connected" {
			continue
		}

		// Worker is dead - reassign task
		log.Printf("! Orphaned task detected: %s (worker %s is dead)", task.ID, task.AssignedWorkerID)

		// Reset task to pending
		no.repo.UpdateTaskStateOrchestrator(task.ID, string(sharedruntime.TaskStatePending), "")

		// Publish task_reassigned event
		if no.eventBus != nil {
			event := sharedruntime.NewEventBuilder(sharedruntime.EventTaskReassigned).
				WorkflowID(task.WorkflowID).
				TaskID(task.ID).
				Data("previous_worker", task.AssignedWorkerID).
				Build()
			no.eventBus.PublishEvent(context.Background(), event)
		}
	}
}

// handleFailureRecovery periodically checks for dead workers
func (no *NATSOrchestrator) handleFailureRecovery() {
	// This is run by detectDeadWorkers in WorkerConnectionManager
	// but we can add additional recovery logic here if needed
}

// isWorkflowComplete checks if all tasks in a workflow are done
func (no *NATSOrchestrator) isWorkflowComplete(tasks []*models.Task) bool {
	if len(tasks) == 0 {
		return false
	}

	for _, task := range tasks {
		if task.State != string(sharedruntime.TaskStateCompleted) &&
			task.State != string(sharedruntime.TaskStateFailed) {
			return false
		}
	}

	return true
}

// findTask finds a task by ID in a slice
func (no *NATSOrchestrator) findTask(taskID string, tasks []*models.Task) *models.Task {
	for _, task := range tasks {
		if task.ID == taskID {
			return task
		}
	}
	return nil
}

// Stop stops the orchestrator
func (no *NATSOrchestrator) Stop() {
	no.cancel()
}
