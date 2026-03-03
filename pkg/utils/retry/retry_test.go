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
		name           string
		isRetriable    IsRetriableFunc
		opts           []Option
		wantBase       time.Duration
		wantMaxRetries int
	}{
		{
			name:           "default backoff",
			isRetriable:    alwaysRetriable,
			opts:           nil,
			wantBase:       1 * time.Second,
			wantMaxRetries: 3,
		},
		{
			name:        "custom base duration",
			isRetriable: alwaysRetriable,
			opts: []Option{
				WithBaseDuration(100 * time.Millisecond),
			},
			wantBase:       100 * time.Millisecond,
			wantMaxRetries: 3,
		},
		{
			name:           "custom max retries",
			isRetriable:    alwaysRetriable,
			opts:           []Option{WithMaxRetries(5)},
			wantBase:       1 * time.Second,
			wantMaxRetries: 5,
		},
		{
			name:           "zero retries",
			isRetriable:    alwaysRetriable,
			opts:           []Option{WithMaxRetries(0)},
			wantBase:       1 * time.Second,
			wantMaxRetries: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retrier := New(tt.isRetriable, tt.opts...)

			assert.NotNil(t, retrier)
			assert.Equal(t, tt.wantBase, retrier.base)
			assert.Equal(t, tt.wantMaxRetries, retrier.maxRetries)
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
		WithBaseDuration(1*time.Millisecond),
		WithMaxRetries(2),
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

	assert.GreaterOrEqual(t, duration, 1*time.Millisecond)
}

func TestRetrier_Do_RetriableErrorExhausted(t *testing.T) {
	retrier := New(
		alwaysRetriable,
		WithBaseDuration(1*time.Millisecond),
		WithMaxRetries(2),
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

	// 1 первая попытка + 2 maxRetries = 3 вызова
	assert.Equal(t, 3, callCount)
}

func TestRetrier_Do_SelectiveRetriable(t *testing.T) {
	retrier := New(
		selectiveRetriable,
		WithBaseDuration(1*time.Millisecond),
		WithMaxRetries(1),
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
		WithBaseDuration(100*time.Millisecond),
		WithMaxRetries(1),
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
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, callCount)
}

func TestRetrier_Do_ContextCancelDuringSleep(t *testing.T) {
	retrier := New(
		alwaysRetriable,
		WithBaseDuration(1*time.Second),
		WithMaxRetries(1),
	)

	ctx, cancel := context.WithCancel(context.Background())

	callCount := 0
	operation := func(ctx context.Context) error {
		callCount++
		return errRetriable
	}

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err := retrier.Do(ctx, operation)
	duration := time.Since(start)

	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, callCount)
	assert.Less(t, duration, 500*time.Millisecond)
}

func TestRetrier_Do_ZeroRetries(t *testing.T) {
	retrier := New(alwaysRetriable, WithMaxRetries(0))
	ctx := context.Background()

	callCount := 0
	operation := func(ctx context.Context) error {
		callCount++
		return errRetriable
	}

	err := retrier.Do(ctx, operation)

	assert.Error(t, err)
	assert.ErrorIs(t, err, errRetriable)

	// только первая попытка, повторов нет
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
		WithBaseDuration(1*time.Nanosecond),
		WithMaxRetries(2),
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

	retrier := New(
		isRetriable,
		WithBaseDuration(100*time.Millisecond),
		WithMaxRetries(2),
	)

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

	retrier := New(
		isRetriable,
		WithBaseDuration(50*time.Millisecond),
		WithMaxRetries(3),
	)

	fmt.Printf("Base: %v, MaxRetries: %d\n", retrier.base, retrier.maxRetries)
	// Output:
	// Base: 50ms, MaxRetries: 3
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
		WithBaseDuration(1*time.Millisecond),
		WithMaxRetries(2),
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
	assert.GreaterOrEqual(t, duration, 1*time.Millisecond)
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
		WithBaseDuration(1*time.Millisecond),
		WithMaxRetries(2),
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
	// 1 первая попытка + 2 maxRetries = 3 вызова
	assert.Equal(t, 3, db.pingCount)
}

func TestRetrier_Do_DatabasePingMixedErrors(t *testing.T) {
	retrier := New(
		postgresqlIsRetriable,
		WithBaseDuration(1*time.Millisecond),
		WithMaxRetries(1),
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
		WithBaseDuration(50*time.Millisecond),
		WithMaxRetries(1),
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

	start := time.Now()
	err := retrier.Do(ctx, op)
	duration := time.Since(start)

	assert.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, duration, 50*time.Millisecond)
	assert.Equal(t, 1, db.pingCount)

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
		WithBaseDuration(1*time.Nanosecond),
		WithMaxRetries(2),
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

	retrier := New(isRetriable, WithMaxRetries(1))

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

func TestDoWithResult_Success(t *testing.T) {
	retrier := New(alwaysRetriable)
	ctx := context.Background()

	callCount := 0
	result, err := DoWithResult(
		ctx,
		retrier,
		func(ctx context.Context) (int, error) {
			callCount++
			return 42, nil
		},
	)

	assert.NoError(t, err)
	assert.Equal(t, 42, result)
	assert.Equal(t, 1, callCount)
}

func TestDoWithResult_NonRetriableError(t *testing.T) {
	retrier := New(neverRetriable)
	ctx := context.Background()

	callCount := 0
	result, err := DoWithResult(
		ctx,
		retrier,
		func(ctx context.Context) (string, error) {
			callCount++
			return "", errNonRetriable
		},
	)

	assert.Error(t, err)
	assert.ErrorIs(t, err, errNonRetriable)
	assert.Equal(t, "", result)
	assert.Equal(t, 1, callCount)
}

func TestDoWithResult_RetriableErrorThenSuccess(t *testing.T) {
	retrier := New(
		alwaysRetriable,
		WithBaseDuration(1*time.Millisecond),
		WithMaxRetries(2),
	)
	ctx := context.Background()

	callCount := 0
	result, err := DoWithResult(
		ctx,
		retrier,
		func(ctx context.Context) (int, error) {
			callCount++
			if callCount < 3 {
				return 0, errRetriable
			}
			return 99, nil
		},
	)

	assert.NoError(t, err)
	assert.Equal(t, 99, result)
	assert.Equal(t, 3, callCount)
}

func TestDoWithResult_ExhaustedRetries(t *testing.T) {
	retrier := New(
		alwaysRetriable,
		WithBaseDuration(1*time.Millisecond),
		WithMaxRetries(2),
	)
	ctx := context.Background()

	callCount := 0
	result, err := DoWithResult(
		ctx,
		retrier,
		func(ctx context.Context) (int, error) {
			callCount++
			return 0, errRetriable
		},
	)

	assert.Error(t, err)
	assert.ErrorIs(t, err, errRetriable)
	assert.Contains(t, err.Error(), "operation failed after retries")
	assert.Equal(t, 0, result)
	// 1 первая попытка + 2 maxRetries = 3 вызова
	assert.Equal(t, 3, callCount)
}

func TestDoWithResult_ContextCancellation(t *testing.T) {
	retrier := New(
		alwaysRetriable,
		WithBaseDuration(1*time.Second),
		WithMaxRetries(3),
	)

	ctx, cancel := context.WithCancel(context.Background())

	callCount := 0
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	result, err := DoWithResult(
		ctx,
		retrier,
		func(ctx context.Context) (int, error) {
			callCount++
			return 0, errRetriable
		},
	)

	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 0, result)
	assert.Equal(t, 1, callCount)
}

func TestDoWithResult_PointerResult(t *testing.T) {
	type item struct{ ID int }
	retrier := New(
		alwaysRetriable,
		WithBaseDuration(1*time.Millisecond),
		WithMaxRetries(1),
	)
	ctx := context.Background()

	callCount := 0
	result, err := DoWithResult(
		ctx,
		retrier,
		func(ctx context.Context) (*item, error) {
			callCount++
			if callCount < 2 {
				return nil, errRetriable
			}
			return &item{ID: 7}, nil
		},
	)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 7, result.ID)
	assert.Equal(t, 2, callCount)
}

func BenchmarkDoWithResult_Success(b *testing.B) {
	retrier := New(alwaysRetriable)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DoWithResult(ctx, retrier, func(ctx context.Context) (int, error) {
			return 1, nil
		})
	}
}

func ExampleDoWithResult() {
	isRetriable := func(err error) bool {
		return errors.Is(err, errRetriable)
	}

	retrier := New(
		isRetriable,
		WithBaseDuration(1*time.Millisecond),
		WithMaxRetries(2),
	)

	callCount := 0
	result, err := DoWithResult(
		context.Background(),
		retrier,
		func(ctx context.Context) (string, error) {
			callCount++
			if callCount < 2 {
				return "", errRetriable
			}
			return "ok", nil
		},
	)

	fmt.Printf("Result: %s, Error: %v, Calls: %d\n", result, err, callCount)
	// Output:
	// Result: ok, Error: <nil>, Calls: 2
}
