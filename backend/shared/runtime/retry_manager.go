package runtime

import (
	"fmt"
	"math"
	"time"
)

// RetryManager handles retry logic with exponential backoff.
type RetryManager struct {
	policy RetryPolicy
}

// NewRetryManager creates a new retry manager with a policy.
func NewRetryManager(policy RetryPolicy) *RetryManager {
	return &RetryManager{
		policy: policy,
	}
}

// CalculateBackoff returns the duration to wait before retrying.
// Uses exponential backoff: backoff = initial * (multiplier ^ attempt)
// Capped at maxBackoff.
func (rm *RetryManager) CalculateBackoff(attemptNumber int) time.Duration {
	if rm.policy.InitialBackoff == 0 {
		rm.policy.InitialBackoff = 1 * time.Second
	}
	if rm.policy.MaxBackoff == 0 {
		rm.policy.MaxBackoff = 5 * time.Minute
	}
	if rm.policy.BackoffMultiplier == 0 {
		rm.policy.BackoffMultiplier = 2.0
	}

	// Exponential: initial * (multiplier ^ (attempt - 1))
	backoff := float64(rm.policy.InitialBackoff.Milliseconds()) *
		math.Pow(rm.policy.BackoffMultiplier, float64(attemptNumber-1))

	// Cap at maxBackoff
	maxMS := float64(rm.policy.MaxBackoff.Milliseconds())
	if backoff > maxMS {
		backoff = maxMS
	}

	return time.Duration(int64(backoff)) * time.Millisecond
}

// CanRetry checks if a task should be retried.
func (rm *RetryManager) CanRetry(currentAttempt int, lastError string) bool {
	// Check max attempts
	if rm.policy.MaxAttempts > 0 && currentAttempt >= rm.policy.MaxAttempts {
		return false
	}

	// If no retryable statuses specified, retry all
	if len(rm.policy.RetryableStatus) == 0 {
		return true
	}

	// Check if error matches retryable statuses
	for _, status := range rm.policy.RetryableStatus {
		if lastError == status {
			return true
		}
	}

	return false
}

// NextRetryTime calculates when a task should be retried.
func (rm *RetryManager) NextRetryTime(attemptNumber int) time.Time {
	backoff := rm.CalculateBackoff(attemptNumber)
	return time.Now().Add(backoff)
}

// RetryDecision encapsulates the decision to retry or fail.
type RetryDecision struct {
	ShouldRetry       bool
	NextAttemptTime   time.Time
	RemainingAttempts int
	Reason            string
}

// MakeRetryDecision determines whether to retry and when.
func (rm *RetryManager) MakeRetryDecision(currentAttempt int, lastError string, maxAttempts int) RetryDecision {
	if !rm.CanRetry(currentAttempt, lastError) {
		return RetryDecision{
			ShouldRetry:       false,
			RemainingAttempts: 0,
			Reason:            fmt.Sprintf("Max retries exceeded (%d/%d)", currentAttempt, maxAttempts),
		}
	}

	nextTime := rm.NextRetryTime(currentAttempt)
	remaining := maxAttempts - currentAttempt

	return RetryDecision{
		ShouldRetry:       true,
		NextAttemptTime:   nextTime,
		RemainingAttempts: remaining,
		Reason: fmt.Sprintf("Retry scheduled after %v (attempt %d/%d)",
			time.Until(nextTime), currentAttempt, maxAttempts),
	}
}

// ExponentialBackoffSeries generates a series of backoff durations for visualization.
func (rm *RetryManager) ExponentialBackoffSeries(attempts int) []time.Duration {
	series := make([]time.Duration, attempts)
	for i := 0; i < attempts; i++ {
		series[i] = rm.CalculateBackoff(i + 1)
	}
	return series
}
