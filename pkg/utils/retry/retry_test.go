package retry

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	errRetriable    = errors.New("retriable error")
	errNonRetriable = errors.New("non-retriable error")
)

func alwaysRetriable(err error) bool {
	return true
}

func neverRetriable(err error) bool {
	return false
}

func selectiveRetriable(err error) bool {
	return errors.Is(err, errRetriable)
}

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		isRetriable IsRetriableFunc
		opts        []Option
		wantBackoff []time.Duration
	}{
		{
			name:        "default backoff",
			isRetriable: alwaysRetriable,
			opts:        nil,
			wantBackoff: []time.Duration{
				1 * time.Second,
				3 * time.Second,
				5 * time.Second,
			},
		},
		{
			name:        "custom backoff",
			isRetriable: alwaysRetriable,
			opts: []Option{
				WithBackoff(
					[]time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
				),
			},
			wantBackoff: []time.Duration{
				100 * time.Millisecond,
				200 * time.Millisecond,
			},
		},
		{
			name:        "empty backoff",
			isRetriable: alwaysRetriable,
			opts:        []Option{WithBackoff([]time.Duration{})},
			wantBackoff: []time.Duration{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retrier := New(tt.isRetriable, tt.opts...)

			assert.NotNil(t, retrier)
			assert.Equal(t, tt.wantBackoff, retrier.backoff)
			assert.NotNil(t, retrier.IsRetriable)
		})
	}
}

func TestRetrier_Do_Success(t *testing.T) {
	retrier := New(alwaysRetriable)
	ctx := context.Background()

	callCount := 0
	operation := func(ctx context.Context) error {
		callCount++
		return nil
	}

	err := retrier.Do(ctx, operation)

	assert.NoError(t, err)
	assert.Equal(t, 1, callCount)
}

func TestRetrier_Do_NonRetriableError(t *testing.T) {
	retrier := New(neverRetriable)
	ctx := context.Background()

	callCount := 0
	operation := func(ctx context.Context) error {
		callCount++
		return errNonRetriable
	}

	err := retrier.Do(ctx, operation)

	assert.Error(t, err)
	assert.ErrorIs(t, err, errNonRetriable)
	assert.Equal(t, 1, callCount)
}

func TestRetrier_Do_RetriableErrorThenSuccess(t *testing.T) {
	retrier := New(
		alwaysRetriable,
		WithBackoff([]time.Duration{1 * time.Millisecond, 2 * time.Millisecond}),
	)
	ctx := context.Background()

	callCount := 0
	operation := func(ctx context.Context) error {
		callCount++
		if callCount < 3 {
			return errRetriable
		}
		return nil
	}

	start := time.Now()
	err := retrier.Do(ctx, operation)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.Equal(t, 3, callCount)

	assert.GreaterOrEqual(t, duration, 3*time.Millisecond)
}

func TestRetrier_Do_RetriableErrorExhausted(t *testing.T) {
	retrier := New(
		alwaysRetriable,
		WithBackoff([]time.Duration{1 * time.Millisecond, 2 * time.Millisecond}),
	)
	ctx := context.Background()

	callCount := 0
	operation := func(ctx context.Context) error {
		callCount++
		return errRetriable
	}

	err := retrier.Do(ctx, operation)

	assert.Error(t, err)
	assert.ErrorIs(t, err, errRetriable)
	assert.Contains(t, err.Error(), "operation failed after retries")

	assert.Equal(t, 3, callCount)
}

func TestRetrier_Do_SelectiveRetriable(t *testing.T) {
	retrier := New(
		selectiveRetriable,
		WithBackoff([]time.Duration{1 * time.Millisecond}),
	)
	ctx := context.Background()

	tests := []struct {
		name      string
		errors    []error
		wantCalls int
		wantError error
	}{
		{
			name:      "retriable error then success",
			errors:    []error{errRetriable, nil},
			wantCalls: 2,
			wantError: nil,
		},
		{
			name:      "non-retriable error stops immediately",
			errors:    []error{errNonRetriable},
			wantCalls: 1,
			wantError: errNonRetriable,
		},
		{
			name:      "retriable then non-retriable",
			errors:    []error{errRetriable, errNonRetriable},
			wantCalls: 2,
			wantError: errNonRetriable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			operation := func(ctx context.Context) error {
				if callCount < len(tt.errors) {
					err := tt.errors[callCount]
					callCount++
					return err
				}
				callCount++
				return nil
			}

			err := retrier.Do(ctx, operation)

			if tt.wantError != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.wantError)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantCalls, callCount)
		})
	}
}

