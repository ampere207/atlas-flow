package runtime

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// TaskScheduler determines which tasks are ready to execute based on DAG dependencies.
type TaskScheduler struct {
	definition WorkflowDefinition
}

// NewTaskScheduler creates a new scheduler for a workflow definition.
func NewTaskScheduler(definition WorkflowDefinition) (*TaskScheduler, error) {
	if err := definition.Validate(); err != nil {
		return nil, err
	}
	return &TaskScheduler{definition: definition}, nil
}

// ScheduleableTask represents a task ready to be scheduled.
type ScheduleableTask struct {
	TaskID         string
	TaskType       string
	Name           string
	Payload        map[string]interface{}
	RetryPolicy    RetryPolicy
	TimeoutSeconds int
}

// DetermineReadyTasks identifies tasks whose dependencies are satisfied.
// This is the core scheduling logic for DAG execution.
func (s *TaskScheduler) DetermineReadyTasks(ctx context.Context, taskStates map[string]TaskState) ([]ScheduleableTask, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	readyTasks := make([]ScheduleableTask, 0)

	// For each task in the definition, check if all dependencies are complete
	for _, taskDef := range s.definition.Tasks {
		currentState := taskStates[taskDef.ID]

		// Skip if already completed or running
		if currentState == TaskStateCompleted || currentState == TaskStateRunning || currentState == TaskStateAssigned {
			continue
		}

		// Check if all dependencies are completed
		allDependenciesMet := true
		for _, depID := range taskDef.DependsOn {
			depState := taskStates[depID]
			if depState != TaskStateCompleted {
				allDependenciesMet = false
				break
			}
		}

		if allDependenciesMet {
			readyTasks = append(readyTasks, ScheduleableTask{
				TaskID:         taskDef.ID,
				TaskType:       taskDef.Type,
				Name:           taskDef.Name,
				Payload:        taskDef.Payload,
				RetryPolicy:    taskDef.RetryPolicy,
				TimeoutSeconds: taskDef.TimeoutSeconds,
			})
		}
	}

	return readyTasks, nil
}

// GetTaskDefinition retrieves a specific task definition by ID.
func (s *TaskScheduler) GetTaskDefinition(taskID string) (*TaskDefinition, error) {
	for i, task := range s.definition.Tasks {
		if task.ID == taskID {
			return &s.definition.Tasks[i], nil
		}
	}
	return nil, fmt.Errorf("task definition not found: %s", taskID)
}

// GetDependencies returns all task IDs that must complete before the given task.
func (s *TaskScheduler) GetDependencies(taskID string) ([]string, error) {
	for _, task := range s.definition.Tasks {
		if task.ID == taskID {
			return task.DependsOn, nil
		}
	}
	return nil, fmt.Errorf("task not found: %s", taskID)
}

// GetDependents returns all task IDs that depend on the given task.
func (s *TaskScheduler) GetDependents(taskID string) []string {
	dependents := make([]string, 0)
	for _, task := range s.definition.Tasks {
		for _, dep := range task.DependsOn {
			if dep == taskID {
				dependents = append(dependents, task.ID)
				break
			}
		}
	}
	return dependents
}

// ExecutionProgress tracks the current state of all tasks in a workflow.
type ExecutionProgress struct {
	WorkflowID  string
	TaskStates  map[string]TaskState
	LastUpdated time.Time
	CompletedAt map[string]time.Time
	StartedAt   map[string]time.Time
	FailedAt    map[string]time.Time
	RetryCount  map[string]int
}

// NewExecutionProgress initializes tracking for a workflow.
func NewExecutionProgress(workflowID string, taskCount int) *ExecutionProgress {
	return &ExecutionProgress{
		WorkflowID:  workflowID,
		TaskStates:  make(map[string]TaskState, taskCount),
		CompletedAt: make(map[string]time.Time, taskCount),
		StartedAt:   make(map[string]time.Time, taskCount),
		FailedAt:    make(map[string]time.Time, taskCount),
		RetryCount:  make(map[string]int, taskCount),
		LastUpdated: time.Now(),
	}
}

// UpdateTaskState records a state transition for a task.
func (progress *ExecutionProgress) UpdateTaskState(taskID string, newState TaskState) error {
	if taskID == "" {
		return errors.New("task_id cannot be empty")
	}

	progress.TaskStates[taskID] = newState
	progress.LastUpdated = time.Now()

	// Record timestamps for key state transitions
	switch newState {
	case TaskStateRunning:
		if progress.StartedAt[taskID].IsZero() {
			progress.StartedAt[taskID] = time.Now()
		}
	case TaskStateCompleted:
		progress.CompletedAt[taskID] = time.Now()
	case TaskStateFailed:
		progress.FailedAt[taskID] = time.Now()
	}

	return nil
}

// IsWorkflowComplete checks if all tasks are completed or failed.
func (progress *ExecutionProgress) IsWorkflowComplete(scheduler *TaskScheduler) bool {
	for _, task := range scheduler.definition.Tasks {
		state := progress.TaskStates[task.ID]
		if state != TaskStateCompleted && state != TaskStateFailed {
			return false
		}
	}
	return true
}

// GetReadyTasks convenience method wrapping the scheduler.
func (progress *ExecutionProgress) GetReadyTasks(scheduler *TaskScheduler) ([]ScheduleableTask, error) {
	return scheduler.DetermineReadyTasks(context.Background(), progress.TaskStates)
}
