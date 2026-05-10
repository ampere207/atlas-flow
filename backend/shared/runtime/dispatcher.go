package runtime

import (
	"context"
	"time"
)

// TaskDispatcher queues ready tasks for worker execution.
// Uses distributed locks to ensure only one worker claims each task.
type TaskDispatcher interface {
	// DispatchTask makes a task available for workers to claim.
	DispatchTask(ctx context.Context, taskID string, payload map[string]interface{}, timeout time.Duration) error

	// ClaimTask atomically claims a task for a worker. Returns nil if claimed successfully.
	ClaimTask(ctx context.Context, taskID string, workerID string, leaseTTL time.Duration) (bool, error)

	// ReleaseClaim releases a task claim if it's held by the same worker.
	ReleaseClaim(ctx context.Context, taskID string, workerID string) error

	// ListUnclaimedTasks returns tasks available for claiming.
	ListUnclaimedTasks(ctx context.Context, limit int) ([]string, error)

	// MarkTaskAvailable schedules a task to become available at a specific time.
	MarkTaskAvailable(ctx context.Context, taskID string, availableAt time.Time) error
}

// InMemoryTaskDispatcher is a simple in-process implementation (for testing/dev).
// Production systems should use RedisTaskDispatcher.
type InMemoryTaskDispatcher struct {
	claims map[string]string    // taskID -> workerID
	queue  map[string]time.Time // taskID -> availableAt
}

// NewInMemoryTaskDispatcher creates a simple dispatcher for development.
func NewInMemoryTaskDispatcher() *InMemoryTaskDispatcher {
	return &InMemoryTaskDispatcher{
		claims: make(map[string]string),
		queue:  make(map[string]time.Time),
	}
}

// DispatchTask adds a task to the queue.
func (d *InMemoryTaskDispatcher) DispatchTask(ctx context.Context, taskID string, payload map[string]interface{}, timeout time.Duration) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	d.queue[taskID] = time.Now().Add(timeout)
	return nil
}

// ClaimTask attempts to claim a task.
func (d *InMemoryTaskDispatcher) ClaimTask(ctx context.Context, taskID string, workerID string, leaseTTL time.Duration) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	if _, exists := d.claims[taskID]; exists {
		return false, nil // Already claimed
	}
	d.claims[taskID] = workerID
	return true, nil
}

// ReleaseClaim releases a claim.
func (d *InMemoryTaskDispatcher) ReleaseClaim(ctx context.Context, taskID string, workerID string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if d.claims[taskID] == workerID {
		delete(d.claims, taskID)
	}
	return nil
}

// ListUnclaimedTasks returns tasks not yet claimed.
func (d *InMemoryTaskDispatcher) ListUnclaimedTasks(ctx context.Context, limit int) ([]string, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	unclaimed := make([]string, 0)
	now := time.Now()
	for taskID, availableAt := range d.queue {
		if _, claimed := d.claims[taskID]; !claimed && availableAt.Before(now) && len(unclaimed) < limit {
			unclaimed = append(unclaimed, taskID)
		}
	}
	return unclaimed, nil
}

// MarkTaskAvailable schedules when a task becomes claimable.
func (d *InMemoryTaskDispatcher) MarkTaskAvailable(ctx context.Context, taskID string, availableAt time.Time) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	d.queue[taskID] = availableAt
	return nil
}

// TaskClaimLease represents a worker's claim on a task.
type TaskClaimLease struct {
	TaskID    string
	WorkerID  string
	ClaimedAt time.Time
	ExpiresAt time.Time
	LeaseKey  string
}

// IsExpired checks if the lease has expired.
func (lease *TaskClaimLease) IsExpired() bool {
	return time.Now().After(lease.ExpiresAt)
}

// RemainingTime returns how long until the lease expires.
func (lease *TaskClaimLease) RemainingTime() time.Duration {
	remaining := time.Until(lease.ExpiresAt)
	if remaining < 0 {
		return 0
	}
	return remaining
}
