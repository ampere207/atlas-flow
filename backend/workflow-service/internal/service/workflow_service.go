package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"atlasflow/backend/shared/models"
	sharedruntime "atlasflow/backend/shared/runtime"
	"atlasflow/backend/workflow-service/internal/repository"
	workflowruntime "atlasflow/backend/workflow-service/internal/runtime"

	"github.com/google/uuid"
)

// WorkflowService handles workflow business logic
type WorkflowService struct {
	repo         *repository.WorkflowRepository
	orchestrator *workflowruntime.Orchestrator
}

// NewWorkflowService creates a new workflow service
func NewWorkflowService(repo *repository.WorkflowRepository, orchestrator *workflowruntime.Orchestrator) *WorkflowService {
	return &WorkflowService{repo: repo, orchestrator: orchestrator}
}

// CreateWorkflowRequest represents a create workflow request
type CreateWorkflowRequest struct {
	Name       string                           `json:"name" binding:"required"`
	Metadata   map[string]interface{}           `json:"metadata"`
	Definition sharedruntime.WorkflowDefinition `json:"definition"`
}

// CreateWorkflow creates a new workflow
func (ws *WorkflowService) CreateWorkflow(userID string, req CreateWorkflowRequest) (*models.Workflow, error) {
	log.Printf("workflow service create start: user=%s name=%q tasks=%d", userID, req.Name, len(req.Definition.Tasks))

	if len(req.Definition.Tasks) == 0 {
		workflow, err := ws.repo.Create(userID, req.Name, req.Metadata)
		if err != nil {
			return nil, fmt.Errorf("create workflow row: %w", err)
		}
		_ = ws.repo.CreateEvent(workflow.ID, "workflow_created", map[string]interface{}{
			"workflow_id":   workflow.ID,
			"workflow_name": workflow.Name,
		})
		return workflow, nil
	}

	// Map client provided IDs to real database UUIDs
	idMap := make(map[string]string)
	for i := range req.Definition.Tasks {
		newID := uuid.New().String()
		oldID := req.Definition.Tasks[i].ID
		if oldID != "" {
			idMap[oldID] = newID
		}
		req.Definition.Tasks[i].ID = newID
	}

	for i := range req.Definition.Tasks {
		newDependsOn := make([]string, 0)
		for _, dep := range req.Definition.Tasks[i].DependsOn {
			if mapped, ok := idMap[dep]; ok {
				newDependsOn = append(newDependsOn, mapped)
			} else {
				newDependsOn = append(newDependsOn, dep) // fallback
			}
		}
		req.Definition.Tasks[i].DependsOn = newDependsOn
	}

	if err := req.Definition.Validate(); err != nil {
		log.Printf("workflow definition validation failed: user=%s name=%q tasks=%d err=%v", userID, req.Name, len(req.Definition.Tasks), err)
		return nil, fmt.Errorf("validate workflow definition: %w", err)
	}

	workflow, err := ws.repo.CreateExecutionWorkflow(userID, req.Name, req.Metadata, req.Definition)
	if err != nil {
		log.Printf("workflow execution row create failed: user=%s name=%q err=%v", userID, req.Name, err)
		return nil, fmt.Errorf("create execution workflow row: %w", err)
	}

	for idx, taskDefinition := range req.Definition.Tasks {
		payloadJSON, _ := json.Marshal(taskDefinition.Payload)
		dependsOnJSON, _ := json.Marshal(taskDefinition.DependsOn)
		maxRetries := taskDefinition.RetryPolicy.MaxAttempts
		if maxRetries == 0 {
			maxRetries = 3
		}
		if taskDefinition.TimeoutSeconds == 0 {
			taskDefinition.TimeoutSeconds = 300
		}
		taskName := taskDefinition.Name
		if taskName == "" {
			taskName = taskDefinition.Type
		}
		if taskName == "" {
			taskName = taskDefinition.ID
		}
		task := &models.Task{
			ID:          taskDefinition.ID,
			WorkflowID:  workflow.ID,
			TaskType:    taskDefinition.Type,
			Name:        taskName,
			Payload:     string(payloadJSON),
			State:       string(sharedruntime.TaskStatePending),
			DependsOn:   string(dependsOnJSON),
			RetryCount:  0,
			MaxRetries:  maxRetries,
			AvailableAt: time.Now().UTC(),
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}
		if err := ws.repo.AddTask(task); err != nil {
			log.Printf("workflow task insert failed: user=%s workflow_id=%s task_index=%d task_id=%s task_type=%s task_name=%q depends_on=%v err=%v", userID, workflow.ID, idx, task.ID, task.TaskType, task.Name, taskDefinition.DependsOn, err)
			return nil, fmt.Errorf("add task %s (%s): %w", task.ID, task.TaskType, err)
		}
		if err := ws.repo.AddTransition(&models.WorkflowTransition{
			ID:         uuid.New().String(),
			WorkflowID: workflow.ID,
			TaskID:     task.ID,
			EntityType: "task",
			FromState:  string(sharedruntime.TaskStatePending),
			ToState:    string(sharedruntime.TaskStatePending),
			Reason:     "task registered",
			CreatedAt:  time.Now().UTC(),
		}); err != nil {
			log.Printf("workflow task transition insert failed: user=%s workflow_id=%s task_id=%s err=%v", userID, workflow.ID, task.ID, err)
			return nil, fmt.Errorf("add task transition %s: %w", task.ID, err)
		}
	}

	_ = ws.repo.CreateEvent(workflow.ID, "workflow_created", map[string]interface{}{
		"workflow_id":   workflow.ID,
		"workflow_name": workflow.Name,
		"definition":    req.Definition,
	})

	return workflow, nil
}

