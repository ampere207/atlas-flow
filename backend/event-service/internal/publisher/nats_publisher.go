package publisher

import (
	"encoding/json"

	"github.com/nats-io/nats.go"
)

// EventPublisher defines the event publishing interface
type EventPublisher interface {
	PublishEvent(subject string, event map[string]interface{}) error
}

// NATSPublisher implements EventPublisher using NATS
type NATSPublisher struct {
	nc *nats.Conn
}

// NewNATSPublisher creates a new NATS publisher
func NewNATSPublisher(nc *nats.Conn) *NATSPublisher {
	return &NATSPublisher{nc: nc}
}

// PublishEvent publishes an event to NATS
func (np *NATSPublisher) PublishEvent(subject string, event map[string]interface{}) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return np.nc.Publish(subject, data)
}

// PublishWorkflowCreated publishes a workflow created event
func (np *NATSPublisher) PublishWorkflowCreated(workflowID, userID, name string) error {
	event := map[string]interface{}{
		"workflow_id":   workflowID,
		"user_id":       userID,
		"workflow_name": name,
	}
	return np.PublishEvent("workflow.created", event)
}

// PublishWorkflowUpdated publishes a workflow updated event
func (np *NATSPublisher) PublishWorkflowUpdated(workflowID, userID, status string) error {
	event := map[string]interface{}{
		"workflow_id": workflowID,
		"user_id":     userID,
		"status":      status,
	}
	return np.PublishEvent("workflow.updated", event)
}

// PublishWorkerRegistered publishes a worker registered event
func (np *NATSPublisher) PublishWorkerRegistered(workerID, userID, name string) error {
	event := map[string]interface{}{
		"worker_id": workerID,
		"user_id":   userID,
		"name":      name,
	}
	return np.PublishEvent("worker.registered", event)
}

// PublishWorkerHeartbeat publishes a worker heartbeat event
func (np *NATSPublisher) PublishWorkerHeartbeat(workerID, userID, status string) error {
	event := map[string]interface{}{
		"worker_id": workerID,
		"user_id":   userID,
		"status":    status,
	}
	return np.PublishEvent("worker.heartbeat", event)
}
