package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"atlasflow/backend/shared/models"
	sharedruntime "atlasflow/backend/shared/runtime"
	"atlasflow/backend/workflow-service/internal/repository"
)

// FailureRecoveryManager alias for shared runtime
type FailureRecoveryManager = sharedruntime.FailureRecoveryManager

// EventPublisher publishes orchestration events to NATS or another bus.
type EventPublisher interface {
	PublishEvent(subject string, event map[string]interface{}) error
}

// ExecutionEngine is the Phase 2 core orchestrator.
// It coordinates workflow DAG execution, task scheduling, worker dispatch, retries, and recovery.
type ExecutionEngine struct {
	repo            *repository.WorkflowRepository
	publisher       EventPublisher
	dispatcher      sharedruntime.TaskDispatcher
	scheduler       *sharedruntime.TaskScheduler
	retryManager    *sharedruntime.RetryManager
	recoveryManager *FailureRecoveryManager
	eventBus        sharedruntime.EventPublisher
}

// NewExecutionEngine creates the Phase 2 execution engine.
func NewExecutionEngine(repo *repository.WorkflowRepository, publisher EventPublisher,
	dispatcher sharedruntime.TaskDispatcher, eventBus sharedruntime.EventPublisher) *ExecutionEngine {
	return &ExecutionEngine{
		repo:       repo,
		publisher:  publisher,
		dispatcher: dispatcher,
		retryManager: sharedruntime.NewRetryManager(sharedruntime.RetryPolicy{
			MaxAttempts:       3,
			InitialBackoff:    1 * time.Second,
			MaxBackoff:        5 * time.Minute,
			BackoffMultiplier: 2.0,
		}),
		recoveryManager: sharedruntime.NewFailureRecoveryManager(10*time.Minute, 30*time.Second),
		eventBus:        eventBus,
	}
}

// Orchestrator (legacy) for backward compatibility - use ExecutionEngine instead
type Orchestrator struct {
	engine *ExecutionEngine
}

// NewOrchestrator creates a workflow orchestrator.
func NewOrchestrator(repo *repository.WorkflowRepository, publisher EventPublisher) *Orchestrator {
	return &Orchestrator{
		engine: NewExecutionEngine(repo, publisher, sharedruntime.NewInMemoryTaskDispatcher(), sharedruntime.NewInMemoryEventBus()),
	}
}

// ScheduleWorkflowTasks determines which tasks are ready to execute and makes them available.
// This is the PRIMARY scheduling loop for Phase 2.
func (o *Orchestrator) ScheduleWorkflowTasks(ctx context.Context, workflowID, userID string) error {
	workflow, err := o.engine.repo.GetExecutionWorkflow(workflowID, userID)
	if err != nil {
		return err
	}

	// Parse the workflow definition
	var definition sharedruntime.WorkflowDefinition
	if err := json.Unmarshal([]byte(workflow.Definition), &definition); err != nil {
		return err
	}
	if err := definition.Validate(); err != nil {
		return err
	}

	// Create a scheduler for this workflow
	scheduler, err := sharedruntime.NewTaskScheduler(definition)
	if err != nil {
		return err
	}

	// Get all tasks for this workflow
	tasks, err := o.engine.repo.ListTasksByWorkflow(workflowID, userID)
	if err != nil {
		return err
	}

	// Build current task state map
	taskStates := make(map[string]sharedruntime.TaskState)
	taskMap := make(map[string]*models.Task)
	for _, task := range tasks {
		taskStates[task.ID] = sharedruntime.TaskState(task.State)
		taskMap[task.ID] = task
	}

	// Determine which tasks are ready to execute
	readyTasks, err := scheduler.DetermineReadyTasks(ctx, taskStates)
	if err != nil {
		return err
	}

	// Dispatch ready tasks to workers
	for _, readyTask := range readyTasks {
		task := taskMap[readyTask.TaskID]
		if task == nil {
			continue
		}

		// Calculate availability time (for retries, this would be delayed)
		availableAt := time.Now()
		if task.RetryCount > 0 {
			// Apply retry backoff
			retryMgr := sharedruntime.NewRetryManager(readyTask.RetryPolicy)
			availableAt = retryMgr.NextRetryTime(task.RetryCount + 1)
		}

		// Dispatch task to the queue
		if err := o.engine.dispatcher.DispatchTask(ctx, task.ID, nil, 24*time.Hour); err != nil {
			_ = err // Log but continue
		}

		// Mark task as scheduled
		if err := o.engine.repo.UpdateTaskStateSimple(task.ID, workflowID, userID, string(sharedruntime.TaskStatePending)); err != nil {
			_ = err
		}

		// Publish event
		if o.engine.eventBus != nil {
			event := sharedruntime.NewEventBuilder(sharedruntime.EventTaskScheduled).
				WorkflowID(workflowID).
				TaskID(task.ID).
				UserID(userID).
				Data("task_name", task.Name).
				Data("available_at", availableAt).
				Build()
			_ = o.engine.eventBus.PublishEvent(ctx, event)
		}
	}

	return nil
}