func TestRetrier_Do_ContextCancellation(t *testing.T) {
	retrier := New(
		alwaysRetriable,
		WithBackoff([]time.Duration{100 * time.Millisecond}),
	)

	ctx, cancel := context.WithCancel(context.Background())

	callCount := 0
	operation := func(ctx context.Context) error {
		callCount++
		if callCount == 1 {

			cancel()
			return errRetriable
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return errRetriable
		}
	}

	err := retrier.Do(ctx, operation)

	assert.Error(t, err)
	assert.GreaterOrEqual(t, callCount, 1)
}

func TestRetrier_Do_EmptyBackoff(t *testing.T) {
	retrier := New(alwaysRetriable, WithBackoff([]time.Duration{}))
	ctx := context.Background()

	callCount := 0
	operation := func(ctx context.Context) error {
		callCount++
		return errRetriable
	}

	err := retrier.Do(ctx, operation)

	assert.Error(t, err)
	assert.ErrorIs(t, err, errRetriable)

	assert.Equal(t, 1, callCount)
}

func TestRetrier_Do_PanicRecovery(t *testing.T) {
	retrier := New(alwaysRetriable)
	ctx := context.Background()

	operation := func(ctx context.Context) error {
		panic("test panic")
	}

	assert.Panics(t, func() {
		_ = retrier.Do(ctx, operation)
	})
}

func TestRetrier_Do_NilOperation(t *testing.T) {
	retrier := New(alwaysRetriable)
	ctx := context.Background()

	assert.Panics(t, func() {
		_ = retrier.Do(ctx, nil)
	})
}

func TestRetrier_Do_NilContext(t *testing.T) {
	retrier := New(alwaysRetriable)

	operation := func(ctx context.Context) error {
		return nil
	}

	err := retrier.Do(context.TODO(), operation)
	assert.NoError(t, err)
}

func BenchmarkRetrier_Do_Success(b *testing.B) {
	retrier := New(alwaysRetriable)
	ctx := context.Background()

	operation := func(ctx context.Context) error {
		return nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = retrier.Do(ctx, operation)
	}
}

func BenchmarkRetrier_Do_WithRetries(b *testing.B) {
	retrier := New(
		alwaysRetriable,
		WithBackoff([]time.Duration{1 * time.Nanosecond, 1 * time.Nanosecond}),
	)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		callCount := 0
		operation := func(ctx context.Context) error {
			callCount++
			if callCount < 3 {
				return errRetriable
			}
			return nil
		}
		_ = retrier.Do(ctx, operation)
	}
}

func ExampleRetrier_Do() {
	isRetriable := func(err error) bool {
		return errors.Is(err, errRetriable)
	}

	retrier := New(isRetriable, WithBackoff([]time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
	}))

	callCount := 0
	operation := func(ctx context.Context) error {
		callCount++
		if callCount < 2 {
			return errRetriable
		}
		return nil
	}

	ctx := context.Background()
	err := retrier.Do(ctx, operation)

	fmt.Printf("Error: %v, Calls: %d\n", err, callCount)
	// Output:
	// Error: <nil>, Calls: 2
}

func ExampleNew() {
	isRetriable := func(err error) bool {
		return true
	}

	retrier := New(isRetriable, WithBackoff([]time.Duration{
		50 * time.Millisecond,
		100 * time.Millisecond,
		200 * time.Millisecond,
	}))

	fmt.Printf("Backoff intervals: %v\n", retrier.backoff)
	// Output:
	// Backoff intervals: [50ms 100ms 200ms]
}

type mockDB struct {
	pingCount   int
	pingErrors  []error
	shouldPanic bool
}

func (m *mockDB) Ping(ctx context.Context) error {
	if m.shouldPanic {
		panic("database panic")
	}

	if m.pingCount < len(m.pingErrors) {
		err := m.pingErrors[m.pingCount]
		m.pingCount++
		return err
	}
	m.pingCount++
	return nil
}

var (
	errConnectionException  = errors.New("connection exception")
	errOperatorIntervention = errors.New("operator intervention")
	errInvalidSQLStatement  = errors.New("invalid SQL statement")
)

func postgresqlIsRetriable(err error) bool {
	return errors.Is(err, errConnectionException) ||
		errors.Is(err, errOperatorIntervention)
}

func TestRetrier_Do_DatabasePingSuccess(t *testing.T) {
	retrier := New(postgresqlIsRetriable)
	ctx := context.Background()

	db := &mockDB{}

	op := func(ctx context.Context) error {
		return db.Ping(ctx)
	}

	err := retrier.Do(ctx, op)

	assert.NoError(t, err)
	assert.Equal(t, 1, db.pingCount)
}

