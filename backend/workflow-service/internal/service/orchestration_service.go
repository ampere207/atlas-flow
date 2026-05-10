package service

import (
	"context"
	"fmt"
	"log"
	"time"

	sharedruntime "atlasflow/backend/shared/runtime"
	"atlasflow/backend/workflow-service/internal/repository"
	"atlasflow/backend/workflow-service/internal/runtime"
)

// OrchestrationService is the main Phase 2 orchestration loop.
// It coordinates all phases of workflow execution:
// 1. Schedule: Determine which tasks are ready
// 2. Dispatch: Make tasks available to workers
// 3. Monitor: Track worker health and task progress
// 4. Recover: Detect and fix failures
// 5. Complete: Transition workflows to final states
type OrchestrationService struct {
	repo         *repository.WorkflowRepository
	eventBus     sharedruntime.EventPublisher
	orchestrator *runtime.Orchestrator
	runInterval  time.Duration
	stopChan     chan struct{}
}

// NewOrchestrationService creates the Phase 2 orchestration service.
func NewOrchestrationService(repo *repository.WorkflowRepository,
	eventBus sharedruntime.EventPublisher,
	orchestrator *runtime.Orchestrator) *OrchestrationService {
	return &OrchestrationService{
		repo:         repo,
		eventBus:     eventBus,
		orchestrator: orchestrator,
		runInterval:  2 * time.Second, // Run orchestration loop every 2 seconds
		stopChan:     make(chan struct{}),
	}
}

// Start begins the orchestration loop.
// This runs continuously in a goroutine.
func (os *OrchestrationService) Start(ctx context.Context) error {
	ticker := time.NewTicker(os.runInterval)
	defer ticker.Stop()

	log.Println("Orchestration Service started")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-os.stopChan:
			return nil
		case <-ticker.C:
			os.orchestrationCycle(ctx)
		}
	}
}

// Stop gracefully shuts down the orchestration service.
func (os *OrchestrationService) Stop() {
	close(os.stopChan)
}

// orchestrationCycle runs a single orchestration cycle.
// This is the heartbeat of the system.
func (os *OrchestrationService) orchestrationCycle(ctx context.Context) {
	// 1. Get all running workflows
	workflows, err := os.repo.ListAllRunningWorkflows()
	if err != nil {
		log.Printf("Error fetching workflows: %v", err)
		return
	}

	for _, workflow := range workflows {
		// Process each running workflow
		os.processWorkflow(ctx, workflow)
	}

	// 2. Perform global recovery checks
	os.performGlobalRecovery(ctx)
}

// processWorkflow handles all orchestration phases for a single workflow.
func (os *OrchestrationService) processWorkflow(ctx context.Context, workflow interface{}) {
	// Cast workflow to appropriate type
	wf := workflow.(*struct {
		ID     string
		UserID string
		Status string
	})

	// Phase 1: Schedule - Determine which tasks can run
	if err := os.orchestrator.ScheduleWorkflowTasks(ctx, wf.ID, wf.UserID); err != nil {
		log.Printf("Error scheduling workflow %s: %v", wf.ID, err)
	}

	// Phase 2: Check retries - Schedule failed tasks for retry
	if err := os.orchestrator.CheckAndExecuteRetries(ctx, wf.ID, wf.UserID); err != nil {
		log.Printf("Error checking retries for workflow %s: %v", wf.ID, err)
	}

	// Phase 3: Monitor and Recover - Detect stalled/orphaned tasks
	if err := os.orchestrator.DetectAndRecoverFailures(ctx, wf.ID, wf.UserID); err != nil {
		log.Printf("Error detecting failures for workflow %s: %v", wf.ID, err)
	}

	// Phase 4: State Transition - Check if workflow is complete
	if err := os.orchestrator.TransitionWorkflowState(ctx, wf.ID, wf.UserID); err != nil {
		log.Printf("Error transitioning workflow state %s: %v", wf.ID, err)
	}
}

// performGlobalRecovery runs system-wide failure detection.
func (os *OrchestrationService) performGlobalRecovery(ctx context.Context) {
	// TODO: Implement global recovery logic
	// - Detect workers that haven't heartbeated
	// - Reclaim tasks from dead workers
	// - Clean up stale leases
}

// GetOrchestrationStats returns current orchestration statistics.
type OrchestrationStats struct {
	TotalWorkflows       int64     `json:"total_workflows"`
	RunningWorkflows     int64     `json:"running_workflows"`
	CompletedWorkflows   int64     `json:"completed_workflows"`
	FailedWorkflows      int64     `json:"failed_workflows"`
	TotalTasks           int64     `json:"total_tasks"`
	CompletedTasks       int64     `json:"completed_tasks"`
	FailedTasks          int64     `json:"failed_tasks"`
	RetryingTasks        int64     `json:"retrying_tasks"`
	ActiveWorkers        int64     `json:"active_workers"`
	LastOrchestrationRun time.Time `json:"last_orchestration_run"`
}

