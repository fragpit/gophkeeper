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

// ErrItemExists is returned when a duplicate item is inserted.
var ErrItemExists = errors.New("item already exists")

var _ model.ItemsRepository = (*itemsRepo)(nil)

type itemsRepo struct {
	db      *pgxpool.Pool
	retrier *retry.Retrier
}

// NewItemsRepo creates an items repository backed by PostgreSQL.
func NewItemsRepo(db *pgxpool.Pool, r *retry.Retrier) *itemsRepo {
	return &itemsRepo{
		db:      db,
		retrier: r,
	}
}

func (r *itemsRepo) List(
	ctx context.Context,
	userID int,
	iType model.ItemType,
) ([]model.ItemMeta, error) {
	q := `
		SELECT title, type
		FROM items
		WHERE user_id = $1
	`

	args := []any{userID}
	if iType != "" {
		q += ` AND type = $2`
		args = append(args, string(iType))
	}
	q += ` ORDER BY updated_at DESC;`

	return retry.DoWithResult(
		ctx,
		r.retrier,
		func(ctx context.Context) ([]model.ItemMeta, error) {
			rows, err := r.db.Query(ctx, q, args...)
			if err != nil {
				return nil, fmt.Errorf("list items: %w", err)
			}
			defer rows.Close()

			items := make([]model.ItemMeta, 0)
			for rows.Next() {
				var title string
				var typ string

				if err := rows.Scan(&title, &typ); err != nil {
					return nil, fmt.Errorf("list items scan: %w", err)
				}
				items = append(items, model.ItemMeta{
					Title: title,
					Type:  model.ItemType(typ),
				})
			}

			if err := rows.Err(); err != nil {
				return nil, fmt.Errorf("list items rows: %w", err)
			}

			return items, nil
		},
	)
}

func (r *itemsRepo) CreateItem(
	ctx context.Context,
	userID int,
	item *model.ItemEncrypted,
) (int, error) {
	q := `
		INSERT INTO items (
			user_id, type, title,
			data_ciphertext, data_nonce,
			object_key, chunk_size, nonce_prefix
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id;
	`

	var objectKey *string
	var chunkSize *int32
	var noncePrefix []byte

	if item.Type == model.ItemTypeFile {
		if item.ObjectKey != "" {
			v := item.ObjectKey
			objectKey = &v
		}
		if item.ChunkSize != 0 {
			v := int32(item.ChunkSize)
			chunkSize = &v
		}
		if len(item.NoncePrefix) != 0 {
			noncePrefix = item.NoncePrefix
		}
	}

	op := func(ctx context.Context) (int, error) {
		row := r.db.QueryRow(
			ctx,
			q,
			userID,
			string(item.Type),
			item.Title,
			item.Ciphertext,
			item.Nonce,
			objectKey,
			chunkSize,
			noncePrefix,
		)
		var id int
		if err := row.Scan(&id); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
				return 0, errors.Join(ErrItemExists, err)
			}
			return 0, fmt.Errorf("create item: %w", err)
		}
		return id, nil
	}

	id, err := retry.DoWithResult(ctx, r.retrier, op)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (r *itemsRepo) GetItemByTitle(
	ctx context.Context,
	userID int,
	iTitle string,
) (*model.ItemEncrypted, error) {
	q := `
		SELECT
			type, title,
			data_ciphertext, data_nonce,
			object_key, chunk_size, nonce_prefix
		FROM items
		WHERE user_id = $1 AND title = $2;
	`

	return retry.DoWithResult(
		ctx,
		r.retrier,
		func(ctx context.Context) (*model.ItemEncrypted, error) {
			var (
				typ        string
				title      string
				ciphertext []byte
				nonce      []byte

				objectKey   *string
				chunkSize   *int32
				noncePrefix []byte
			)

			row := r.db.QueryRow(ctx, q, userID, iTitle)
			if err := row.Scan(
				&typ,
				&title,
				&ciphertext,
				&nonce,
				&objectKey,
				&chunkSize,
				&noncePrefix,
			); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return nil, model.ErrItemNotFound
				}
				return nil, fmt.Errorf("get item by title: %w", err)
			}

			item := &model.ItemEncrypted{
				ItemMeta: &model.ItemMeta{
					Title: title,
					Type:  model.ItemType(typ),
				},
				Ciphertext: ciphertext,
				Nonce:      nonce,
			}

			if objectKey != nil {
				item.ObjectKey = *objectKey
			}
			if chunkSize != nil {
				item.ChunkSize = int(*chunkSize)
			}
			if len(noncePrefix) != 0 {
				item.NoncePrefix = noncePrefix
			}

			return item, nil
		},
	)
}

func (r *itemsRepo) UpdateItem(
	ctx context.Context,
	userID int,
	title string,
	item *model.ItemEncrypted,
) error {
	q := `
		UPDATE items
		SET
			title = $3,
			data_ciphertext = $4,
			data_nonce = $5,
			object_key = $6,
			chunk_size = $7,
			nonce_prefix = $8,
			updated_at = NOW(),
			revision = revision + 1
		WHERE user_id = $1 AND title = $2;
	`

	var objectKey *string
	var chunkSize *int32
	var noncePrefix []byte

	if item.Type == model.ItemTypeFile {
		if item.ObjectKey != "" {
			v := item.ObjectKey
			objectKey = &v
		}
		if item.ChunkSize != 0 {
			v := int32(item.ChunkSize)
			chunkSize = &v
		}
		if len(item.NoncePrefix) != 0 {
			noncePrefix = item.NoncePrefix
		}
	}

	op := func(ctx context.Context) error {
		cmdTag, err := r.db.Exec(
			ctx,
			q,
			userID,
			title,
			item.Title,
			item.Ciphertext,
			item.Nonce,
			objectKey,
			chunkSize,
			noncePrefix,
		)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
				return errors.Join(model.ErrAlreadyExists, err)
			}
			return fmt.Errorf("update item: %w", err)
		}
		if cmdTag.RowsAffected() == 0 {
			return model.ErrItemNotFound
		}
		return nil
	}

	return r.retrier.Do(ctx, op)
}

func (r *itemsRepo) DeleteItem(
	ctx context.Context,
	userID int,
	title string,
) error {
	q := `
		DELETE FROM items
		WHERE user_id = $1 AND title = $2;
	`

	op := func(ctx context.Context) error {
		cmdTag, err := r.db.Exec(ctx, q, userID, title)
		if err != nil {
			return fmt.Errorf("delete item: %w", err)
		}
		if cmdTag.RowsAffected() == 0 {
			return model.ErrItemNotFound
		}
		return nil
	}

	return r.retrier.Do(ctx, op)
}
