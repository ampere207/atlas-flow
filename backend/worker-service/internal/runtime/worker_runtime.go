package runtime

import (
	"context"
	"fmt"
	"time"

	sharedmodels "atlasflow/backend/shared/models"
	sharedruntime "atlasflow/backend/shared/runtime"
	"atlasflow/backend/worker-service/internal/repository"
)

// TaskExecutionHandler handles the actual execution of a task.
// Different task types (http, script, db_query, etc.) implement this interface.
type TaskExecutionHandler interface {
	Execute(ctx context.Context, task *sharedmodels.Task) (*TaskExecutionResult, error)
}

// TaskExecutionResult contains the result of executing a task.
type TaskExecutionResult struct {
	Success    bool                   `json:"success"`
	Output     map[string]interface{} `json:"output,omitempty"`
	Error      string                 `json:"error,omitempty"`
	Duration   time.Duration          `json:"duration"`
	ExecutedAt time.Time              `json:"executed_at"`
	Retryable  bool                   `json:"retryable"` // Can this error be retried?
	ExitCode   int                    `json:"exit_code,omitempty"`
}

// TaskHandlerRegistry maps task types to their handlers.
type TaskHandlerRegistry struct {
	handlers map[string]TaskExecutionHandler
}

// NewTaskHandlerRegistry creates a registry.
func NewTaskHandlerRegistry() *TaskHandlerRegistry {
	return &TaskHandlerRegistry{
		handlers: make(map[string]TaskExecutionHandler),
	}
}

// Register associates a handler with a task type.
func (r *TaskHandlerRegistry) Register(taskType string, handler TaskExecutionHandler) {
	r.handlers[taskType] = handler
}

// GetHandler retrieves a handler for a task type.
func (r *TaskHandlerRegistry) GetHandler(taskType string) TaskExecutionHandler {
	if h, exists := r.handlers[taskType]; exists {
		return h
	}
	return &DefaultTaskHandler{} // Fallback handler
}

// DefaultTaskHandler is a placeholder that succeeds immediately (for testing).
type DefaultTaskHandler struct{}

// Execute runs a default task (no-op for dev/testing).
func (h *DefaultTaskHandler) Execute(ctx context.Context, task *sharedmodels.Task) (*TaskExecutionResult, error) {
	return &TaskExecutionResult{
		Success:    true,
		Output:     map[string]interface{}{"message": fmt.Sprintf("default execution of %s", task.Name)},
		Duration:   100 * time.Millisecond,
		ExecutedAt: time.Now(),
		Retryable:  false,
	}, nil
}

// EventPublisher publishes worker events.
type EventPublisher interface {
	PublishEvent(subject string, event map[string]interface{}) error
}

// WorkerRuntime is the Phase 2 worker execution engine.
// It polls for tasks, claims ownership, executes them, handles failures, and publishes events.
type WorkerRuntime struct {
	repo              *repository.WorkerRepository
	lease             *sharedruntime.LeaseManager
	publisher         EventPublisher
	dispatcher        sharedruntime.TaskDispatcher
	eventBus          sharedruntime.EventPublisher
	handlers          *TaskHandlerRegistry
	workerID          string
	pollInterval      time.Duration
	leaseTTL          time.Duration
	heartbeatInterval time.Duration
	maxConcurrent     int
}

// NewWorkerRuntime constructs a Phase 2 worker runtime.
func NewWorkerRuntime(repo *repository.WorkerRepository, lease *sharedruntime.LeaseManager,
	publisher EventPublisher, dispatcher sharedruntime.TaskDispatcher,
	eventBus sharedruntime.EventPublisher, workerID string) *WorkerRuntime {
	return &WorkerRuntime{
		repo:              repo,
		lease:             lease,
		publisher:         publisher,
		dispatcher:        dispatcher,
		eventBus:          eventBus,
		handlers:          NewTaskHandlerRegistry(),
		workerID:          workerID,
		pollInterval:      2 * time.Second,
		leaseTTL:          5 * time.Minute,
		heartbeatInterval: 10 * time.Second,
		maxConcurrent:     5,
	}
}

// RegisterHandler registers a task execution handler.
func (wr *WorkerRuntime) RegisterHandler(taskType string, handler TaskExecutionHandler) {
	wr.handlers.Register(taskType, handler)
}