// CheckAndExecuteRetries checks for failed tasks that should be retried.
func (o *Orchestrator) CheckAndExecuteRetries(ctx context.Context, workflowID, userID string) error {
	tasks, err := o.engine.repo.ListTasksByWorkflow(workflowID, userID)
	if err != nil {
		return err
	}

	now := time.Now()

	for _, task := range tasks {
		if sharedruntime.TaskState(task.State) != sharedruntime.TaskStateFailed {
			continue
		}

		// Check if we should retry
		retryMgr := sharedruntime.NewRetryManager(sharedruntime.RetryPolicy{
			MaxAttempts:       task.MaxRetries,
			InitialBackoff:    1 * time.Second,
			MaxBackoff:        5 * time.Minute,
			BackoffMultiplier: 2.0,
		})

		decision := retryMgr.MakeRetryDecision(task.RetryCount, task.ErrorMessage, task.MaxRetries)
		if !decision.ShouldRetry {
			continue
		}

		// Check if it's time to retry
		if now.Before(decision.NextAttemptTime) {
			continue
		}

		// Schedule retry
		if err := o.engine.dispatcher.DispatchTask(ctx, task.ID, nil, 24*time.Hour); err != nil {
			_ = err
		}

		// Update task state
		if err := o.engine.repo.UpdateTaskStateSimple(task.ID, workflowID, userID, string(sharedruntime.TaskStateRetrying)); err != nil {
			_ = err
		}

		// Publish retry event
		if o.engine.eventBus != nil {
			event := sharedruntime.NewEventBuilder(sharedruntime.EventTaskRetrying).
				WorkflowID(workflowID).
				TaskID(task.ID).
				UserID(userID).
				Data("attempt", task.RetryCount+1).
				Data("max_attempts", task.MaxRetries).
				Build()
			_ = o.engine.eventBus.PublishEvent(ctx, event)
		}
	}

	return nil
}

// DetectAndRecoverFailures detects stalled/orphaned tasks and takes recovery actions.
func (o *Orchestrator) DetectAndRecoverFailures(ctx context.Context, workflowID, userID string) error {
	tasks, err := o.engine.repo.ListTasksByWorkflow(workflowID, userID)
	if err != nil {
		return err
	}

	// Build task assignment map
	assignments := make(map[string]sharedruntime.TaskAssignment)
	for _, task := range tasks {
		startedAt := time.Time{}
		if task.StartedAt != nil {
			startedAt = *task.StartedAt
		}
		assignments[task.ID] = sharedruntime.TaskAssignment{
			TaskID:           task.ID,
			WorkflowID:       workflowID,
			AssignedWorkerID: task.AssignedWorkerID,
			State:            sharedruntime.TaskState(task.State),
			AssignedAt:       task.CreatedAt,
			StartedAt:        startedAt,
			UpdatedAt:        task.UpdatedAt,
			Attempt:          task.RetryCount,
		}
	}

	// Detect stalled tasks
	stalledTasks, err := o.engine.recoveryManager.DetectStalledTasks(ctx, assignments)
	if err != nil {
		return err
	}

	// Recover stalled tasks
	for _, stalled := range stalledTasks {
		// Mark as failed so it can be retried
		if err := o.engine.repo.UpdateTaskStateSimple(stalled.TaskID, workflowID, userID, string(sharedruntime.TaskStateFailed)); err != nil {
			_ = err
		}

		// Publish recovery event
		if o.engine.eventBus != nil {
			event := sharedruntime.NewEventBuilder(sharedruntime.ExecutionEventRecoveryStarted).
				WorkflowID(workflowID).
				TaskID(stalled.TaskID).
				UserID(userID).
				Data("reason", stalled.ReasonForRecovery).
				Build()
			_ = o.engine.eventBus.PublishEvent(ctx, event)
		}
	}

	return nil
}

// TransitionWorkflowState transitions workflow to completed/failed if all tasks done.
func (o *Orchestrator) TransitionWorkflowState(ctx context.Context, workflowID, userID string) error {
	workflow, err := o.engine.repo.GetExecutionWorkflow(workflowID, userID)
	if err != nil {
		return err
	}

	tasks, err := o.engine.repo.ListTasksByWorkflow(workflowID, userID)
	if err != nil {
		return err
	}

	// Count task states
	completed := 0
	failed := 0
	total := len(tasks)

	for _, task := range tasks {
		switch sharedruntime.TaskState(task.State) {
		case sharedruntime.TaskStateCompleted:
			completed++
		case sharedruntime.TaskStateFailed:
			failed++
		}
	}

	// All tasks done?
	if completed+failed != total {
		return nil // Still running
	}

	// Determine workflow final state
	var newState sharedruntime.WorkflowState
	reason := ""
	if failed == 0 {
		newState = sharedruntime.WorkflowStateCompleted
		reason = "all tasks completed successfully"
	} else {
		newState = sharedruntime.WorkflowStateFailed
		reason = fmt.Sprintf("%d tasks failed", failed)
	}

	// Update workflow state
	if err := o.engine.repo.UpdateWorkflowState(workflowID, userID, string(newState)); err != nil {
		return err
	}

	// Record transition
	_ = o.engine.repo.AddTransition(&models.WorkflowTransition{
		ID:         workflowID + "-final-" + time.Now().UTC().Format(time.RFC3339Nano),
		WorkflowID: workflowID,
		EntityType: "workflow",
		FromState:  workflow.Status,
		ToState:    string(newState),
		Reason:     reason,
		CreatedAt:  time.Now().UTC(),
	})

	// Publish event
	if o.engine.eventBus != nil {
		eventType := sharedruntime.EventWorkflowCompleted
		if failed > 0 {
			eventType = sharedruntime.EventWorkflowFailed
		}
		event := sharedruntime.NewEventBuilder(eventType).
			WorkflowID(workflowID).
			UserID(userID).
			Data("task_completed", completed).
			Data("task_failed", failed).
			Build()
		_ = o.engine.eventBus.PublishEvent(ctx, event)
	}

	return nil
}

