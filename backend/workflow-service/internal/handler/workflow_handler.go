package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"atlasflow/backend/shared/middleware"
	"atlasflow/backend/shared/runtime"
	"atlasflow/backend/workflow-service/internal/service"
)

// WorkflowHandler handles workflow routes
type WorkflowHandler struct {
	service       *service.WorkflowService
	workerConnMgr *runtime.WorkerConnectionManager
}

// NewWorkflowHandler creates a new workflow handler
func NewWorkflowHandler(service *service.WorkflowService, workerConnMgr *runtime.WorkerConnectionManager) *WorkflowHandler {
	return &WorkflowHandler{
		service:       service,
		workerConnMgr: workerConnMgr,
	}
}

// CreateWorkflow creates a new workflow
func (wh *WorkflowHandler) CreateWorkflow(c *gin.Context) {
	userID := middleware.ExtractUserID(c.Request)
	if userID == "" {
		middleware.RespondError(c.Writer, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req service.CreateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("workflow create bind failed: user=%s err=%v", userID, err)
		middleware.RespondError(c.Writer, http.StatusBadRequest, "invalid request")
		return
	}

	log.Printf("workflow create request received: user=%s name=%q tasks=%d metadata_keys=%d", userID, req.Name, len(req.Definition.Tasks), len(req.Metadata))

	workflow, err := wh.service.CreateWorkflow(userID, req)
	if err != nil {
		log.Printf("workflow create failed: user=%s name=%q tasks=%d err=%v", userID, req.Name, len(req.Definition.Tasks), err)
		middleware.RespondError(c.Writer, http.StatusInternalServerError, "failed to create workflow")
		return
	}

	middleware.RespondJSON(c.Writer, http.StatusCreated, workflow)
}

// GetWorkflow retrieves a workflow
func (wh *WorkflowHandler) GetWorkflow(c *gin.Context) {
	userID := middleware.ExtractUserID(c.Request)
	if userID == "" {
		middleware.RespondError(c.Writer, http.StatusUnauthorized, "user not authenticated")
		return
	}

	workflowID := c.Param("id")
	workflow, err := wh.service.GetWorkflow(workflowID, userID)
	if err != nil {
		middleware.RespondError(c.Writer, http.StatusNotFound, "workflow not found")
		return
	}

	middleware.RespondJSON(c.Writer, http.StatusOK, workflow)
}

// ListWorkflows retrieves workflows for a user
func (wh *WorkflowHandler) ListWorkflows(c *gin.Context) {
	userID := middleware.ExtractUserID(c.Request)
	if userID == "" {
		middleware.RespondError(c.Writer, http.StatusUnauthorized, "user not authenticated")
		return
	}

	limit := 10
	offset := 0

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil {
			offset = parsed
		}
	}

	workflows, err := wh.service.ListWorkflows(userID, limit, offset)
	if err != nil {
		middleware.RespondError(c.Writer, http.StatusInternalServerError, "failed to list workflows")
		return
	}

	middleware.RespondJSON(c.Writer, http.StatusOK, workflows)
}

// UpdateWorkflowStatus updates workflow status
func (wh *WorkflowHandler) UpdateWorkflowStatus(c *gin.Context) {
	userID := middleware.ExtractUserID(c.Request)
	if userID == "" {
		middleware.RespondError(c.Writer, http.StatusUnauthorized, "user not authenticated")
		return
	}

	workflowID := c.Param("id")

	var req struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondError(c.Writer, http.StatusBadRequest, "invalid request")
		return
	}

	err := wh.service.UpdateWorkflowStatus(workflowID, userID, req.Status)
	if err != nil {
		middleware.RespondError(c.Writer, http.StatusInternalServerError, "failed to update workflow")
		return
	}

	middleware.RespondMessage(c.Writer, http.StatusOK, "workflow updated successfully")
}

// ExecuteWorkflow starts a workflow execution.
func (wh *WorkflowHandler) ExecuteWorkflow(c *gin.Context) {
	userID := middleware.ExtractUserID(c.Request)
	if userID == "" {
		middleware.RespondError(c.Writer, http.StatusUnauthorized, "user not authenticated")
		return
	}

	workflowID := c.Param("id")
	workflow, err := wh.service.ExecuteWorkflow(workflowID, userID)
	if err != nil {
		middleware.RespondError(c.Writer, http.StatusInternalServerError, "failed to execute workflow")
		return
	}

	middleware.RespondJSON(c.Writer, http.StatusOK, workflow)
}

// CancelWorkflow cancels a running workflow.
func (wh *WorkflowHandler) CancelWorkflow(c *gin.Context) {
	userID := middleware.ExtractUserID(c.Request)
	if userID == "" {
		middleware.RespondError(c.Writer, http.StatusUnauthorized, "user not authenticated")
		return
	}

	workflowID := c.Param("id")
	if err := wh.service.CancelWorkflow(workflowID, userID); err != nil {
		middleware.RespondError(c.Writer, http.StatusInternalServerError, "failed to cancel workflow")
		return
	}

	middleware.RespondMessage(c.Writer, http.StatusOK, "workflow cancelled")
}

