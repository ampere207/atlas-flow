package runtime

import "time"

// WorkflowState represents lifecycle states for workflows.
type WorkflowState string

const (
	WorkflowStatePending   WorkflowState = "pending"
	WorkflowStateQueued    WorkflowState = "queued"
	WorkflowStateRunning   WorkflowState = "running"
	WorkflowStateRetrying  WorkflowState = "retrying"
	WorkflowStateCompleted WorkflowState = "completed"
	WorkflowStateFailed    WorkflowState = "failed"
	WorkflowStateCancelled WorkflowState = "cancelled"
)

// TaskState represents lifecycle states for tasks.
type TaskState string

const (
	TaskStatePending   TaskState = "pending"
	TaskStateAssigned  TaskState = "assigned"
	TaskStateRunning   TaskState = "running"
	TaskStateRetrying  TaskState = "retrying"
	TaskStateCompleted TaskState = "completed"
	TaskStateFailed    TaskState = "failed"
)

// RetryPolicy controls retry behavior for a task.
type RetryPolicy struct {
	MaxAttempts       int           `json:"max_attempts"`
	InitialBackoff    time.Duration `json:"initial_backoff"`
	MaxBackoff        time.Duration `json:"max_backoff"`
	BackoffMultiplier float64       `json:"backoff_multiplier"`
	RetryableStatus   []string      `json:"retryable_status,omitempty"`
}

// TaskDefinition describes a node in the workflow DAG.
type TaskDefinition struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Type           string                 `json:"type"`
	Payload        map[string]interface{} `json:"payload,omitempty"`
	DependsOn      []string               `json:"depends_on,omitempty"`
	RetryPolicy    RetryPolicy            `json:"retry_policy"`
	TimeoutSeconds int                    `json:"timeout_seconds,omitempty"`
}

// WorkflowDefinition describes an executable DAG.
type WorkflowDefinition struct {
	Tasks []TaskDefinition `json:"tasks"`
}

// Validate ensures the DAG is structurally valid.
func (definition WorkflowDefinition) Validate() error {
	seen := make(map[string]struct{}, len(definition.Tasks))
	for _, task := range definition.Tasks {
		if task.ID == "" {
			return ErrInvalidDefinition
		}
		if _, exists := seen[task.ID]; exists {
			return ErrDuplicateTaskID
		}
		seen[task.ID] = struct{}{}
	}
	for _, task := range definition.Tasks {
		for _, dependency := range task.DependsOn {
			if _, exists := seen[dependency]; !exists {
				return ErrUnknownDependency
			}
		}
	}
	return nil
}

// ReadyTasks returns tasks whose dependencies are all satisfied.
func (definition WorkflowDefinition) ReadyTasks(completed map[string]bool, states map[string]TaskState) []TaskDefinition {
	ready := make([]TaskDefinition, 0)
	for _, task := range definition.Tasks {
		if states[task.ID] == TaskStateCompleted {
			continue
		}
		if taskReady(task, completed, states) {
			ready = append(ready, task)
		}
	}
	return ready
}

func taskReady(task TaskDefinition, completed map[string]bool, states map[string]TaskState) bool {
	for _, dependency := range task.DependsOn {
		if !completed[dependency] || states[dependency] != TaskStateCompleted {
			return false
		}
	}
	return true
}

// TopologicalOrder returns an execution order for visualization.
func (definition WorkflowDefinition) TopologicalOrder() ([]string, error) {
	indegree := make(map[string]int, len(definition.Tasks))
	graph := make(map[string][]string, len(definition.Tasks))
	for _, task := range definition.Tasks {
		indegree[task.ID] = len(task.DependsOn)
		for _, dependency := range task.DependsOn {
			graph[dependency] = append(graph[dependency], task.ID)
		}
	}

	queue := make([]string, 0)
	for _, task := range definition.Tasks {
		if indegree[task.ID] == 0 {
			queue = append(queue, task.ID)
		}
	}

	order := make([]string, 0, len(definition.Tasks))
	for len(queue) > 0 {
		next := queue[0]
		queue = queue[1:]
		order = append(order, next)
		for _, dependent := range graph[next] {
			indegree[dependent]--
			if indegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	if len(order) != len(definition.Tasks) {
		return nil, ErrCyclicDefinition
	}

	return order, nil
}
