package runtime

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// LeaseManager coordinates distributed ownership using Redis.
type LeaseManager struct {
	client *redis.Client
}

// NewLeaseManager creates a lease manager backed by Redis.
func NewLeaseManager(client *redis.Client) *LeaseManager {
	return &LeaseManager{client: client}
}

// Acquire tries to claim a lease for a task or worker.
func (manager *LeaseManager) Acquire(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return manager.client.SetNX(ctx, key, "1", ttl).Result()
}

// Release relinquishes a lease.
func (manager *LeaseManager) Release(ctx context.Context, key string) error {
	return manager.client.Del(ctx, key).Err()
}

// Extend renews a lease if it still exists.
func (manager *LeaseManager) Extend(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	exists, err := manager.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if exists == 0 {
		return false, nil
	}
	ok, err := manager.client.Expire(ctx, key, ttl).Result()
	return ok, err
}
