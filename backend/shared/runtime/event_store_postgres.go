package runtime

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

// PostgresEventStore implements EventStore using PostgreSQL.
type PostgresEventStore struct {
	db *sql.DB
}

// NewPostgresEventStore creates a new PostgresEventStore.
func NewPostgresEventStore(db *sql.DB) *PostgresEventStore {
	return &PostgresEventStore{db: db}
}

// StoreEvent persists an execution event.
func (s *PostgresEventStore) StoreEvent(ctx context.Context, event *ExecutionEvent) error {
	payload, err := json.Marshal(event.Data)
	if err != nil {
		payload = []byte("{}")
	}

	var query string
	var args []interface{}

	if event.EventID != "" {
		query = `
			INSERT INTO workflow_events (
				id, workflow_id, task_id, worker_id, user_id, event_type, payload, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`
		args = append(args, event.EventID)
	} else {
		query = `
			INSERT INTO workflow_events (
				workflow_id, task_id, worker_id, user_id, event_type, payload, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
		`
	}

	var workflowID, taskID, userID sql.NullString
	if event.WorkflowID != "" {
		workflowID = sql.NullString{String: event.WorkflowID, Valid: true}
	}
	if event.TaskID != "" {
		taskID = sql.NullString{String: event.TaskID, Valid: true}
	}
	if event.UserID != "" {
		userID = sql.NullString{String: event.UserID, Valid: true}
	}

	args = append(args,
		workflowID,
		taskID,
		event.WorkerID,
		userID,
		event.EventType,
		payload,
		event.Timestamp,
	)

	_, err = s.db.ExecContext(ctx, query, args...)

	return err
}

// GetEventsByWorkflow retrieves all events for a workflow.
func (s *PostgresEventStore) GetEventsByWorkflow(ctx context.Context, workflowID string) ([]*ExecutionEvent, error) {
	query := `
		SELECT id, workflow_id, task_id, worker_id, user_id, event_type, payload, created_at
		FROM workflow_events
		WHERE workflow_id = $1
		ORDER BY created_at ASC
	`

	rows, err := s.db.QueryContext(ctx, query, workflowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*ExecutionEvent
	for rows.Next() {
		event := &ExecutionEvent{}
		var eID, wID, tID, uID sql.NullString
		var payload []byte

		err := rows.Scan(
			&eID,
			&wID,
			&tID,
			&event.WorkerID,
			&uID,
			&event.EventType,
			&payload,
			&event.Timestamp,
		)
		if err != nil {
			return nil, err
		}

		if eID.Valid {
			event.EventID = eID.String
		}
		if wID.Valid {
			event.WorkflowID = wID.String
		}
		if tID.Valid {
			event.TaskID = tID.String
		}
		if uID.Valid {
			event.UserID = uID.String
		}

		err = json.Unmarshal(payload, &event.Data)
		if err != nil {
			event.Data = make(map[string]interface{})
		}

		events = append(events, event)
	}

	return events, nil
}

// GetEventsByTask retrieves events for a specific task.
func (s *PostgresEventStore) GetEventsByTask(ctx context.Context, workflowID, taskID string) ([]*ExecutionEvent, error) {
	query := `
		SELECT id, workflow_id, task_id, worker_id, user_id, event_type, payload, created_at
		FROM workflow_events
		WHERE workflow_id = $1 AND task_id = $2
		ORDER BY created_at ASC
	`

	rows, err := s.db.QueryContext(ctx, query, workflowID, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*ExecutionEvent
	for rows.Next() {
		event := &ExecutionEvent{}
		var eID, wID, tID, uID sql.NullString
		var payload []byte

		err := rows.Scan(
			&eID,
			&wID,
			&tID,
			&event.WorkerID,
			&uID,
			&event.EventType,
			&payload,
			&event.Timestamp,
		)
		if err != nil {
			return nil, err
		}

		if eID.Valid {
			event.EventID = eID.String
		}
		if wID.Valid {
			event.WorkflowID = wID.String
		}
		if tID.Valid {
			event.TaskID = tID.String
		}
		if uID.Valid {
			event.UserID = uID.String
		}

		err = json.Unmarshal(payload, &event.Data)
		if err != nil {
			event.Data = make(map[string]interface{})
		}

		events = append(events, event)
	}

	return events, nil
}

// GetEventsSince retrieves events after a certain time.
func (s *PostgresEventStore) GetEventsSince(ctx context.Context, workflowID string, since time.Time) ([]*ExecutionEvent, error) {
	query := `
		SELECT id, workflow_id, task_id, worker_id, user_id, event_type, payload, created_at
		FROM workflow_events
		WHERE workflow_id = $1 AND created_at > $2
		ORDER BY created_at ASC
	`

	rows, err := s.db.QueryContext(ctx, query, workflowID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*ExecutionEvent
	for rows.Next() {
		event := &ExecutionEvent{}
		var eID, wID, tID, uID sql.NullString
		var payload []byte

		err := rows.Scan(
			&eID,
			&wID,
			&tID,
			&event.WorkerID,
			&uID,
			&event.EventType,
			&payload,
			&event.Timestamp,
		)
		if err != nil {
			return nil, err
		}

		if eID.Valid {
			event.EventID = eID.String
		}
		if wID.Valid {
			event.WorkflowID = wID.String
		}
		if tID.Valid {
			event.TaskID = tID.String
		}
		if uID.Valid {
			event.UserID = uID.String
		}

		err = json.Unmarshal(payload, &event.Data)
		if err != nil {
			event.Data = make(map[string]interface{})
		}

		events = append(events, event)
	}

	return events, nil
}

// StoreEventBatch stores multiple events in a transaction.
func (s *PostgresEventStore) StoreEventBatch(ctx context.Context, events []*ExecutionEvent) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	for _, event := range events {
		if err := s.StoreEvent(ctx, event); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}