// GetStats returns orchestration statistics.
func (os *OrchestrationService) GetStats(ctx context.Context) (*OrchestrationStats, error) {
	// TODO: Implement stats gathering
	return &OrchestrationStats{}, nil
}

// HealthCheck returns the health status of the orchestration service.
type HealthStatus struct {
	Status    string        `json:"status"` // "healthy", "degraded", "unhealthy"
	Uptime    time.Duration `json:"uptime"`
	LastCheck time.Time     `json:"last_check"`
	Details   string        `json:"details"`
}

// Health returns the service health status.
func (os *OrchestrationService) Health() HealthStatus {
	return HealthStatus{
		Status:    "healthy",
		LastCheck: time.Now(),
		Details:   "Orchestration service running normally",
	}
}

// WorkflowExecutor provides a high-level API for executing workflows.
type WorkflowExecutor struct {
	orchestrator *runtime.Orchestrator
	eventBus     sharedruntime.EventPublisher
}

// NewWorkflowExecutor creates a workflow executor.
func NewWorkflowExecutor(orchestrator *runtime.Orchestrator, eventBus sharedruntime.EventPublisher) *WorkflowExecutor {
	return &WorkflowExecutor{
		orchestrator: orchestrator,
		eventBus:     eventBus,
	}
}

// Execute starts a workflow execution.
func (we *WorkflowExecutor) Execute(ctx context.Context, workflowID, userID string) error {
	// Start the workflow
	workflow, err := we.orchestrator.StartWorkflow(ctx, workflowID, userID)
	if err != nil {
		return fmt.Errorf("failed to start workflow: %w", err)
	}

	// Publish execution started event
	if we.eventBus != nil {
		event := sharedruntime.NewEventBuilder(sharedruntime.EventWorkflowStarted).
			WorkflowID(workflowID).
			UserID(userID).
			Data("workflow_status", workflow.Status).
			Build()
		_ = we.eventBus.PublishEvent(ctx, event)
	}

	return nil
}

// Cancel cancels a workflow execution.
func (we *WorkflowExecutor) Cancel(ctx context.Context, workflowID, userID string) error {
	if err := we.orchestrator.CancelWorkflow(ctx, workflowID, userID); err != nil {
		return fmt.Errorf("failed to cancel workflow: %w", err)
	}

	// Publish cancellation event
	if we.eventBus != nil {
		event := sharedruntime.NewEventBuilder(sharedruntime.EventWorkflowCancelled).
			WorkflowID(workflowID).
			UserID(userID).
			Build()
		_ = we.eventBus.PublishEvent(ctx, event)
	}

	return nil
}

// GetStatus returns the current status of a workflow execution.
func (we *WorkflowExecutor) GetStatus(ctx context.Context, workflowID, userID string) (*ExecutionStatus, error) {
	workflow, err := we.orchestrator.GetStatus(workflowID, userID)
	if err != nil {
		return nil, err
	}

	tasks, err := we.orchestrator.ListTasks(workflowID, userID)
	if err != nil {
		return nil, err
	}

	// Count task states
	var completed, failed, running int
	for _, task := range tasks {
		switch sharedruntime.TaskState(task.State) {
		case sharedruntime.TaskStateCompleted:
			completed++
		case sharedruntime.TaskStateFailed:
			failed++
		case sharedruntime.TaskStateRunning, sharedruntime.TaskStateAssigned:
			running++
		}
	}

	return &ExecutionStatus{
		WorkflowID:      workflow.ID,
		Status:          workflow.Status,
		CreatedAt:       workflow.CreatedAt,
		UpdatedAt:       workflow.UpdatedAt,
		TotalTasks:      len(tasks),
		CompletedTasks:  completed,
		FailedTasks:     failed,
		RunningTasks:    running,
		PercentComplete: (completed * 100) / len(tasks),
	}, nil
}

// ExecutionStatus represents the current state of a workflow execution.
type ExecutionStatus struct {
	WorkflowID      string    `json:"workflow_id"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	TotalTasks      int       `json:"total_tasks"`
	CompletedTasks  int       `json:"completed_tasks"`
	FailedTasks     int       `json:"failed_tasks"`
	RunningTasks    int       `json:"running_tasks"`
	PercentComplete int       `json:"percent_complete"`
}
