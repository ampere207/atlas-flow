package streaming

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"

	sharedruntime "atlasflow/backend/shared/runtime"
)

// ExecutionStream represents a real-time stream of execution events for a workflow.
type ExecutionStream struct {
	WorkflowID string
	UserID     string
	StreamID   string
	Events     chan *StreamEvent
	ctx        context.Context
	cancel     context.CancelFunc
}

// StreamEvent wraps an execution event with stream metadata.
type StreamEvent struct {
	StreamID    string                        `json:"stream_id"`
	SequenceNum int64                         `json:"sequence_num"`
	Event       *sharedruntime.ExecutionEvent `json:"event"`
	ServerTime  int64                         `json:"server_time_ms"`
}

// NewExecutionStream creates a new stream for a workflow.
func NewExecutionStream(workflowID, userID string) *ExecutionStream {
	ctx, cancel := context.WithCancel(context.Background())
	return &ExecutionStream{
		WorkflowID: workflowID,
		UserID:     userID,
		StreamID:   uuid.New().String(),
		Events:     make(chan *StreamEvent, 100),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Close closes the stream.
func (es *ExecutionStream) Close() {
	es.cancel()
	close(es.Events)
}

// Send sends an event to the stream.
func (es *ExecutionStream) Send(event *sharedruntime.ExecutionEvent) bool {
	select {
	case <-es.ctx.Done():
		return false
	case es.Events <- &StreamEvent{
		StreamID:    es.StreamID,
		SequenceNum: int64(len(es.Events)),
		Event:       event,
		ServerTime:  time.Now().UnixMilli(),
	}:
		return true
	}
}

// StreamManager manages execution streams for clients.
type StreamManager struct {
	streams map[string]*ExecutionStream
	mu      sync.RWMutex
}

// NewStreamManager creates a stream manager.
func NewStreamManager() *StreamManager {
	return &StreamManager{
		streams: make(map[string]*ExecutionStream),
	}
}

// CreateStream creates a new stream for a workflow.
func (sm *StreamManager) CreateStream(workflowID, userID string) *ExecutionStream {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	stream := NewExecutionStream(workflowID, userID)
	sm.streams[stream.StreamID] = stream
	return stream
}

// GetStream retrieves a stream by ID.
func (sm *StreamManager) GetStream(streamID string) *ExecutionStream {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.streams[streamID]
}

// CloseStream closes and removes a stream.
func (sm *StreamManager) CloseStream(streamID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if stream, exists := sm.streams[streamID]; exists {
		stream.Close()
		delete(sm.streams, streamID)
	}
}

// BroadcastEvent sends an event to all streams for a workflow.
func (sm *StreamManager) BroadcastEvent(workflowID string, event *sharedruntime.ExecutionEvent) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for _, stream := range sm.streams {
		if stream.WorkflowID == workflowID {
			stream.Send(event)
		}
	}
}

// SSEHandler handles Server-Sent Events streaming.
type SSEHandler struct {
	streamManager *StreamManager
}

// NewSSEHandler creates an SSE handler.
func NewSSEHandler(streamManager *StreamManager) *SSEHandler {
	return &SSEHandler{streamManager: streamManager}
}

// ServeHTTP handles SSE connections for execution streaming.
// Path: /api/workflows/{workflowID}/stream
func (h *SSEHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	workflowID := r.PathValue("workflowID")
	userID := r.Header.Get("X-User-ID")

	if workflowID == "" || userID == "" {
		http.Error(w, "missing workflow_id or user_id", http.StatusBadRequest)
		return
	}

	// Create execution stream
	stream := h.streamManager.CreateStream(workflowID, userID)
	defer h.streamManager.CloseStream(stream.StreamID)

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send initial connection event
	_, _ = fmt.Fprintf(w, "data: {\"type\":\"stream_connected\",\"stream_id\":\"%s\"}\n\n", stream.StreamID)
	flusher.Flush()

	// Stream events
	for {
		select {
		case <-r.Context().Done():
			return
		case event := <-stream.Events:
			if event == nil {
				return
			}

			eventJSON, _ := json.Marshal(event)
			_, _ = fmt.Fprintf(w, "data: %s\n\n", eventJSON)
			flusher.Flush()
		}
	}
}

// WebSocketHandler handles WebSocket connections for bidirectional execution streaming.
type WebSocketHandler struct {
	streamManager *StreamManager
}

// NewWebSocketHandler creates a WebSocket handler.
func NewWebSocketHandler(streamManager *StreamManager) *WebSocketHandler {
	return &WebSocketHandler{streamManager: streamManager}
}

// ExecutionStreamDashboard sends periodic execution status updates.
type ExecutionStreamDashboard struct {
	streamManager *StreamManager
	eventBus      sharedruntime.EventPublisher
	ticker        *time.Ticker
}

// NewExecutionStreamDashboard creates a dashboard streamer.
func NewExecutionStreamDashboard(streamManager *StreamManager, eventBus sharedruntime.EventPublisher) *ExecutionStreamDashboard {
	return &ExecutionStreamDashboard{
		streamManager: streamManager,
		eventBus:      eventBus,
		ticker:        time.NewTicker(5 * time.Second),
	}
}

// Start begins streaming dashboard updates.
func (esd *ExecutionStreamDashboard) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				esd.ticker.Stop()
				return
			case <-esd.ticker.C:
				// Periodic dashboard update logic here
			}
		}
	}()
}

