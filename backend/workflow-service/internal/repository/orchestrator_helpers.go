package repository

import (
	"time"

	"atlasflow/backend/shared/models"
)

// This file contains helper methods needed by the NATS orchestrator
// Add these methods to the WorkflowRepository in workflow_repository.go

// GetRunningWorkflows returns all workflows with status "running"
func (r *WorkflowRepository) GetRunningWorkflows() ([]*models.Workflow, error) {
	query := `
		SELECT id, user_id, name, status, metadata, definition, created_at, updated_at
		FROM workflows
		WHERE status = 'running'
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	workflows := make([]*models.Workflow, 0)
	for rows.Next() {
		workflow := &models.Workflow{}
		err := rows.Scan(
			&workflow.ID, &workflow.UserID, &workflow.Name, &workflow.Status, &workflow.Metadata, &workflow.Definition, &workflow.CreatedAt, &workflow.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		workflows = append(workflows, workflow)
	}

	return workflows, nil
}

// UpdateWorkflowStatus updates a workflow's status
func (r *WorkflowRepository) UpdateWorkflowStatus(workflowID string, status string) error {
	query := `
		UPDATE workflows
		SET status = $1, updated_at = $2
		WHERE id = $3
	`

	_, err := r.db.Exec(query, status, time.Now(), workflowID)
	return err
}

// GetTask retrieves a task by ID
func (r *WorkflowRepository) GetTask(taskID string, workflowID string) (*models.Task, error) {
	query := `
		SELECT id, workflow_id, task_type, name, payload, state, assigned_worker_id, retry_count,
		       max_retries, depends_on, available_at, started_at, completed_at, failed_at,
		       error_message, created_at, updated_at
		FROM tasks
		WHERE id = $1 AND workflow_id = $2
	`

	task := &models.Task{}
	err := r.db.QueryRow(query, taskID, workflowID).Scan(
		&task.ID, &task.WorkflowID, &task.TaskType, &task.Name, &task.Payload, &task.State, &task.AssignedWorkerID,
		&task.RetryCount, &task.MaxRetries, &task.DependsOn, &task.AvailableAt, &task.StartedAt, &task.CompletedAt,
		&task.FailedAt, &task.ErrorMessage, &task.CreatedAt, &task.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return task, nil
}

// UpdateTaskStateOrchestrator updates task state (simpler version for orchestrator use)
func (r *WorkflowRepository) UpdateTaskStateOrchestrator(taskID string, state string, workerID string) error {
	query := `
		UPDATE tasks
		SET state = $1, assigned_worker_id = $2, updated_at = $3
		WHERE id = $4
	`

	_, err := r.db.Exec(query, state, workerID, time.Now(), taskID)
	return err
}

// UpdateTaskRetryCount updates the retry count for a task
func (r *WorkflowRepository) UpdateTaskRetryCount(taskID string, retryCount int) error {
	query := `
		UPDATE tasks
		SET retry_count = $1, updated_at = $2
		WHERE id = $3
	`

	_, err := r.db.Exec(query, retryCount, time.Now(), taskID)
	return err
}

// UpdateTaskError updates the error message for a task
func (r *WorkflowRepository) UpdateTaskError(taskID string, errorMsg string) error {
	query := `
		UPDATE tasks
		SET error_message = $1, updated_at = $2, failed_at = $3
		WHERE id = $4
	`

	_, err := r.db.Exec(query, errorMsg, time.Now(), time.Now(), taskID)
	return err
}
