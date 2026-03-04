package postgresql

import (
	"context"
	"errors"
	"fmt"

	"github.com/fragpit/gophkeeper/internal/service/auth"
	"github.com/fragpit/gophkeeper/pkg/utils/retry"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrTokenExists is returned when a duplicate token is inserted.
var ErrTokenExists = errors.New("token already exists")

var _ auth.TokenRepository = (*tokenRepo)(nil)

type tokenRepo struct {
	db      *pgxpool.Pool
	retrier *retry.Retrier
}

// NewTokenRepo creates a token repository backed by PostgreSQL.
func NewTokenRepo(db *pgxpool.Pool, r *retry.Retrier) *tokenRepo {
	return &tokenRepo{
		db:      db,
		retrier: r,
	}
}

func (r *tokenRepo) StoreRefreshToken(
	ctx context.Context,
	data *auth.RefreshTokenData,
) error {
	q := `
		INSERT INTO refresh_tokens (id, user_id, created_at, expires_at)
		VALUES ($1, $2, $3, $4)
	`

	op := func(ctx context.Context) error {
		_, err := r.db.Exec(
			ctx,
			q,
			data.TokenID,
			data.UserID,
			data.CreatedAt,
			data.ExpiresAt,
		)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
				return ErrTokenExists
			}
			return fmt.Errorf("store refresh token: %w", err)
		}
		return nil
	}

	return r.retrier.Do(ctx, op)
}

// UseRefreshToken atomically marks the token as used (optimistic locking).
// Returns userID if successful, or error if token doesn't exist, expired, or already used.
func (r *tokenRepo) UseRefreshToken(
	ctx context.Context,
	tokenID string,
) (int, error) {
	q := `
		UPDATE refresh_tokens
		SET used_at = NOW()
		WHERE id = $1
		  AND expires_at > NOW()
		  AND used_at IS NULL
		RETURNING user_id
	`

	var userID int
	err := r.db.QueryRow(ctx, q, tokenID).Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf("use refresh token: %w", err)
	}

	return userID, nil
}

func (r *tokenRepo) RefreshTokenExists(
	ctx context.Context,
	tokenID string,
) (bool, error) {
	q := `SELECT EXISTS(
		SELECT 1 FROM refresh_tokens WHERE id = $1 AND expires_at > NOW()
	)`

	var exists bool
	err := r.db.QueryRow(ctx, q, tokenID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check refresh token exists: %w", err)
	}

	return exists, nil
}

func (r *tokenRepo) DeleteRefreshToken(
	ctx context.Context,
	tokenID string,
) error {
	q := `DELETE FROM refresh_tokens WHERE id = $1`

	_, err := r.db.Exec(ctx, q, tokenID)
	if err != nil {
		return fmt.Errorf("delete refresh token: %w", err)
	}
	return nil
}
