package subscriber

import (
	"context"
	"log"

	"github.com/nats-io/nats.go"
)

// StartSubscriptions wires runtime subjects into the event service.
func StartSubscriptions(ctx context.Context, nc *nats.Conn) error {
	subjects := []string{"workflow.*", "task.*", "worker.*"}
	for _, subject := range subjects {
		_, err := nc.Subscribe(subject, func(msg *nats.Msg) {
			log.Printf("event-service received %s: %s", msg.Subject, string(msg.Data))
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