// ListWorkflowTasks returns the workflow's execution tasks.
func (wh *WorkflowHandler) ListWorkflowTasks(c *gin.Context) {
	userID := middleware.ExtractUserID(c.Request)
	if userID == "" {
		middleware.RespondError(c.Writer, http.StatusUnauthorized, "user not authenticated")
		return
	}

	workflowID := c.Param("id")
	tasks, err := wh.service.ListWorkflowTasks(workflowID, userID)
	if err != nil {
		middleware.RespondError(c.Writer, http.StatusInternalServerError, "failed to list workflow tasks")
		return
	}

	middleware.RespondJSON(c.Writer, http.StatusOK, tasks)
}

// ListWorkflowHistory returns the workflow transition history.
func (wh *WorkflowHandler) ListWorkflowHistory(c *gin.Context) {
	userID := middleware.ExtractUserID(c.Request)
	if userID == "" {
		middleware.RespondError(c.Writer, http.StatusUnauthorized, "user not authenticated")
		return
	}

	workflowID := c.Param("id")
	history, err := wh.service.ListWorkflowHistory(workflowID, userID)
	if err != nil {
		middleware.RespondError(c.Writer, http.StatusInternalServerError, "failed to load workflow history")
		return
	}

	middleware.RespondJSON(c.Writer, http.StatusOK, history)
}

// GetWorkflowExecutionStatus returns the current execution snapshot.
func (wh *WorkflowHandler) GetWorkflowExecutionStatus(c *gin.Context) {
	userID := middleware.ExtractUserID(c.Request)
	if userID == "" {
		middleware.RespondError(c.Writer, http.StatusUnauthorized, "user not authenticated")
		return
	}

	workflowID := c.Param("id")
	status, err := wh.service.GetWorkflowExecutionStatus(workflowID, userID)
	if err != nil {
		middleware.RespondError(c.Writer, http.StatusInternalServerError, "failed to load workflow status")
		return
	}

	middleware.RespondJSON(c.Writer, http.StatusOK, status)
}

// StreamWorkflowExecution emits live workflow snapshots over SSE.
func (wh *WorkflowHandler) StreamWorkflowExecution(c *gin.Context) {
	userID := middleware.ExtractUserID(c.Request)
	if userID == "" {
		middleware.RespondError(c.Writer, http.StatusUnauthorized, "user not authenticated")
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		middleware.RespondError(c.Writer, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	workflowID := c.Param("id")
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			workflow, workflowErr := wh.service.GetWorkflowExecutionStatus(workflowID, userID)
			tasks, tasksErr := wh.service.ListWorkflowTasks(workflowID, userID)
			history, historyErr := wh.service.ListWorkflowHistory(workflowID, userID)
			payload := map[string]interface{}{}
			if workflowErr == nil {
				payload["workflow"] = workflow
			}
			if tasksErr == nil {
				payload["tasks"] = tasks
			}
			if historyErr == nil {
				payload["history"] = history
			}
			encoded, _ := json.Marshal(payload)
			_, _ = c.Writer.Write([]byte("event: snapshot\n"))
			_, _ = c.Writer.Write([]byte("data: "))
			_, _ = c.Writer.Write(encoded)
			_, _ = c.Writer.Write([]byte("\n\n"))
			flusher.Flush()
		}
	}
}

// GetWorkers retrieves all workers for the authenticated user
func (wh *WorkflowHandler) GetWorkers(c *gin.Context) {
	userID := middleware.ExtractUserID(c.Request)
	if userID == "" {
		middleware.RespondError(c.Writer, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Get workers owned by this user
	workers := wh.workerConnMgr.GetWorkersByUser(userID)

	// Format response
	response := make([]map[string]interface{}, len(workers))
	for i, w := range workers {
		response[i] = map[string]interface{}{
			"worker_id":       w.WorkerID,
			"user_id":         w.UserID,
			"status":          w.Status,
			"capabilities":    w.Capabilities,
			"capacity":        w.Capacity,
			"running_tasks":   w.RunningTasks,
			"completed_tasks": w.CompletedTasks,
			"failed_tasks":    w.FailedTasks,
			"last_heartbeat":  w.LastHeartbeat,
		}
	}

	middleware.RespondJSON(c.Writer, http.StatusOK, response)
}

// GetClusterMetrics retrieves cluster-wide metrics
func (wh *WorkflowHandler) GetClusterMetrics(c *gin.Context) {
	userID := middleware.ExtractUserID(c.Request)
	if userID == "" {
		middleware.RespondError(c.Writer, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Get workers owned by this user
	workers := wh.workerConnMgr.GetWorkersByUser(userID)

	// Calculate metrics
	activeWorkers := 0
	idleWorkers := 0
	offlineWorkers := 0
	totalCompletedTasks := int64(0)
	totalRunningTasks := int32(0)

	for _, w := range workers {
		switch w.Status {
		case "connected":
			if w.RunningTasks > 0 {
				activeWorkers++
			} else {
				idleWorkers++
			}
		case "dead", "disconnected":
			offlineWorkers++
		}
		totalCompletedTasks += w.CompletedTasks
		totalRunningTasks += w.RunningTasks
	}

	response := map[string]interface{}{
		"total_workers":         len(workers),
		"active_workers":        activeWorkers,
		"idle_workers":          idleWorkers,
		"offline_workers":       offlineWorkers,
		"total_tasks_in_queue":  totalRunningTasks,
		"completed_tasks_total": totalCompletedTasks,
	}

	middleware.RespondJSON(c.Writer, http.StatusOK, response)
}
