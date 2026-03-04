package postgresql

import (
	"context"
	"errors"

	"github.com/fragpit/gophkeeper/internal/service/healthcheck"
	"github.com/fragpit/gophkeeper/pkg/utils/retry"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	// ErrConnectionNotInitialized indicates database connection is missing.
	ErrConnectionNotInitialized = errors.New("connection not initialized")
	// ErrHealthCheckFailed signals that the health check failed.
	ErrHealthCheckFailed = errors.New("health check failed")
)

var _ healthcheck.HealthRepository = (*healthCheckRepo)(nil)

type healthCheckRepo struct {
	db      *pgxpool.Pool
	retrier *retry.Retrier
}

// NewHealthCheckRepo returns a health check repository backed by PostgreSQL.
func NewHealthCheckRepo(db *pgxpool.Pool, r *retry.Retrier) *healthCheckRepo {
	return &healthCheckRepo{
		db:      db,
		retrier: r,
	}
}

func (h *healthCheckRepo) Ping(ctx context.Context) error {
	if h.db == nil {
		return ErrConnectionNotInitialized
	}

	op := func(ctx context.Context) error {
		return h.db.Ping(ctx)
	}

	if err := h.retrier.Do(ctx, op); err != nil {
		return errors.Join(ErrHealthCheckFailed, err)
	}

	return nil
}
