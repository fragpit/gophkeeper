package postgresql

import (
	"context"
	"errors"
	"fmt"

	"github.com/fragpit/gophkeeper/internal/model"
	"github.com/fragpit/gophkeeper/pkg/utils/retry"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ model.UsersRepository = (*usersRepo)(nil)

type usersRepo struct {
	db      *pgxpool.Pool
	retrier *retry.Retrier
}

func NewUsersRepo(db *pgxpool.Pool, r *retry.Retrier) *usersRepo {
	return &usersRepo{
		db:      db,
		retrier: r,
	}
}

func (r *usersRepo) CreateWithDEK(
	ctx context.Context,
	user *model.User,
	dek *model.EncryptedDEK,
) (*model.User, error) {
	if user == nil {
		return nil, fmt.Errorf("user is nil")
	}
	if dek == nil {
		return nil, fmt.Errorf("dek is nil")
	}

	qCreateUser := `
		INSERT INTO users (login, password_hash)
		VALUES ($1, $2)
		RETURNING id;
	`
	qCreateDEK := `
		INSERT INTO user_encryption_keys (user_id, encrypted_key, key_nonce)
		VALUES ($1, $2, $3);
	`

	op := func(ctx context.Context) error {
		tx, err := r.db.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback(ctx)

		var id int
		row := tx.QueryRow(ctx, qCreateUser, user.Login, user.PasswordHash)
		if err := row.Scan(&id); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
				return model.ErrUserExists
			}
			return fmt.Errorf("create user: %w", err)
		}

		if _, err := tx.Exec(ctx, qCreateDEK, id, dek.EncryptedKey, dek.Nonce); err != nil {
			return fmt.Errorf("create user dek: %w", err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit tx: %w", err)
		}

		user.ID = id
		return nil
	}

	if r.retrier == nil {
		if err := op(ctx); err != nil {
			return nil, err
		}
		return user, nil
	}
	if err := r.retrier.Do(ctx, op); err != nil {
		return nil, err
	}

	return user, nil
}

func (r *usersRepo) GetByLogin(
	ctx context.Context,
	login string,
) (*model.User, error) {
	q := `
		SELECT id, login, password_hash
		FROM users
		WHERE login = $1
	`

	var (
		userID    int
		userLogin string
		userPHash string
	)
	row := r.db.QueryRow(ctx, q, login)
	if err := row.Scan(&userID, &userLogin, &userPHash); err != nil {
		return nil, fmt.Errorf("failed to get user by login: %w", err)
	}

	u := &model.User{
		ID:           userID,
		Login:        userLogin,
		PasswordHash: userPHash,
	}

	return u, nil
}

func (r *usersRepo) GetUserDEK(
	ctx context.Context,
	userID int,
) (*model.EncryptedDEK, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("invalid user id")
	}

	q := `
		SELECT encrypted_key, key_nonce
		FROM user_encryption_keys
		WHERE user_id = $1
		LIMIT 1;
	`

	var dek *model.EncryptedDEK
	op := func(ctx context.Context) error {
		var (
			encryptedKey []byte
			nonce        []byte
		)

		row := r.db.QueryRow(ctx, q, userID)
		if err := row.Scan(&encryptedKey, &nonce); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return model.ErrUserNotFound
			}
			return fmt.Errorf("get user dek: %w", err)
		}

		dek = &model.EncryptedDEK{
			EncryptedKey: encryptedKey,
			Nonce:        nonce,
		}
		return nil
	}

	if r.retrier == nil {
		if err := op(ctx); err != nil {
			return nil, err
		}
		return dek, nil
	}
	if err := r.retrier.Do(ctx, op); err != nil {
		return nil, err
	}

	return dek, nil
}
