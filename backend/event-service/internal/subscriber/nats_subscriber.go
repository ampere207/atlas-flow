package subscriber

import (
	"context"
	"encoding/json"
	"log"

	"atlasflow/backend/shared/runtime"

	"github.com/nats-io/nats.go"
)

// StartSubscriptions wires runtime subjects into the event service.
func StartSubscriptions(ctx context.Context, nc *nats.Conn, store runtime.EventStore) error {
	// Listen to all events published by the orchestrator and workers
	subjects := []string{"workflows.*.events", "tasks.*.events", "workers.*.events"}
	
	for _, subject := range subjects {
		_, err := nc.Subscribe(subject, func(msg *nats.Msg) {
			var event runtime.ExecutionEvent
			if err := json.Unmarshal(msg.Data, &event); err != nil {
				log.Printf("! Failed to unmarshal event from %s: %v", msg.Subject, err)
				return
			}

			// Persist to Postgres
			if err := store.StoreEvent(context.Background(), &event); err != nil {
				log.Printf("! Failed to persist event %s: %v", event.EventID, err)
			} else {
				log.Printf("✓ Persisted event: %s (%s)", event.EventType, event.EventID)
			}
		})
		if err != nil {
			return err
		}
	}

	go func() {
		<-ctx.Done()
		_ = nc.Drain()
	}()

	return nil
}
