package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/nats-io/nats.go"

	sharedruntime "atlasflow/backend/shared/runtime"
)

// EventConsumer processes orchestration events from NATS.
type EventConsumer struct {
	nc       *nats.Conn
	handlers map[string]EventConsumerHandler
}

// EventConsumerHandler processes an event.
type EventConsumerHandler func(ctx context.Context, event *sharedruntime.ExecutionEvent) error

// NewEventConsumer creates an event consumer.
func NewEventConsumer(nc *nats.Conn) *EventConsumer {
	return &EventConsumer{
		nc:       nc,
		handlers: make(map[string]EventConsumerHandler),
	}
}

// RegisterHandler registers a handler for an event type.
func (ec *EventConsumer) RegisterHandler(eventType sharedruntime.ExecutionEventType,
	handler EventConsumerHandler) {
	ec.handlers[string(eventType)] = handler
}

// Start begins consuming events from NATS.
func (ec *EventConsumer) Start(ctx context.Context) error {
	// Subscribe to workflow events
	if _, err := ec.nc.ChanSubscribe("workflow.*", ec.createMessageHandler(ctx)); err != nil {
		return err
	}

	// Subscribe to task events
	if _, err := ec.nc.ChanSubscribe("task.*", ec.createMessageHandler(ctx)); err != nil {
		return err
	}

	// Subscribe to worker events
	if _, err := ec.nc.ChanSubscribe("worker.*", ec.createMessageHandler(ctx)); err != nil {
		return err
	}

	return nil
}

// createMessageHandler creates a message handler for NATS messages.
func (ec *EventConsumer) createMessageHandler(ctx context.Context) chan *nats.Msg {
	msgChan := make(chan *nats.Msg, 100)
	go ec.handleMessages(ctx, msgChan)
	return msgChan
}

// handleMessages processes incoming NATS messages.
func (ec *EventConsumer) handleMessages(ctx context.Context, msgChan chan *nats.Msg) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-msgChan:
			if msg == nil {
				continue
			}
			ec.handleMessage(ctx, msg)
		}
	}
}

// handleMessage processes a single NATS message.
func (ec *EventConsumer) handleMessage(ctx context.Context, msg *nats.Msg) {
	event := &sharedruntime.ExecutionEvent{}
	if err := json.Unmarshal(msg.Data, event); err != nil {
		log.Printf("Failed to unmarshal event: %v", err)
		return
	}

	// Find and execute handler
	handler, exists := ec.handlers[event.EventType]
	if !exists {
		// No specific handler, try wildcard handler
		if wildcard, hasWildcard := ec.handlers["*"]; hasWildcard {
			handler = wildcard
		} else {
			return // No handler for this event
		}
	}

	if err := handler(ctx, event); err != nil {
		log.Printf("Event handler error for %s: %v", event.EventType, err)
	}
}

// StateTransitionManager handles state transitions triggered by events.
type StateTransitionManager struct {
	workflowRepo interface{} // WorkflowRepository
	eventBus     sharedruntime.EventPublisher
}

// NewStateTransitionManager creates a transition manager.
func NewStateTransitionManager(workflowRepo interface{}, eventBus sharedruntime.EventPublisher) *StateTransitionManager {
	return &StateTransitionManager{
		workflowRepo: workflowRepo,
		eventBus:     eventBus,
	}
}

// OnTaskCompleted handles task completion and advances DAG execution.
func (stm *StateTransitionManager) OnTaskCompleted(ctx context.Context, event *sharedruntime.ExecutionEvent) error {
	log.Printf("Task %s completed in workflow %s", event.TaskID, event.WorkflowID)

	// Publish state transition event
	if stm.eventBus != nil {
		transitionEvent := sharedruntime.NewEventBuilder(sharedruntime.EventTaskCompleted).
			WorkflowID(event.WorkflowID).
			TaskID(event.TaskID).
			UserID(event.UserID).
			Data("source_event", event.EventID).
			Build()
		_ = stm.eventBus.PublishEvent(ctx, transitionEvent)
	}

	return nil
}

// OnTaskFailed handles task failures and determines next action.
func (stm *StateTransitionManager) OnTaskFailed(ctx context.Context, event *sharedruntime.ExecutionEvent) error {
	log.Printf("Task %s failed in workflow %s: %s", event.TaskID, event.WorkflowID, event.ErrorMessage)

	// Publish state transition event
	if stm.eventBus != nil {
		transitionEvent := sharedruntime.NewEventBuilder(sharedruntime.EventTaskFailed).
			WorkflowID(event.WorkflowID).
			TaskID(event.TaskID).
			UserID(event.UserID).
			Error(event.ErrorMessage).
			Data("source_event", event.EventID).
			Build()
		_ = stm.eventBus.PublishEvent(ctx, transitionEvent)
	}

	return nil
}

