package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"atlasflow/backend/event-service/internal/publisher"
	"atlasflow/backend/shared/middleware"
)

// EventHandler handles event routes
type EventHandler struct {
	publisher publisher.EventPublisher
}

// NewEventHandler creates a new event handler
func NewEventHandler(publisher publisher.EventPublisher) *EventHandler {
	return &EventHandler{publisher: publisher}
}

// PublishEvent publishes an event
func (eh *EventHandler) PublishEvent(c *gin.Context) {
	var req struct {
		Subject string                 `json:"subject" binding:"required"`
		Event   map[string]interface{} `json:"event" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondError(c.Writer, http.StatusBadRequest, "invalid request")
		return
	}

	err := eh.publisher.PublishEvent(req.Subject, req.Event)
	if err != nil {
		middleware.RespondError(c.Writer, http.StatusInternalServerError, "failed to publish event")
		return
	}

	middleware.RespondMessage(c.Writer, http.StatusOK, "event published successfully")
}

// GetStatus returns the status of the event service
func (eh *EventHandler) GetStatus(c *gin.Context) {
	middleware.RespondJSON(c.Writer, http.StatusOK, map[string]string{
		"status": "healthy",
		"service": "event-service",
	})
}