// StartWorkflow transitions a workflow into the running state.
func (o *Orchestrator) StartWorkflow(ctx context.Context, workflowID, userID string) (*models.Workflow, error) {
	workflow, err := o.engine.repo.GetExecutionWorkflow(workflowID, userID)
	if err != nil {
		return nil, err
	}

	var definition sharedruntime.WorkflowDefinition
	if workflow.Definition != "" {
		if err := json.Unmarshal([]byte(workflow.Definition), &definition); err != nil {
			return nil, err
		}
		if err := definition.Validate(); err != nil {
			return nil, err
		}
	}

	previousState := workflow.Status
	if err := o.engine.repo.UpdateWorkflowState(workflowID, userID, string(sharedruntime.WorkflowStateRunning)); err != nil {
		return nil, err
	}

	transition := &models.WorkflowTransition{
		ID:         workflowID + "-started-" + time.Now().UTC().Format(time.RFC3339Nano),
		WorkflowID: workflowID,
		EntityType: "workflow",
		FromState:  previousState,
		ToState:    string(sharedruntime.WorkflowStateRunning),
		Reason:     "workflow execution started",
		CreatedAt:  time.Now().UTC(),
	}
	_ = o.engine.repo.AddTransition(transition)

	if o.engine.publisher != nil {
		_ = o.engine.publisher.PublishEvent("workflow_started", map[string]interface{}{
			"workflow_id": workflowID,
			"user_id":     userID,
			"status":      string(sharedruntime.WorkflowStateRunning),
		})
	}

	return o.engine.repo.GetExecutionWorkflow(workflowID, userID)
}

// CancelWorkflow transitions a workflow to cancelled.
func (o *Orchestrator) CancelWorkflow(ctx context.Context, workflowID, userID string) error {
	workflow, err := o.engine.repo.GetExecutionWorkflow(workflowID, userID)
	if err != nil {
		return err
	}

	if err := o.engine.repo.UpdateWorkflowState(workflowID, userID, string(sharedruntime.WorkflowStateCancelled)); err != nil {
		return err
	}

	_ = o.engine.repo.AddTransition(&models.WorkflowTransition{
		ID:         workflowID + "-cancelled-" + time.Now().UTC().Format(time.RFC3339Nano),
		WorkflowID: workflowID,
		EntityType: "workflow",
		FromState:  workflow.Status,
		ToState:    string(sharedruntime.WorkflowStateCancelled),
		Reason:     "workflow cancelled by user",
		CreatedAt:  time.Now().UTC(),
	})

	if o.engine.publisher != nil {
		_ = o.engine.publisher.PublishEvent("workflow_cancelled", map[string]interface{}{
			"workflow_id": workflowID,
			"user_id":     userID,
		})
	}

	return nil
}

// ListTasks returns all execution tasks for a workflow.
func (o *Orchestrator) ListTasks(workflowID, userID string) ([]*models.Task, error) {
	return o.engine.repo.ListTasksByWorkflow(workflowID, userID)
}

// ListHistory returns transition history for a workflow.
func (o *Orchestrator) ListHistory(workflowID, userID string) ([]*models.WorkflowTransition, error) {
	return o.engine.repo.ListTransitionsByWorkflow(workflowID, userID)
}

// GetStatus returns the workflow snapshot.
func (o *Orchestrator) GetStatus(workflowID, userID string) (*models.Workflow, error) {
	return o.engine.repo.GetExecutionWorkflow(workflowID, userID)
}

// StartWorkflowIfDefinitionMissing keeps the API predictable for legacy workflows.
func (o *Orchestrator) StartWorkflowIfDefinitionMissing(ctx context.Context, workflowID, userID string) (*models.Workflow, error) {
	workflow, err := o.engine.repo.GetExecutionWorkflow(workflowID, userID)
	if err != nil {
		return nil, err
	}
	if workflow.Definition == "" {
		return nil, errors.New("workflow definition is missing")
	}
	return o.StartWorkflow(ctx, workflowID, userID)
}
