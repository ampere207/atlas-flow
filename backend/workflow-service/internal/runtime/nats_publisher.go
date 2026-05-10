package runtime

import (
	"encoding/json"

	"github.com/nats-io/nats.go"
)

// NATSPublisher publishes orchestration events to NATS.
type NATSPublisher struct {
	nc *nats.Conn
}

// NewNATSPublisher creates a NATS-backed publisher.
func NewNATSPublisher(nc *nats.Conn) *NATSPublisher {
	return &NATSPublisher{nc: nc}
}

// PublishEvent publishes the event payload to a NATS subject.
func (publisher *NATSPublisher) PublishEvent(subject string, event map[string]interface{}) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return publisher.nc.Publish(subject, payload)
}