// OnTaskRetrying handles task retry scheduling.
func (stm *StateTransitionManager) OnTaskRetrying(ctx context.Context, event *sharedruntime.ExecutionEvent) error {
	log.Printf("Task %s retrying in workflow %s (attempt %v)", event.TaskID, event.WorkflowID, event.Data["attempt"])

	// Publish retry event
	if stm.eventBus != nil {
		retryEvent := sharedruntime.NewEventBuilder(sharedruntime.EventTaskRetrying).
			WorkflowID(event.WorkflowID).
			TaskID(event.TaskID).
			UserID(event.UserID).
			Data("source_event", event.EventID).
			Data("attempt", event.Data["attempt"]).
			Build()
		_ = stm.eventBus.PublishEvent(ctx, retryEvent)
	}

	return nil
}

// OnWorkerHeartbeat processes worker heartbeats for health tracking.
func (stm *StateTransitionManager) OnWorkerHeartbeat(ctx context.Context, event *sharedruntime.ExecutionEvent) error {
	// Log worker is alive
	log.Printf("Worker %s heartbeat received", event.WorkerID)
	return nil
}

// OnRecoveryStarted handles failure recovery.
func (stm *StateTransitionManager) OnRecoveryStarted(ctx context.Context, event *sharedruntime.ExecutionEvent) error {
	reason := ""
	if v, ok := event.Data["reason"]; ok {
		reason = fmt.Sprintf("%v", v)
	}
	log.Printf("Recovery started for task %s: %s", event.TaskID, reason)

	// Publish recovery event
	if stm.eventBus != nil {
		recoveryEvent := sharedruntime.NewEventBuilder(sharedruntime.ExecutionEventRecoveryStarted).
			WorkflowID(event.WorkflowID).
			TaskID(event.TaskID).
			UserID(event.UserID).
			Data("source_event", event.EventID).
			Data("reason", reason).
			Build()
		_ = stm.eventBus.PublishEvent(ctx, recoveryEvent)
	}

	return nil
}

// WorkflowProgressTracker tracks overall workflow progress.
type WorkflowProgressTracker struct {
	progressMap map[string]*sharedruntime.ExecutionProgress
	eventBus    sharedruntime.EventPublisher
}

// NewWorkflowProgressTracker creates a progress tracker.
func NewWorkflowProgressTracker(eventBus sharedruntime.EventPublisher) *WorkflowProgressTracker {
	return &WorkflowProgressTracker{
		progressMap: make(map[string]*sharedruntime.ExecutionProgress),
		eventBus:    eventBus,
	}
}

// GetProgress retrieves or creates progress for a workflow.
func (wpt *WorkflowProgressTracker) GetProgress(workflowID string, taskCount int) *sharedruntime.ExecutionProgress {
	if progress, exists := wpt.progressMap[workflowID]; exists {
		return progress
	}
	progress := sharedruntime.NewExecutionProgress(workflowID, taskCount)
	wpt.progressMap[workflowID] = progress
	return progress
}

// UpdateProgress updates progress with a task state change.
func (wpt *WorkflowProgressTracker) UpdateProgress(ctx context.Context, workflowID, taskID string,
	newState sharedruntime.TaskState) error {
	progress := wpt.GetProgress(workflowID, 10) // TODO: Get actual task count
	if err := progress.UpdateTaskState(taskID, newState); err != nil {
		return err
	}

	// Publish progress update
	if wpt.eventBus != nil {
		progressEvent := sharedruntime.NewEventBuilder(sharedruntime.EventWorkflowRunning).
			WorkflowID(workflowID).
			Data("progress_updated_at", progress.LastUpdated).
			Build()
		_ = wpt.eventBus.PublishEvent(ctx, progressEvent)
	}

	return nil
}

// NATSEventPublisher publishes events to NATS.
type NATSEventPublisher struct {
	nc *nats.Conn
}

// NewNATSEventPublisher creates a NATS event publisher.
func NewNATSEventPublisher(nc *nats.Conn) *NATSEventPublisher {
	return &NATSEventPublisher{nc: nc}
}

// PublishEvent publishes an event to NATS.
func (nep *NATSEventPublisher) PublishEvent(ctx context.Context, event *sharedruntime.ExecutionEvent) error {
	// Derive subject from event type
	subject := fmt.Sprintf("%s.%s", getEventCategory(event.EventType), event.EventType)

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return nep.nc.Publish(subject, data)
}

// PublishEventBatch publishes multiple events.
func (nep *NATSEventPublisher) PublishEventBatch(ctx context.Context, events []*sharedruntime.ExecutionEvent) error {
	for _, event := range events {
		if err := nep.PublishEvent(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

// getEventCategory derives the category from event type.
func getEventCategory(eventType string) string {
	if len(eventType) > 6 && eventType[:7] == "workflow" {
		return "workflow"
	}
	if len(eventType) > 4 && eventType[:4] == "task" {
		return "task"
	}
	if len(eventType) > 6 && eventType[:6] == "worker" {
		return "worker"
	}
	return "system"
}
