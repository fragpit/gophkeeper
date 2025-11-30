package postgresql

import (
	"context"
	"errors"
	"fmt"

	"github.com/fragpit/gophkeeper/pkg/utils/retry"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repositories aggregates available PostgreSQL-backed repositories.
type Repositories struct {
	Healthcheck *healthCheckRepo
	Users       *usersRepo
	Items       *itemsRepo
}

// NewStorage initializes database connections and repositories.
func NewStorage(ctx context.Context, dbDSN string) (*Repositories, error) {
	dbPool, err := pgxpool.New(ctx, dbDSN)
	if err != nil {
		return nil, fmt.Errorf("create pgxpool: %w", err)
	}

	if err := dbPool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	_, err = dbPool.Exec(ctx, "SET timezone = 'UTC'")
	if err != nil {
		return nil, fmt.Errorf("set timezone to UTC: %w", err)
	}

	if err := runMigrations(ctx, dbPool); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	isRetriable := func(err error) bool {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return pgerrcode.IsConnectionException(pgErr.Code) ||
				pgerrcode.IsOperatorIntervention(pgErr.Code)
		}

		var connErr *pgconn.ConnectError
		return errors.As(err, &connErr)
	}

	retrier := retry.New(isRetriable)
	return &Repositories{
		Healthcheck: NewHealthCheckRepo(dbPool, retrier),
		Users:       NewUsersRepo(dbPool, retrier),
		Items:       NewItemsRepo(dbPool, retrier),
	}, nil
}