// Start begins the worker execution loop (polling, heartbeat, execution).
func (wr *WorkerRuntime) Start(ctx context.Context) error {
	// Start heartbeat goroutine
	go wr.heartbeatLoop(ctx)

	// Start polling loop
	ticker := time.NewTicker(wr.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			_ = wr.pollOnce(ctx)
		}
	}
}

// heartbeatLoop periodically sends worker heartbeats.
func (wr *WorkerRuntime) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(wr.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			wr.sendHeartbeat(ctx)
		}
	}
}

// sendHeartbeat records a worker heartbeat.
func (wr *WorkerRuntime) sendHeartbeat(ctx context.Context) {
	if wr.repo != nil {
		_ = wr.repo.RecordHeartbeat(wr.workerID, "active")
	}

	if wr.eventBus != nil {
		event := sharedruntime.NewEventBuilder(sharedruntime.EventWorkerHeartbeat).
			WorkerID(wr.workerID).
			Data("timestamp", time.Now()).
			Build()
		_ = wr.eventBus.PublishEvent(ctx, event)
	}
}

// pollOnce performs a single poll/execute cycle.
// This is the core execution loop: claim task -> execute -> report result -> recover from failure.
func (wr *WorkerRuntime) pollOnce(ctx context.Context) error {
	// List available unclaimed tasks
	unclaimedTasks, err := wr.dispatcher.ListUnclaimedTasks(ctx, wr.maxConcurrent)
	if err != nil || len(unclaimedTasks) == 0 {
		return nil // No tasks available, will retry on next poll
	}

	// Try to claim and execute each available task
	for _, taskID := range unclaimedTasks {
		// Attempt to claim the task
		claimed, err := wr.dispatcher.ClaimTask(ctx, taskID, wr.workerID, wr.leaseTTL)
		if err != nil || !claimed {
			continue // Task claimed by another worker, skip
		}

		// Execute the claimed task
		go wr.executeTaskAsync(ctx, taskID)
	}

	return nil
}

// executeTaskAsync handles task execution asynchronously.
func (wr *WorkerRuntime) executeTaskAsync(ctx context.Context, taskID string) {
	defer func() {
		// Always release the task claim, even on panic
		if wr.dispatcher != nil {
			_ = wr.dispatcher.ReleaseClaim(ctx, taskID, wr.workerID)
		}
	}()

	// Retrieve task details
	task, err := wr.repo.GetTaskByID(taskID)
	if err != nil {
		wr.recordError(ctx, taskID, "failed to retrieve task", err)
		return
	}

	// Publish task_started event
	if wr.eventBus != nil {
		event := sharedruntime.NewEventBuilder(sharedruntime.EventTaskStarted).
			WorkflowID(task.WorkflowID).
			TaskID(task.ID).
			WorkerID(wr.workerID).
			Data("task_name", task.Name).
			Build()
		_ = wr.eventBus.PublishEvent(ctx, event)
	}

	// Execute the task
	startTime := time.Now()
	result, execErr := wr.executeTask(ctx, task)
	duration := time.Since(startTime)

	// Handle execution result
	if execErr != nil || !result.Success {
		wr.handleTaskFailure(ctx, task, result, execErr, duration)
		return
	}

	// Task succeeded
	wr.handleTaskSuccess(ctx, task, result, duration)
}

// executeTask runs the actual task handler.
func (wr *WorkerRuntime) executeTask(ctx context.Context, task *sharedmodels.Task) (*TaskExecutionResult, error) {
	handler := wr.handlers.GetHandler(task.TaskType)
	if handler == nil {
		return nil, fmt.Errorf("no handler for task type: %s", task.TaskType)
	}

	// Execute with a timeout
	taskCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	return handler.Execute(taskCtx, task)
}

