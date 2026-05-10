package repository

import (
	"database/sql"
	"encoding/json"
	"time"

	"atlasflow/backend/shared/models"
	"atlasflow/backend/shared/runtime"

	"github.com/google/uuid"
)

// WorkflowRepository handles workflow database operations
type WorkflowRepository struct {
	db *sql.DB
}

// NewWorkflowRepository creates a new workflow repository
func NewWorkflowRepository(db *sql.DB) *WorkflowRepository {
	return &WorkflowRepository{db: db}
}

// Create creates a new workflow
func (r *WorkflowRepository) Create(userID, name string, metadata map[string]interface{}) (*models.Workflow, error) {
	workflowID := uuid.New().String()
	now := time.Now()

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		metadataJSON = []byte("{}")
	}

	workflow := &models.Workflow{
		ID:         workflowID,
		UserID:     userID,
		Name:       name,
		Status:     "pending",
		Metadata:   string(metadataJSON),
		Definition: "",
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	query := `
		INSERT INTO workflows (id, user_id, name, status, metadata, definition, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err = r.db.Exec(query, workflow.ID, workflow.UserID, workflow.Name, workflow.Status, workflow.Metadata, workflow.Definition, workflow.CreatedAt, workflow.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return workflow, nil
}

// CreateExecutionWorkflow stores a workflow DAG definition and returns the persisted workflow.
func (r *WorkflowRepository) CreateExecutionWorkflow(userID, name string, metadata map[string]interface{}, definition runtime.WorkflowDefinition) (*models.Workflow, error) {
	definitionJSON, err := json.Marshal(definition)
	if err != nil {
		return nil, err
	}

	workflowID := uuid.New().String()
	now := time.Now()
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		metadataJSON = []byte("{}")
	}

	workflow := &models.Workflow{
		ID:         workflowID,
		UserID:     userID,
		Name:       name,
		Status:     string(runtime.WorkflowStatePending),
		Metadata:   string(metadataJSON),
		Definition: string(definitionJSON),
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	query := `
		INSERT INTO workflows (id, user_id, name, status, metadata, definition, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err = r.db.Exec(query, workflow.ID, workflow.UserID, workflow.Name, workflow.Status, workflow.Metadata, workflow.Definition, workflow.CreatedAt, workflow.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return workflow, nil
}

// AddTask persists a workflow task.
func (r *WorkflowRepository) AddTask(task *models.Task) error {
	query := `
		INSERT INTO tasks (
			id, workflow_id, task_type, name, payload, state, assigned_worker_id, retry_count, max_retries,
			depends_on, available_at, started_at, completed_at, failed_at, error_message, created_at, updated_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
	`

	assignedWorker := sql.NullString{String: task.AssignedWorkerID, Valid: task.AssignedWorkerID != ""}
	
	_, err := r.db.Exec(query,
		task.ID, task.WorkflowID, task.TaskType, task.Name, task.Payload, task.State, assignedWorker,
		task.RetryCount, task.MaxRetries, task.DependsOn, task.AvailableAt, task.StartedAt, task.CompletedAt,
		task.FailedAt, task.ErrorMessage, task.CreatedAt, task.UpdatedAt,
	)
	return err
}

// ListTasksByWorkflow returns tasks scoped to a user-owned workflow.
func (r *WorkflowRepository) ListTasksByWorkflow(workflowID, userID string) ([]*models.Task, error) {
	query := `
		SELECT t.id, t.workflow_id, t.task_type, t.name, t.payload, t.state, t.assigned_worker_id, t.retry_count,
		       t.max_retries, t.depends_on, t.available_at, t.started_at, t.completed_at, t.failed_at,
		       t.error_message, t.created_at, t.updated_at
		FROM tasks t
		INNER JOIN workflows w ON w.id = t.workflow_id
		WHERE t.workflow_id = $1 AND w.user_id = $2
		ORDER BY t.created_at ASC
	`

	rows, err := r.db.Query(query, workflowID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]*models.Task, 0)
	for rows.Next() {
		task := &models.Task{}
		var assignedWorker sql.NullString
		err := rows.Scan(
			&task.ID, &task.WorkflowID, &task.TaskType, &task.Name, &task.Payload, &task.State, &assignedWorker,
			&task.RetryCount, &task.MaxRetries, &task.DependsOn, &task.AvailableAt, &task.StartedAt, &task.CompletedAt,
			&task.FailedAt, &task.ErrorMessage, &task.CreatedAt, &task.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if assignedWorker.Valid {
			task.AssignedWorkerID = assignedWorker.String
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

// ListTransitionsByWorkflow returns workflow/task state transitions.
func (r *WorkflowRepository) ListTransitionsByWorkflow(workflowID, userID string) ([]*models.WorkflowTransition, error) {
	query := `
		SELECT wt.id, wt.workflow_id, wt.task_id, wt.entity_type, wt.from_state, wt.to_state, wt.reason, wt.created_at
		FROM workflow_transitions wt
		INNER JOIN workflows w ON w.id = wt.workflow_id
		WHERE wt.workflow_id = $1 AND w.user_id = $2
		ORDER BY wt.created_at ASC
	`

	rows, err := r.db.Query(query, workflowID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	transitions := make([]*models.WorkflowTransition, 0)
	for rows.Next() {
		transition := &models.WorkflowTransition{}
		var taskID sql.NullString
		err := rows.Scan(
			&transition.ID, &transition.WorkflowID, &taskID, &transition.EntityType, &transition.FromState,
			&transition.ToState, &transition.Reason, &transition.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if taskID.Valid {
			transition.TaskID = taskID.String
		}
		transitions = append(transitions, transition)
	}
	return transitions, nil
}

// AddTransition persists a workflow or task transition.
func (r *WorkflowRepository) AddTransition(transition *models.WorkflowTransition) error {
	query := `
		INSERT INTO workflow_transitions (id, workflow_id, task_id, entity_type, from_state, to_state, reason, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`

	taskID := sql.NullString{String: transition.TaskID, Valid: transition.TaskID != ""}

	_, err := r.db.Exec(query,
		transition.ID, transition.WorkflowID, taskID, transition.EntityType, transition.FromState,
		transition.ToState, transition.Reason, transition.CreatedAt,
	)
	return err
}

// UpdateWorkflowState updates a workflow state.
func (r *WorkflowRepository) UpdateWorkflowState(id, userID, state string) error {
	query := `
		UPDATE workflows
		SET status = $1, updated_at = $2
		WHERE id = $3 AND user_id = $4
	`
	_, err := r.db.Exec(query, state, time.Now(), id, userID)
	return err
}

// UpdateTaskState updates task execution fields.
func (r *WorkflowRepository) UpdateTaskState(taskID, state, workerID, errorMessage string, retryCount int, startedAt, completedAt, failedAt *time.Time, availableAt time.Time) error {
	query := `
		UPDATE tasks
		SET state = $1,
		    assigned_worker_id = $2,
		    error_message = $3,
		    retry_count = $4,
		    started_at = $5,
		    completed_at = $6,
		    failed_at = $7,
		    available_at = $8,
		    updated_at = $9
		WHERE id = $10
	`
	workerIDNull := sql.NullString{String: workerID, Valid: workerID != ""}

	_, err := r.db.Exec(query, state, workerIDNull, errorMessage, retryCount, startedAt, completedAt, failedAt, availableAt, time.Now(), taskID)
	return err
}

// UpdateTaskStateSimple is a simpler version that just updates the state field.
func (r *WorkflowRepository) UpdateTaskStateSimple(taskID, workflowID, userID, newState string) error {
	query := `
		UPDATE tasks
		SET state = $1, updated_at = $2
		WHERE id = $3
		AND workflow_id = (
			SELECT id FROM workflows WHERE id = $4 AND user_id = $5
		)
	`
	_, err := r.db.Exec(query, newState, time.Now(), taskID, workflowID, userID)
	return err
}

// QueueRootTasks moves dependency-free tasks into queued state.
func (r *WorkflowRepository) QueueRootTasks(workflowID string) ([]*models.Task, error) {
	tasks, err := r.ListTasksByWorkflowAll(workflowID)
	if err != nil {
		return nil, err
	}

	ready := make([]*models.Task, 0)
	for _, task := range tasks {
		if task.DependsOn == "[]" || task.DependsOn == "" {
			ready = append(ready, task)
		}
	}

	for _, task := range ready {
		_ = r.UpdateTaskState(task.ID, string(runtime.TaskStatePending), "", "", task.RetryCount, nil, nil, nil, time.Now())
	}

	return ready, nil
}

// ListTasksByWorkflowAll loads tasks without an ownership filter for internal scheduling.
func (r *WorkflowRepository) ListTasksByWorkflowAll(workflowID string) ([]*models.Task, error) {
	query := `
		SELECT t.id, t.workflow_id, t.task_type, t.name, t.payload, t.state, t.assigned_worker_id, t.retry_count,
		       t.max_retries, t.depends_on, t.available_at, t.started_at, t.completed_at, t.failed_at,
		       t.error_message, t.created_at, t.updated_at
		FROM tasks t
		INNER JOIN workflows w ON w.id = t.workflow_id
		WHERE t.workflow_id = $1
	`
	query += " ORDER BY t.created_at ASC"

	rows, err := r.db.Query(query, workflowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]*models.Task, 0)
	for rows.Next() {
		task := &models.Task{}
		var assignedWorker sql.NullString
		err := rows.Scan(
			&task.ID, &task.WorkflowID, &task.TaskType, &task.Name, &task.Payload, &task.State, &assignedWorker,
			&task.RetryCount, &task.MaxRetries, &task.DependsOn, &task.AvailableAt, &task.StartedAt, &task.CompletedAt,
			&task.FailedAt, &task.ErrorMessage, &task.CreatedAt, &task.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if assignedWorker.Valid {
			task.AssignedWorkerID = assignedWorker.String
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

// GetByID retrieves a workflow by ID
func (r *WorkflowRepository) GetByID(id, userID string) (*models.Workflow, error) {
	workflow := &models.Workflow{}

	query := `
		SELECT id, user_id, name, status, metadata, created_at, updated_at
		FROM workflows
		WHERE id = $1 AND user_id = $2
	`

	err := r.db.QueryRow(query, id, userID).Scan(
		&workflow.ID, &workflow.UserID, &workflow.Name, &workflow.Status, &workflow.Metadata, &workflow.CreatedAt, &workflow.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return workflow, nil
}

// GetExecutionWorkflow retrieves a workflow including its persisted DAG definition.
func (r *WorkflowRepository) GetExecutionWorkflow(id, userID string) (*models.Workflow, error) {
	workflow := &models.Workflow{}

	query := `
		SELECT id, user_id, name, status, metadata, definition, created_at, updated_at
		FROM workflows
		WHERE id = $1 AND user_id = $2
	`

	err := r.db.QueryRow(query, id, userID).Scan(
		&workflow.ID, &workflow.UserID, &workflow.Name, &workflow.Status, &workflow.Metadata, &workflow.Definition, &workflow.CreatedAt, &workflow.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return workflow, nil
}

// ListByUserID retrieves all workflows for a user
func (r *WorkflowRepository) ListByUserID(userID string, limit, offset int) ([]*models.Workflow, error) {
	query := `
		SELECT id, user_id, name, status, metadata, created_at, updated_at
		FROM workflows
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workflows []*models.Workflow
	for rows.Next() {
		workflow := &models.Workflow{}
		err := rows.Scan(
			&workflow.ID, &workflow.UserID, &workflow.Name, &workflow.Status, &workflow.Metadata, &workflow.CreatedAt, &workflow.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		workflows = append(workflows, workflow)
	}

	return workflows, nil
}

// ListAllRunningWorkflows returns all workflows in running state (for orchestration).
func (r *WorkflowRepository) ListAllRunningWorkflows() ([]interface{}, error) {
	query := `
		SELECT id, user_id, name, status, created_at, updated_at
		FROM workflows
		WHERE status = $1
		ORDER BY updated_at ASC
	`

	rows, err := r.db.Query(query, string(runtime.WorkflowStateRunning))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workflows []interface{}
	for rows.Next() {
		var id, userID, name, status string
		var createdAt, updatedAt time.Time
		err := rows.Scan(&id, &userID, &name, &status, &createdAt, &updatedAt)
		if err != nil {
			continue
		}
		workflows = append(workflows, struct {
			ID     string
			UserID string
			Status string
		}{ID: id, UserID: userID, Status: status})
	}

	return workflows, nil
}

// UpdateStatus updates workflow status
func (r *WorkflowRepository) UpdateStatus(id, userID, status string) error {
	query := `
		UPDATE workflows
		SET status = $1, updated_at = $2
		WHERE id = $3 AND user_id = $4
	`

	_, err := r.db.Exec(query, status, time.Now(), id, userID)
	return err
}

// CreateEvent creates a workflow event
func (r *WorkflowRepository) CreateEvent(workflowID, eventType string, payload map[string]interface{}) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		payloadJSON = []byte("{}")
	}

	query := `
		INSERT INTO workflow_events (id, workflow_id, event_type, payload, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err = r.db.Exec(query, uuid.New().String(), workflowID, eventType, string(payloadJSON), time.Now())
	return err
}
