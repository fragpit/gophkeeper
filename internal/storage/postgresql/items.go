package postgresql

import (
	"context"
	"errors"
	"fmt"

	"github.com/fragpit/gophkeeper/internal/model"
	"github.com/fragpit/gophkeeper/pkg/utils/retry"
	"github.com/jackc/pgerrcode"
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
		WHERE user_id = $1 AND deleted_at IS NULL
	`

	args := []any{userID}
	if iType != "" {
		q += ` AND type = $2`
		args = append(args, string(iType))
	}
	q += ` ORDER BY updated_at DESC;`

	items := make([]model.ItemMeta, 0)
	op := func(ctx context.Context) error {
		rows, err := r.db.Query(ctx, q, args...)
		if err != nil {
			return fmt.Errorf("list items: %w", err)
		}
		defer rows.Close()

		items = items[:0]
		for rows.Next() {
			var title string
			var typ string

			if err := rows.Scan(&title, &typ); err != nil {
				return fmt.Errorf("list items scan: %w", err)
			}
			items = append(items, model.ItemMeta{
				Title: title,
				Type:  model.ItemType(typ),
			})
		}

		if err := rows.Err(); err != nil {
			return fmt.Errorf("list items rows: %w", err)
		}

		return nil
	}

	if r.retrier == nil {
		if err := op(ctx); err != nil {
			return nil, err
		}
		return items, nil
	}
	if err := r.retrier.Do(ctx, op); err != nil {
		return nil, err
	}

	return items, nil
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

	var id int
	op := func(ctx context.Context) error {
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
		if err := row.Scan(&id); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
				return errors.Join(ErrItemExists, err)
			}
			return fmt.Errorf("create item: %w", err)
		}
		return nil
	}

	if r.retrier == nil {
		if err := op(ctx); err != nil {
			return 0, err
		}
		return id, nil
	}
	if err := r.retrier.Do(ctx, op); err != nil {
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
		WHERE user_id = $1 AND title = $2 AND deleted_at IS NULL;
	`

	var item *model.ItemEncrypted
	op := func(ctx context.Context) error {
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
			return fmt.Errorf("get item by title: %w", err)
		}

		item = &model.ItemEncrypted{
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

		return nil
	}

	if r.retrier == nil {
		if err := op(ctx); err != nil {
			return nil, err
		}
		return item, nil
	}
	if err := r.retrier.Do(ctx, op); err != nil {
		return nil, err
	}

	return item, nil
}