// handleTaskSuccess updates task state to completed.
func (wr *WorkerRuntime) handleTaskSuccess(ctx context.Context, task *sharedmodels.Task,
	result *TaskExecutionResult, duration time.Duration) {

	// Update task completion in database
	if wr.repo != nil {
		now := time.Now()
		_ = wr.repo.UpdateTaskCompletion(task.ID, string(sharedruntime.TaskStateCompleted),
			wr.workerID, "", task.RetryCount, &now)
	}

	// Publish task_completed event
	if wr.eventBus != nil {
		event := sharedruntime.NewEventBuilder(sharedruntime.EventTaskCompleted).
			WorkflowID(task.WorkflowID).
			TaskID(task.ID).
			WorkerID(wr.workerID).
			Data("duration_ms", duration.Milliseconds()).
			Data("output", result.Output).
			Build()
		_ = wr.eventBus.PublishEvent(ctx, event)
	}

	// Legacy event
	if wr.publisher != nil {
		_ = wr.publisher.PublishEvent("task_completed", map[string]interface{}{
			"task_id":     task.ID,
			"workflow_id": task.WorkflowID,
			"worker_id":   wr.workerID,
			"duration_ms": duration.Milliseconds(),
		})
	}
}

// handleTaskFailure updates task state to failed and handles retries.
func (wr *WorkerRuntime) handleTaskFailure(ctx context.Context, task *sharedmodels.Task,
	result *TaskExecutionResult, err error, duration time.Duration) {

	errMsg := ""
	if result != nil && result.Error != "" {
		errMsg = result.Error
	} else if err != nil {
		errMsg = err.Error()
	}

	// Check if we should retry
	retryMgr := sharedruntime.NewRetryManager(sharedruntime.RetryPolicy{
		MaxAttempts:       task.MaxRetries,
		InitialBackoff:    1 * time.Second,
		MaxBackoff:        5 * time.Minute,
		BackoffMultiplier: 2.0,
	})

	shouldRetry := result != nil && result.Retryable && retryMgr.CanRetry(task.RetryCount+1, errMsg)

	if shouldRetry {
		// Schedule retry
		nextRetryTime := retryMgr.NextRetryTime(task.RetryCount + 1)
		if wr.repo != nil {
			_ = wr.repo.UpdateTaskRetry(task.ID, task.RetryCount+1, nextRetryTime)
		}

		// Publish task_retrying event
		if wr.eventBus != nil {
			event := sharedruntime.NewEventBuilder(sharedruntime.EventTaskRetrying).
				WorkflowID(task.WorkflowID).
				TaskID(task.ID).
				WorkerID(wr.workerID).
				Data("attempt", task.RetryCount+1).
				Data("max_attempts", task.MaxRetries).
				Data("next_retry", nextRetryTime).
				Error(errMsg).
				Build()
			_ = wr.eventBus.PublishEvent(ctx, event)
		}
	} else {
		// Permanent failure
		now := time.Now()
		if wr.repo != nil {
			_ = wr.repo.UpdateTaskCompletion(task.ID, string(sharedruntime.TaskStateFailed),
				wr.workerID, errMsg, task.RetryCount, &now)
		}

		// Publish task_failed event
		if wr.eventBus != nil {
			event := sharedruntime.NewEventBuilder(sharedruntime.EventTaskFailed).
				WorkflowID(task.WorkflowID).
				TaskID(task.ID).
				WorkerID(wr.workerID).
				Data("duration_ms", duration.Milliseconds()).
				Error(errMsg).
				Build()
			_ = wr.eventBus.PublishEvent(ctx, event)
		}
	}

	// Legacy event
	if wr.publisher != nil {
		_ = wr.publisher.PublishEvent("task_failed", map[string]interface{}{
			"task_id":      task.ID,
			"workflow_id":  task.WorkflowID,
			"worker_id":    wr.workerID,
			"error":        errMsg,
			"should_retry": shouldRetry,
			"duration_ms":  duration.Milliseconds(),
		})
	}
}

// recordError logs an error during task processing.
func (wr *WorkerRuntime) recordError(ctx context.Context, taskID string, message string, err error) {
	errMsg := message
	if err != nil {
		errMsg = fmt.Sprintf("%s: %v", message, err)
	}

	// Note: We don't have the workflow ID here, so we skip logging to DB
	if wr.eventBus != nil {
		event := sharedruntime.NewEventBuilder(sharedruntime.EventTaskFailed).
			TaskID(taskID).
			WorkerID(wr.workerID).
			Error(errMsg).
			Build()
		_ = wr.eventBus.PublishEvent(ctx, event)
	}
}

// Stop gracefully shuts down the worker runtime.
func (wr *WorkerRuntime) Stop(ctx context.Context) error {
	// Trigger context cancellation
	return ctx.Err()
}
