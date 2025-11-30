package retry

import (
	"context"
	"fmt"
	"log"
	"time"
)

type Operation func(ctx context.Context) error

type IsRetriableFunc func(err error) bool

type Retrier struct {
	backoff     []time.Duration
	IsRetriable IsRetriableFunc
}

type Option func(*Retrier)

func WithBackoff(durations []time.Duration) Option {
	return func(r *Retrier) {
		r.backoff = durations
	}
}

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
