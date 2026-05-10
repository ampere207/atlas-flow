package repository

import (
	"database/sql"
	"encoding/json"
	"time"

	"atlasflow/backend/shared/models"
	"atlasflow/backend/shared/runtime"

	"github.com/google/uuid"
)

// WorkerRepository handles worker database operations
type WorkerRepository struct {
	db *sql.DB
}

// NewWorkerRepository creates a new worker repository
func NewWorkerRepository(db *sql.DB) *WorkerRepository {
	return &WorkerRepository{db: db}
}

// Create creates a new worker
func (r *WorkerRepository) Create(userID, name string) (*models.Worker, error) {
	workerID := uuid.New().String()
	now := time.Now()

	worker := &models.Worker{
		ID:            workerID,
		UserID:        userID,
		Name:          name,
		Status:        "idle",
		LastHeartbeat: now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	query := `
		INSERT INTO workers (id, user_id, name, status, last_heartbeat, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.Exec(query, worker.ID, worker.UserID, worker.Name, worker.Status, worker.LastHeartbeat, worker.CreatedAt, worker.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return worker, nil
}

// GetByID retrieves a worker by ID
func (r *WorkerRepository) GetByID(id, userID string) (*models.Worker, error) {
	worker := &models.Worker{}

	query := `
		SELECT id, user_id, name, status, last_heartbeat, created_at, updated_at
		FROM workers
		WHERE id = $1 AND user_id = $2
	`

	err := r.db.QueryRow(query, id, userID).Scan(
		&worker.ID, &worker.UserID, &worker.Name, &worker.Status, &worker.LastHeartbeat, &worker.CreatedAt, &worker.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return worker, nil
}

// ListByUserID retrieves all workers for a user
func (r *WorkerRepository) ListByUserID(userID string, limit, offset int) ([]*models.Worker, error) {
	query := `
		SELECT id, user_id, name, status, last_heartbeat, created_at, updated_at
		FROM workers
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workers []*models.Worker
	for rows.Next() {
		worker := &models.Worker{}
		err := rows.Scan(
			&worker.ID, &worker.UserID, &worker.Name, &worker.Status, &worker.LastHeartbeat, &worker.CreatedAt, &worker.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		workers = append(workers, worker)
	}

	return workers, nil
}

// UpdateHeartbeat updates worker's last heartbeat
func (r *WorkerRepository) UpdateHeartbeat(id, userID, status string) error {
	now := time.Now()
	_, err := r.db.Exec(`
		UPDATE workers
		SET last_heartbeat = $1, status = $2, updated_at = $3
		WHERE id = $4 AND user_id = $5
	`, now, status, now, id, userID)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(`
		INSERT INTO worker_heartbeats (id, worker_id, user_id, status, recorded_at)
		VALUES ($1, $2, $3, $4, $5)
	`, uuid.New().String(), id, userID, status, now)
	return err
}

// ClaimNextTask atomically claims the next runnable task.
func (r *WorkerRepository) ClaimNextTask(workerID string) (*models.Task, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := `
		SELECT t.id, t.workflow_id, t.task_type, t.name, t.payload, t.state, t.assigned_worker_id, t.retry_count,
		       t.max_retries, t.depends_on, t.available_at, t.started_at, t.completed_at, t.failed_at,
		       t.error_message, t.created_at, t.updated_at
		FROM tasks t
		WHERE t.state IN ($1, $2)
		  AND t.available_at <= NOW()
		  AND NOT EXISTS (
			SELECT 1
			FROM jsonb_array_elements_text(COALESCE(t.depends_on::jsonb, '[]'::jsonb)) dependency(task_id)
			JOIN tasks prerequisite ON prerequisite.workflow_id = t.workflow_id AND prerequisite.id = (dependency.task_id::uuid)
			WHERE prerequisite.state <> $3
		  )
		ORDER BY t.available_at ASC, t.created_at ASC
		FOR UPDATE SKIP LOCKED
		LIMIT 1
	`

	task := &models.Task{}
	err = tx.QueryRow(query, string(runtime.TaskStatePending), string(runtime.TaskStateRetrying), string(runtime.TaskStateCompleted)).Scan(
		&task.ID, &task.WorkflowID, &task.TaskType, &task.Name, &task.Payload, &task.State, &task.AssignedWorkerID,
		&task.RetryCount, &task.MaxRetries, &task.DependsOn, &task.AvailableAt, &task.StartedAt, &task.CompletedAt,
		&task.FailedAt, &task.ErrorMessage, &task.CreatedAt, &task.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, tx.Commit()
	}
	if err != nil {
		return nil, err
	}

	now := time.Now()
	_, err = tx.Exec(`
		UPDATE tasks
		SET state = $1,
		    assigned_worker_id = $2,
		    started_at = COALESCE(started_at, $3),
		    updated_at = $3
		WHERE id = $4
	`, string(runtime.TaskStateAssigned), workerID, now, task.ID)
	if err != nil {
		return nil, err
	}

	task.State = string(runtime.TaskStateAssigned)
	task.AssignedWorkerID = workerID
	startedAt := now
	task.StartedAt = &startedAt
	task.UpdatedAt = now

	if _, err := tx.Exec(`
		INSERT INTO task_attempts (id, task_id, workflow_id, worker_id, attempt_number, state, started_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, uuid.New().String(), task.ID, task.WorkflowID, workerID, task.RetryCount+1, string(runtime.TaskStateAssigned), now, now); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return task, nil
}

// CompleteTask marks a task as completed.
func (r *WorkerRepository) CompleteTask(taskID, workerID string) error {
	now := time.Now()
	_, err := r.db.Exec(`
		UPDATE tasks
		SET state = $1, assigned_worker_id = $2, completed_at = $3, updated_at = $3
		WHERE id = $4
	`, string(runtime.TaskStateCompleted), workerID, now, taskID)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(`
		UPDATE task_attempts
		SET state = $1, finished_at = $2
		WHERE task_id = $3 AND worker_id = $4 AND finished_at IS NULL
	`, string(runtime.TaskStateCompleted), now, taskID, workerID)
	return err
}

// FailTask marks a task as failed or retrying.
func (r *WorkerRepository) FailTask(taskID, workerID, errorMessage string, retryCount int, availableAt time.Time, finalState runtime.TaskState) error {
	now := time.Now()
	_, err := r.db.Exec(`
		UPDATE tasks
		SET state = $1, assigned_worker_id = $2, error_message = $3, retry_count = $4,
		    failed_at = $5, available_at = $6, updated_at = $5
		WHERE id = $7
	`, string(finalState), workerID, errorMessage, retryCount, now, availableAt, taskID)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(`
		UPDATE task_attempts
		SET state = $1, finished_at = $2, error_message = $3
		WHERE task_id = $4 AND worker_id = $5 AND finished_at IS NULL
	`, string(finalState), now, errorMessage, taskID, workerID)
	return err
}

// WorkflowComplete checks if a workflow has finished all tasks.
func (r *WorkerRepository) WorkflowComplete(workflowID string) (bool, error) {
	var remaining int
	err := r.db.QueryRow(`
		SELECT COUNT(1)
		FROM tasks
		WHERE workflow_id = $1 AND state NOT IN ($2)
	`, workflowID, string(runtime.TaskStateCompleted)).Scan(&remaining)
	if err != nil {
		return false, err
	}
	return remaining == 0, nil
}

// UpdateWorkflowState updates workflow execution state.
func (r *WorkerRepository) UpdateWorkflowState(workflowID, state string) error {
	_, err := r.db.Exec(`
		UPDATE workflows
		SET status = $1, updated_at = $2
		WHERE id = $3
	`, state, time.Now(), workflowID)
	return err
}

// RecordWorkflowTransition persists a state transition.
func (r *WorkerRepository) RecordWorkflowTransition(workflowID, taskID, entityType, fromState, toState, reason string) error {
	_, err := r.db.Exec(`
		INSERT INTO workflow_transitions (id, workflow_id, task_id, entity_type, from_state, to_state, reason, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, uuid.New().String(), workflowID, taskID, entityType, fromState, toState, reason, time.Now())
	return err
}

// RecordExecutionLog persists an execution log line.
func (r *WorkerRepository) RecordExecutionLog(workflowID, taskID, workerID, level, message string, metadata map[string]interface{}) error {
	encoded, _ := json.Marshal(metadata)
	_, err := r.db.Exec(`
		INSERT INTO execution_logs (id, workflow_id, task_id, worker_id, level, message, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, uuid.New().String(), workflowID, taskID, workerID, level, message, string(encoded), time.Now())
	return err
}

// GetTaskByID retrieves a task by its ID for phase 2 worker runtime.
func (r *WorkerRepository) GetTaskByID(taskID string) (*models.Task, error) {
	task := &models.Task{}
	query := `
		SELECT t.id, t.workflow_id, t.task_type, t.name, t.payload, t.state, t.assigned_worker_id, t.retry_count,
		       t.max_retries, t.depends_on, t.available_at, t.started_at, t.completed_at, t.failed_at,
		       t.error_message, t.created_at, t.updated_at
		FROM tasks t
		WHERE t.id = $1
	`
	err := r.db.QueryRow(query, taskID).Scan(
		&task.ID, &task.WorkflowID, &task.TaskType, &task.Name, &task.Payload, &task.State, &task.AssignedWorkerID,
		&task.RetryCount, &task.MaxRetries, &task.DependsOn, &task.AvailableAt, &task.StartedAt, &task.CompletedAt,
		&task.FailedAt, &task.ErrorMessage, &task.CreatedAt, &task.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return task, nil
}

// UpdateTaskCompletion updates task state and completion time.
func (r *WorkerRepository) UpdateTaskCompletion(taskID string, newState string, workerID string,
	errorMessage string, retryCount int, completedAt *time.Time) error {
	query := `
		UPDATE tasks
		SET state = $1,
		    assigned_worker_id = $2,
		    error_message = $3,
		    retry_count = $4,
		    completed_at = $5,
		    updated_at = NOW()
		WHERE id = $6
	`
	_, err := r.db.Exec(query, newState, workerID, errorMessage, retryCount, completedAt, taskID)
	return err
}

// UpdateTaskRetry schedules a task for retry.
func (r *WorkerRepository) UpdateTaskRetry(taskID string, newRetryCount int, nextRetryTime time.Time) error {
	query := `
		UPDATE tasks
		SET state = $1,
		    retry_count = $2,
		    available_at = $3,
		    updated_at = NOW()
		WHERE id = $4
	`
	_, err := r.db.Exec(query, string(runtime.TaskStateRetrying), newRetryCount, nextRetryTime, taskID)
	return err
}

// RecordHeartbeatSimple records a worker heartbeat without needing userID.
func (r *WorkerRepository) RecordHeartbeat(workerID string, status string) error {
	now := time.Now()
	// Update worker heartbeat
	_, err := r.db.Exec(`
		UPDATE workers
		SET last_heartbeat = $1, status = $2, updated_at = $3
		WHERE id = $4
	`, now, status, now, workerID)
	return err
}
