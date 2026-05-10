package models

import "time"

// APIResponse is the standard API response format
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// User represents a system user
type User struct {
	ID        string    `json:"id" db:"id"`
	Email     string    `json:"email" db:"email"`
	FullName  string    `json:"full_name" db:"full_name"`
	Password  string    `json:"-" db:"password_hash"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// RefreshToken stores user refresh tokens
type RefreshToken struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	Token     string    `json:"token" db:"token"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Workflow represents a distributed workflow
type Workflow struct {
	ID         string    `json:"id" db:"id"`
	UserID     string    `json:"user_id" db:"user_id"`
	Name       string    `json:"name" db:"name"`
	Status     string    `json:"status" db:"status"`
	Metadata   string    `json:"metadata" db:"metadata"`
	Definition string    `json:"definition,omitempty" db:"definition"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// Task represents an atomic execution unit inside a workflow DAG.
type Task struct {
	ID               string     `json:"id" db:"id"`
	WorkflowID       string     `json:"workflow_id" db:"workflow_id"`
	TaskType         string     `json:"task_type" db:"task_type"`
	Name             string     `json:"name" db:"name"`
	Payload          string     `json:"payload" db:"payload"`
	State            string     `json:"state" db:"state"`
	AssignedWorkerID string     `json:"assigned_worker_id" db:"assigned_worker_id"`
	RetryCount       int        `json:"retry_count" db:"retry_count"`
	MaxRetries       int        `json:"max_retries" db:"max_retries"`
	DependsOn        string     `json:"depends_on" db:"depends_on"`
	AvailableAt      time.Time  `json:"available_at" db:"available_at"`
	StartedAt        *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt      *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	FailedAt         *time.Time `json:"failed_at,omitempty" db:"failed_at"`
	ErrorMessage     string     `json:"error_message,omitempty" db:"error_message"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
}

// TaskAttempt records one try of a task execution.
type TaskAttempt struct {
	ID            string     `json:"id" db:"id"`
	TaskID        string     `json:"task_id" db:"task_id"`
	WorkflowID    string     `json:"workflow_id" db:"workflow_id"`
	WorkerID      string     `json:"worker_id" db:"worker_id"`
	AttemptNumber int        `json:"attempt_number" db:"attempt_number"`
	State         string     `json:"state" db:"state"`
	StartedAt     time.Time  `json:"started_at" db:"started_at"`
	FinishedAt    *time.Time `json:"finished_at,omitempty" db:"finished_at"`
	ErrorMessage  string     `json:"error_message,omitempty" db:"error_message"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
}

// WorkflowTransition records state changes for workflows and tasks.
type WorkflowTransition struct {
	ID         string    `json:"id" db:"id"`
	WorkflowID string    `json:"workflow_id" db:"workflow_id"`
	TaskID     string    `json:"task_id,omitempty" db:"task_id"`
	EntityType string    `json:"entity_type" db:"entity_type"`
	FromState  string    `json:"from_state" db:"from_state"`
	ToState    string    `json:"to_state" db:"to_state"`
	Reason     string    `json:"reason,omitempty" db:"reason"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// WorkerHeartbeat stores worker liveness records.
type WorkerHeartbeat struct {
	ID         string    `json:"id" db:"id"`
	WorkerID   string    `json:"worker_id" db:"worker_id"`
	UserID     string    `json:"user_id" db:"user_id"`
	Status     string    `json:"status" db:"status"`
	RecordedAt time.Time `json:"recorded_at" db:"recorded_at"`
}

// ExecutionLog records orchestration events and task diagnostics.
type ExecutionLog struct {
	ID         string    `json:"id" db:"id"`
	WorkflowID string    `json:"workflow_id" db:"workflow_id"`
	TaskID     string    `json:"task_id,omitempty" db:"task_id"`
	WorkerID   string    `json:"worker_id,omitempty" db:"worker_id"`
	Level      string    `json:"level" db:"level"`
	Message    string    `json:"message" db:"message"`
	Metadata   string    `json:"metadata,omitempty" db:"metadata"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// WorkflowEvent represents workflow execution events
type WorkflowEvent struct {
	ID         string    `json:"id" db:"id"`
	WorkflowID string    `json:"workflow_id" db:"workflow_id"`
	EventType  string    `json:"event_type" db:"event_type"`
	Payload    string    `json:"payload" db:"payload"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// Worker represents a distributed execution node
type Worker struct {
	ID            string    `json:"id" db:"id"`
	UserID        string    `json:"user_id" db:"user_id"`
	Name          string    `json:"name" db:"name"`
	Status        string    `json:"status" db:"status"`
	LastHeartbeat time.Time `json:"last_heartbeat" db:"last_heartbeat"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// TokenClaims represents JWT token claims
type TokenClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	StandardClaims
}

// StandardClaims standard JWT claims
type StandardClaims struct {
	ExpiresAt int64 `json:"exp"`
	IssuedAt  int64 `json:"iat"`
}
