package runtime

import (
	"context"
	"encoding/json"
	"time"
)

// ExecutionEvent represents an event in the workflow execution lifecycle.
type ExecutionEvent struct {
	EventID      string                 `json:"event_id"`
	EventType    string                 `json:"event_type"` // workflow_started, task_assigned, task_completed, etc.
	WorkflowID   string                 `json:"workflow_id"`
	TaskID       string                 `json:"task_id,omitempty"`
	WorkerID     string                 `json:"worker_id,omitempty"`
	UserID       string                 `json:"user_id"`
	Timestamp    time.Time              `json:"timestamp"`
	Data         map[string]interface{} `json:"data,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ExecutionEventType defines event types in the system.
type ExecutionEventType string

const (
	// Workflow events
	EventWorkflowCreated   ExecutionEventType = "workflow_created"
	EventWorkflowStarted   ExecutionEventType = "workflow_started"
	EventWorkflowRunning   ExecutionEventType = "workflow_running"
	EventWorkflowCompleted ExecutionEventType = "workflow_completed"
	EventWorkflowFailed    ExecutionEventType = "workflow_failed"
	EventWorkflowCancelled ExecutionEventType = "workflow_cancelled"
	EventWorkflowRetrying  ExecutionEventType = "workflow_retrying"

	// Task events
	EventTaskCreated         ExecutionEventType = "task_created"
	EventTaskScheduled       ExecutionEventType = "task_scheduled"
	EventTaskAssigned        ExecutionEventType = "task_assigned"
	EventTaskStarted         ExecutionEventType = "task_started"
	EventTaskCompleted       ExecutionEventType = "task_completed"
	EventTaskFailed          ExecutionEventType = "task_failed"
	EventTaskRetrying        ExecutionEventType = "task_retrying"
	EventTaskTimeoutOccurred ExecutionEventType = "task_timeout_occurred"

	// Worker events
	EventWorkerRegistered ExecutionEventType = "worker_registered"
	EventWorkerHeartbeat  ExecutionEventType = "worker_heartbeat"
	EventWorkerUnhealthy  ExecutionEventType = "worker_unhealthy"
	EventWorkerDead       ExecutionEventType = "worker_dead"

	// Failure recovery events
	EventTaskOrphaned               ExecutionEventType = "task_orphaned"
	EventTaskReassigned             ExecutionEventType = "task_reassigned"
	ExecutionEventRecoveryStarted   ExecutionEventType = "recovery_started"
	ExecutionEventRecoveryCompleted ExecutionEventType = "recovery_completed"
)

// EventPublisher publishes execution events.
type EventPublisher interface {
	PublishEvent(ctx context.Context, event *ExecutionEvent) error
	PublishEventBatch(ctx context.Context, events []*ExecutionEvent) error
}

// EventSubscriber subscribes to execution events.
type EventSubscriber interface {
	Subscribe(ctx context.Context, eventType ExecutionEventType, handler EventHandler) error
	SubscribeAll(ctx context.Context, handler EventHandler) error
	Unsubscribe(eventType ExecutionEventType) error
}

// EventHandler is called when an event is published.
type EventHandler func(ctx context.Context, event *ExecutionEvent) error

// InMemoryEventBus is a simple pub/sub for development/testing.
type InMemoryEventBus struct {
	subscribers map[string][]EventHandler
	history     []*ExecutionEvent
}

// NewInMemoryEventBus creates an event bus for testing.
func NewInMemoryEventBus() *InMemoryEventBus {
	return &InMemoryEventBus{
		subscribers: make(map[string][]EventHandler),
		history:     make([]*ExecutionEvent, 0),
	}
}

// PublishEvent publishes an event to all subscribers.
func (bus *InMemoryEventBus) PublishEvent(ctx context.Context, event *ExecutionEvent) error {
	if event.EventID == "" {
		event.EventID = "event-" + time.Now().Format("20060102150405")
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	bus.history = append(bus.history, event)

	// Call handlers
	handlers := bus.subscribers[event.EventType]
	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			// Log error but continue with other handlers
			_ = err
		}
	}

	// Also call "all" subscribers
	if allHandlers, exists := bus.subscribers["*"]; exists {
		for _, handler := range allHandlers {
			if err := handler(ctx, event); err != nil {
				_ = err
			}
		}
	}

	return nil
}

// PublishEventBatch publishes multiple events.
func (bus *InMemoryEventBus) PublishEventBatch(ctx context.Context, events []*ExecutionEvent) error {
	for _, event := range events {
		if err := bus.PublishEvent(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

// Subscribe registers a handler for a specific event type.
func (bus *InMemoryEventBus) Subscribe(ctx context.Context, eventType ExecutionEventType, handler EventHandler) error {
	typeStr := string(eventType)
	bus.subscribers[typeStr] = append(bus.subscribers[typeStr], handler)
	return nil
}

// SubscribeAll registers a handler for all event types.
func (bus *InMemoryEventBus) SubscribeAll(ctx context.Context, handler EventHandler) error {
	bus.subscribers["*"] = append(bus.subscribers["*"], handler)
	return nil
}

// Unsubscribe removes all handlers for an event type.
func (bus *InMemoryEventBus) Unsubscribe(eventType ExecutionEventType) error {
	delete(bus.subscribers, string(eventType))
	return nil
}

// GetEventHistory returns all published events.
func (bus *InMemoryEventBus) GetEventHistory() []*ExecutionEvent {
	return bus.history
}

// EventBuilder helps construct events fluently.
type EventBuilder struct {
	event *ExecutionEvent
}

// NewEventBuilder creates a new event builder.
func NewEventBuilder(eventType ExecutionEventType) *EventBuilder {
	return &EventBuilder{
		event: &ExecutionEvent{
			EventType: string(eventType),
			Timestamp: time.Now(),
			Data:      make(map[string]interface{}),
			Metadata:  make(map[string]interface{}),
		},
	}
}

// WorkflowID sets the workflow ID.
func (eb *EventBuilder) WorkflowID(id string) *EventBuilder {
	eb.event.WorkflowID = id
	return eb
}

// TaskID sets the task ID.
func (eb *EventBuilder) TaskID(id string) *EventBuilder {
	eb.event.TaskID = id
	return eb
}

// WorkerID sets the worker ID.
func (eb *EventBuilder) WorkerID(id string) *EventBuilder {
	eb.event.WorkerID = id
	return eb
}

// UserID sets the user ID.
func (eb *EventBuilder) UserID(id string) *EventBuilder {
	eb.event.UserID = id
	return eb
}

// Data adds data to the event.
func (eb *EventBuilder) Data(key string, value interface{}) *EventBuilder {
	eb.event.Data[key] = value
	return eb
}

// Error sets an error message.
func (eb *EventBuilder) Error(err string) *EventBuilder {
	eb.event.ErrorMessage = err
	return eb
}

// Build returns the constructed event.
func (eb *EventBuilder) Build() *ExecutionEvent {
	return eb.event
}

// EventStore persists and retrieves events.
type EventStore interface {
	StoreEvent(ctx context.Context, event *ExecutionEvent) error
	StoreEventBatch(ctx context.Context, events []*ExecutionEvent) error
	GetEventsByWorkflow(ctx context.Context, workflowID string) ([]*ExecutionEvent, error)
	GetEventsByTask(ctx context.Context, workflowID, taskID string) ([]*ExecutionEvent, error)
	GetEventsSince(ctx context.Context, workflowID string, since time.Time) ([]*ExecutionEvent, error)
}

// EventToJSON converts an event to JSON for transmission.
func EventToJSON(event *ExecutionEvent) ([]byte, error) {
	return json.Marshal(event)
}

// EventFromJSON reconstructs an event from JSON.
func EventFromJSON(data []byte) (*ExecutionEvent, error) {
	event := &ExecutionEvent{}
	if err := json.Unmarshal(data, event); err != nil {
		return nil, err
	}
	return event, nil
}
