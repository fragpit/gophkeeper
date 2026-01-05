package retry

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Operation is a function that performs an operation that may fail and need to be retried.
type Operation func(ctx context.Context) error

// IsRetriableFunc is a function that determines if an error should trigger a retry.
type IsRetriableFunc func(err error) bool

// Retrier provides retry functionality for operations that may fail transiently.
// It uses a configurable backoff strategy and custom logic to determine which errors are retriable.
type Retrier struct {
	backoff     []time.Duration
	IsRetriable IsRetriableFunc
}

// Option is a function that configures a Retrier.
type Option func(*Retrier)

// WithBackoff returns an Option that sets custom backoff durations for retry attempts.
// The durations slice specifies the wait time before each retry attempt.
func WithBackoff(durations []time.Duration) Option {
	return func(r *Retrier) {
		r.backoff = durations
	}
}

// New creates a new Retrier with the given IsRetriable function and optional configuration.
// By default, it uses a backoff strategy of 1s, 3s, and 5s between retry attempts.
// Additional options can be provided to customize the behavior.
func New(IsRetriable IsRetriableFunc, opts ...Option) *Retrier {
	r := &Retrier{
		backoff: []time.Duration{
			1 * time.Second,
			3 * time.Second,
			5 * time.Second,
		},
		IsRetriable: IsRetriable,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// Do executes the given operation with retry logic.
// It attempts the operation immediately, and if it fails with a retriable error,
// retries according to the configured backoff strategy.
// Returns nil on success, or the last error if all retry attempts are exhausted.
func (r *Retrier) Do(ctx context.Context, op Operation) error {
	var lastErr error
	err := op(ctx)
	if err == nil {
		return nil
	}

	if !r.IsRetriable(err) {
		return err
	}
	lastErr = err

	for _, t := range r.backoff {
		log.Printf("operation error, retrying in %v", t)
		time.Sleep(t)

		err = op(ctx)
		if err == nil {
			return nil
		}
		if !r.IsRetriable(err) {
			return err
		}
		lastErr = err
	}
	return fmt.Errorf("operation failed after retries: %w", lastErr)
}
