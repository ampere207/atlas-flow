package runtime

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
)

// NATSEventPublisher implements EventPublisher using NATS.
type NATSEventPublisher struct {
	nc *nats.Conn
}

// NewNATSEventPublisher creates a new NATSEventPublisher.
func NewNATSEventPublisher(nc *nats.Conn) *NATSEventPublisher {
	return &NATSEventPublisher{nc: nc}
}

// PublishEvent publishes a single event to NATS.
func (p *NATSEventPublisher) PublishEvent(ctx context.Context, event *ExecutionEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// Determine subject based on event type
	var subject string
	if event.TaskID != "" {
		subject = fmt.Sprintf("tasks.%s.events", event.TaskID)
	} else if event.WorkerID != "" {
		subject = fmt.Sprintf("workers.%s.events", event.WorkerID)
	} else {
		subject = fmt.Sprintf("workflows.%s.events", event.WorkflowID)
	}

	return p.nc.Publish(subject, data)
}

// PublishEventBatch publishes multiple events.
func (p *NATSEventPublisher) PublishEventBatch(ctx context.Context, events []*ExecutionEvent) error {
	for _, event := range events {
		if err := p.PublishEvent(ctx, event); err != nil {
			return err
		}
	}
	return nil
}