// ExecutionTimelineBuilder constructs execution timelines for visualization.
type ExecutionTimelineBuilder struct {
	events []*sharedruntime.ExecutionEvent
	mu     sync.RWMutex
}

// NewExecutionTimelineBuilder creates a timeline builder.
func NewExecutionTimelineBuilder() *ExecutionTimelineBuilder {
	return &ExecutionTimelineBuilder{
		events: make([]*sharedruntime.ExecutionEvent, 0),
	}
}

// AddEvent adds an event to the timeline.
func (etb *ExecutionTimelineBuilder) AddEvent(event *sharedruntime.ExecutionEvent) {
	etb.mu.Lock()
	defer etb.mu.Unlock()
	etb.events = append(etb.events, event)
}

// GetTimeline retrieves the timeline.
func (etb *ExecutionTimelineBuilder) GetTimeline() []*sharedruntime.ExecutionEvent {
	etb.mu.RLock()
	defer etb.mu.RUnlock()

	timeline := make([]*sharedruntime.ExecutionEvent, len(etb.events))
	copy(timeline, etb.events)
	return timeline
}

// TimelineEntry represents a single execution event in the timeline.
type TimelineEntry struct {
	EventID      string `json:"event_id"`
	EventType    string `json:"event_type"`
	WorkflowID   string `json:"workflow_id"`
	TaskID       string `json:"task_id,omitempty"`
	WorkerID     string `json:"worker_id,omitempty"`
	Timestamp    int64  `json:"timestamp"`
	Duration     int64  `json:"duration_ms,omitempty"`
	Message      string `json:"message,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// BuildTimelineEntry converts an event to a timeline entry.
func BuildTimelineEntry(event *sharedruntime.ExecutionEvent) TimelineEntry {
	entry := TimelineEntry{
		EventID:      event.EventID,
		EventType:    event.EventType,
		WorkflowID:   event.WorkflowID,
		TaskID:       event.TaskID,
		WorkerID:     event.WorkerID,
		Timestamp:    event.Timestamp.UnixMilli(),
		ErrorMessage: event.ErrorMessage,
	}

	if duration, ok := event.Data["duration_ms"].(float64); ok {
		entry.Duration = int64(duration)
	}

	// Build friendly message
	switch event.EventType {
	case string(sharedruntime.EventTaskStarted):
		entry.Message = fmt.Sprintf("Task %s started", event.TaskID)
	case string(sharedruntime.EventTaskCompleted):
		entry.Message = fmt.Sprintf("Task %s completed", event.TaskID)
	case string(sharedruntime.EventTaskFailed):
		entry.Message = fmt.Sprintf("Task %s failed", event.TaskID)
	case string(sharedruntime.EventWorkflowCompleted):
		entry.Message = "Workflow completed"
	case string(sharedruntime.EventWorkerHeartbeat):
		entry.Message = fmt.Sprintf("Worker %s alive", event.WorkerID)
	default:
		entry.Message = event.EventType
	}

	return entry
}
