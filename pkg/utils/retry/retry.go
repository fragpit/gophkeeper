package retry

import (
	"context"
	"fmt"
	"iter"
	"time"
)

// Operation is a function that performs an operation that may fail and need to be retried.
type Operation func(ctx context.Context) error

// IsRetriableFunc is a function that determines if an error should trigger a retry.
type IsRetriableFunc func(err error) bool

// Retrier provides retry functionality for operations that may fail transiently.
// It uses a configurable backoff strategy and custom logic to determine which errors are retriable.
type Retrier struct {
	base        time.Duration
	maxRetries  int
	IsRetriable IsRetriableFunc
	logger      func(format string, args ...any)
}

// Option is a function that configures a Retrier.
type Option func(*Retrier)

// WithBaseDuration returns an Option that sets the base duration for exponential backoff calculations.
// The base duration is multiplied by powers of 2 for each retry attempt.
func WithBaseDuration(base time.Duration) Option {
	return func(r *Retrier) {
		r.base = base
	}
}

// WithMaxRetries returns an Option that sets the maximum number of retry attempts.
func WithMaxRetries(max int) Option {
	return func(r *Retrier) {
		r.maxRetries = max
	}
}

// WithLogger returns an Option that sets a logger for retry attempts.
// Pass nil to disable logging.
func WithLogger(logger func(format string, args ...any)) Option {
	return func(r *Retrier) {
		r.logger = logger
	}
}

// New creates a new Retrier with the given IsRetriable function and optional configuration.
// By default, it uses a backoff strategy of 1s, 3s, and 5s between retry attempts.
// Additional options can be provided to customize the behavior.
func New(IsRetriable IsRetriableFunc, opts ...Option) *Retrier {
	r := &Retrier{
		base:        1 * time.Second,
		maxRetries:  3,
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
// If the context is cancelled or times out, returns ctx.Err() immediately.
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

	for d := range ExponentialBackoff(r.base, r.maxRetries) {
		if r.logger != nil {
			r.logger("operation error, retrying in %v", d)
		}

		timer := time.NewTimer(d)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}

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

func ExponentialBackoff(
	base time.Duration,
	maxRetries int,
) iter.Seq[time.Duration] {
	return func(yield func(time.Duration) bool) {
		for i := range maxRetries {
			d := base * (1 << i)
			if !yield(d) {
				return
			}
		}
	}
}