// GetWorkflow retrieves a workflow
func (ws *WorkflowService) GetWorkflow(id, userID string) (*models.Workflow, error) {
	return ws.repo.GetByID(id, userID)
}

// ListWorkflows retrieves workflows for a user
func (ws *WorkflowService) ListWorkflows(userID string, limit, offset int) ([]*models.Workflow, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	return ws.repo.ListByUserID(userID, limit, offset)
}

// UpdateWorkflowStatus updates workflow status
func (ws *WorkflowService) UpdateWorkflowStatus(id, userID, status string) error {
	_ = ws.repo.CreateEvent(id, "workflow_updated", map[string]interface{}{
		"workflow_id": id,
		"status":      status,
	})

	return ws.repo.UpdateWorkflowState(id, userID, status)
}

// ExecuteWorkflow starts workflow execution.
func (ws *WorkflowService) ExecuteWorkflow(id, userID string) (*models.Workflow, error) {
	if ws.orchestrator == nil {
		if err := ws.repo.UpdateWorkflowState(id, userID, string(sharedruntime.WorkflowStateRunning)); err != nil {
			return nil, err
		}
		return ws.repo.GetExecutionWorkflow(id, userID)
	}
	return ws.orchestrator.StartWorkflow(context.Background(), id, userID)
}

// CancelWorkflow cancels a workflow execution.
func (ws *WorkflowService) CancelWorkflow(id, userID string) error {
	if ws.orchestrator == nil {
		return ws.repo.UpdateWorkflowState(id, userID, string(sharedruntime.WorkflowStateCancelled))
	}
	return ws.orchestrator.CancelWorkflow(context.Background(), id, userID)
}

// ListWorkflowTasks returns execution tasks.
func (ws *WorkflowService) ListWorkflowTasks(id, userID string) ([]*models.Task, error) {
	if ws.orchestrator != nil {
		return ws.orchestrator.ListTasks(id, userID)
	}
	return ws.repo.ListTasksByWorkflow(id, userID)
}

// ListWorkflowHistory returns execution transitions.
func (ws *WorkflowService) ListWorkflowHistory(id, userID string) ([]*models.WorkflowTransition, error) {
	if ws.orchestrator != nil {
		return ws.orchestrator.ListHistory(id, userID)
	}
	return ws.repo.ListTransitionsByWorkflow(id, userID)
}

// GetWorkflowExecutionStatus returns the latest workflow execution snapshot.
func (ws *WorkflowService) GetWorkflowExecutionStatus(id, userID string) (*models.Workflow, error) {
	if ws.orchestrator != nil {
		return ws.orchestrator.GetStatus(id, userID)
	}
	return ws.repo.GetExecutionWorkflow(id, userID)
}