func TestRetrier_Do_DatabasePingRetriableError(t *testing.T) {
	retrier := New(
		postgresqlIsRetriable,
		WithBackoff([]time.Duration{1 * time.Millisecond, 2 * time.Millisecond}),
	)
	ctx := context.Background()

	db := &mockDB{
		pingErrors: []error{
			errConnectionException,
			errOperatorIntervention,
			nil,
		},
	}

	op := func(ctx context.Context) error {
		return db.Ping(ctx)
	}

	start := time.Now()
	err := retrier.Do(ctx, op)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.Equal(t, 3, db.pingCount)
	assert.GreaterOrEqual(t, duration, 3*time.Millisecond)
}

func TestRetrier_Do_DatabasePingNonRetriableError(t *testing.T) {
	retrier := New(postgresqlIsRetriable)
	ctx := context.Background()

	db := &mockDB{
		pingErrors: []error{errInvalidSQLStatement},
	}

	op := func(ctx context.Context) error {
		return db.Ping(ctx)
	}

	err := retrier.Do(ctx, op)

	assert.Error(t, err)
	assert.ErrorIs(t, err, errInvalidSQLStatement)
	assert.Equal(t, 1, db.pingCount)
}

func TestRetrier_Do_DatabasePingExhaustedRetries(t *testing.T) {
	retrier := New(
		postgresqlIsRetriable,
		WithBackoff([]time.Duration{1 * time.Millisecond, 2 * time.Millisecond}),
	)
	ctx := context.Background()

	db := &mockDB{
		pingErrors: []error{
			errConnectionException,
			errConnectionException,
			errConnectionException,
		},
	}

	op := func(ctx context.Context) error {
		return db.Ping(ctx)
	}

	err := retrier.Do(ctx, op)

	assert.Error(t, err)
	assert.ErrorIs(t, err, errConnectionException)
	assert.Contains(t, err.Error(), "operation failed after retries")
	assert.Equal(t, 3, db.pingCount)
}

func TestRetrier_Do_DatabasePingMixedErrors(t *testing.T) {
	retrier := New(
		postgresqlIsRetriable,
		WithBackoff([]time.Duration{1 * time.Millisecond}),
	)
	ctx := context.Background()

	db := &mockDB{
		pingErrors: []error{
			errConnectionException,
			errInvalidSQLStatement,
		},
	}

	op := func(ctx context.Context) error {
		return db.Ping(ctx)
	}

	err := retrier.Do(ctx, op)

	assert.Error(t, err)
	assert.ErrorIs(t, err, errInvalidSQLStatement)
	assert.Equal(t, 2, db.pingCount)
}

func TestRetrier_Do_DatabasePingContextTimeout(t *testing.T) {
	retrier := New(
		postgresqlIsRetriable,
		WithBackoff([]time.Duration{50 * time.Millisecond}),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	db := &mockDB{
		pingErrors: []error{errConnectionException},
	}

	op := func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return db.Ping(ctx)
		}
	}

	err := retrier.Do(ctx, op)

	assert.Error(t, err)

}

func TestRetrier_Do_DatabasePingPanic(t *testing.T) {
	retrier := New(postgresqlIsRetriable)
	ctx := context.Background()

	db := &mockDB{shouldPanic: true}

	op := func(ctx context.Context) error {
		return db.Ping(ctx)
	}

	assert.Panics(t, func() {
		_ = retrier.Do(ctx, op)
	})
}

func BenchmarkRetrier_Do_DatabasePingSuccess(b *testing.B) {
	retrier := New(postgresqlIsRetriable)
	ctx := context.Background()

	db := &mockDB{}

	op := func(ctx context.Context) error {
		return db.Ping(ctx)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.pingCount = 0
		_ = retrier.Do(ctx, op)
	}
}

func BenchmarkRetrier_Do_DatabasePingWithRetries(b *testing.B) {
	retrier := New(
		postgresqlIsRetriable,
		WithBackoff([]time.Duration{1 * time.Nanosecond, 1 * time.Nanosecond}),
	)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db := &mockDB{
			pingErrors: []error{
				errConnectionException,
				errConnectionException,
				nil,
			},
		}

		op := func(ctx context.Context) error {
			return db.Ping(ctx)
		}

		_ = retrier.Do(ctx, op)
	}
}

func ExampleRetrier_Do_databasePing() {
	isRetriable := func(err error) bool {
		return errors.Is(err, errConnectionException) ||
			errors.Is(err, errOperatorIntervention)
	}

	retrier := New(isRetriable)

	db := &mockDB{
		pingErrors: []error{
			errConnectionException,
			nil,
		},
	}

	op := func(ctx context.Context) error {
		return db.Ping(ctx)
	}

	ctx := context.Background()
	err := retrier.Do(ctx, op)

	fmt.Printf("Ping successful: %v, Attempts: %d\n", err == nil, db.pingCount)
	// Output:
	// Ping successful: true, Attempts: 2
}
