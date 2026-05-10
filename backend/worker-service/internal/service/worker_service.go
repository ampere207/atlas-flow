package service

import (
	"atlasflow/backend/worker-service/internal/repository"
	"atlasflow/backend/shared/models"
)

// WorkerService handles worker business logic
type WorkerService struct {
	repo *repository.WorkerRepository
}

// NewWorkerService creates a new worker service
func NewWorkerService(repo *repository.WorkerRepository) *WorkerService {
	return &WorkerService{repo: repo}
}

// RegisterWorkerRequest represents a worker registration request
type RegisterWorkerRequest struct {
	Name string `json:"name" binding:"required"`
}

// HeartbeatRequest represents a heartbeat request
type HeartbeatRequest struct {
	Status string `json:"status" binding:"required"`
}

// RegisterWorker registers a new worker
func (ws *WorkerService) RegisterWorker(userID string, req RegisterWorkerRequest) (*models.Worker, error) {
	return ws.repo.Create(userID, req.Name)
}

// GetWorker retrieves a worker
func (ws *WorkerService) GetWorker(id, userID string) (*models.Worker, error) {
	return ws.repo.GetByID(id, userID)
}

// ListWorkers retrieves workers for a user
func (ws *WorkerService) ListWorkers(userID string, limit, offset int) ([]*models.Worker, error) {
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

// RecordHeartbeat records a worker heartbeat
func (ws *WorkerService) RecordHeartbeat(id, userID string, req HeartbeatRequest) error {
	return ws.repo.UpdateHeartbeat(id, userID, req.Status)
}
