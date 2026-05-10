package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"atlasflow/backend/worker-service/internal/service"
	"atlasflow/backend/shared/middleware"
)

// WorkerHandler handles worker routes
type WorkerHandler struct {
	service *service.WorkerService
}

// NewWorkerHandler creates a new worker handler
func NewWorkerHandler(service *service.WorkerService) *WorkerHandler {
	return &WorkerHandler{service: service}
}

// RegisterWorker registers a new worker
func (wh *WorkerHandler) RegisterWorker(c *gin.Context) {
	userID := middleware.ExtractUserID(c.Request)
	if userID == "" {
		middleware.RespondError(c.Writer, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req service.RegisterWorkerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondError(c.Writer, http.StatusBadRequest, "invalid request")
		return
	}

	worker, err := wh.service.RegisterWorker(userID, req)
	if err != nil {
		middleware.RespondError(c.Writer, http.StatusInternalServerError, "failed to register worker")
		return
	}

	middleware.RespondJSON(c.Writer, http.StatusCreated, worker)
}

// GetWorker retrieves a worker
func (wh *WorkerHandler) GetWorker(c *gin.Context) {
	userID := middleware.ExtractUserID(c.Request)
	if userID == "" {
		middleware.RespondError(c.Writer, http.StatusUnauthorized, "user not authenticated")
		return
	}

	workerID := c.Param("id")
	worker, err := wh.service.GetWorker(workerID, userID)
	if err != nil {
		middleware.RespondError(c.Writer, http.StatusNotFound, "worker not found")
		return
	}

	middleware.RespondJSON(c.Writer, http.StatusOK, worker)
}

// ListWorkers retrieves workers for a user
func (wh *WorkerHandler) ListWorkers(c *gin.Context) {
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

	workers, err := wh.service.ListWorkers(userID, limit, offset)
	if err != nil {
		middleware.RespondError(c.Writer, http.StatusInternalServerError, "failed to list workers")
		return
	}

	middleware.RespondJSON(c.Writer, http.StatusOK, workers)
}

// RecordHeartbeat records a worker heartbeat
func (wh *WorkerHandler) RecordHeartbeat(c *gin.Context) {
	userID := middleware.ExtractUserID(c.Request)
	if userID == "" {
		middleware.RespondError(c.Writer, http.StatusUnauthorized, "user not authenticated")
		return
	}

	workerID := c.Param("id")

	var req service.HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondError(c.Writer, http.StatusBadRequest, "invalid request")
		return
	}

	err := wh.service.RecordHeartbeat(workerID, userID, req)
	if err != nil {
		middleware.RespondError(c.Writer, http.StatusInternalServerError, "failed to record heartbeat")
		return
	}

	middleware.RespondMessage(c.Writer, http.StatusOK, "heartbeat recorded")
}
