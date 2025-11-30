package postgresql

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/tern/v2/migrate"
)

func runMigrations(ctx context.Context, conn *pgxpool.Pool) error {
	poolConn, err := conn.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("create pool connection: %w", err)
	}
	defer poolConn.Release()

	m, err := migrate.NewMigrator(ctx, poolConn.Conn(), "metrics_migrations")
	if err != nil {
		return fmt.Errorf("init migrations: %w", err)
	}

	m.Migrations = []*migrate.Migration{
		{
			Sequence: 1,
			Name:     "init",
			UpSQL: `
			CREATE TABLE IF NOT EXISTS users (
				id SERIAL PRIMARY KEY,
				login VARCHAR(255) UNIQUE NOT NULL,
				password_hash VARCHAR(255) NOT NULL,
				created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
			);

			CREATE TABLE IF NOT EXISTS user_encryption_keys (
				user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
				encrypted_key BYTEA NOT NULL,
				key_nonce BYTEA NOT NULL,
				created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
			);

			CREATE TABLE IF NOT EXISTS items (
				id SERIAL PRIMARY KEY,
				user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				type TEXT NOT NULL,
				title TEXT NOT NULL,
				meta JSONB NOT NULL DEFAULT '{}'::jsonb,
				data_ciphertext BYTEA NOT NULL,
				data_nonce BYTEA NOT NULL,
				object_key TEXT NULL,
				chunk_size INTEGER NULL,
				nonce_prefix BYTEA NULL,
				revision BIGINT NOT NULL DEFAULT 1,
				created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
				updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
				deleted_at TIMESTAMP WITH TIME ZONE NULL,
				CONSTRAINT items_user_title_unique UNIQUE (user_id, title)
			);

			CREATE INDEX IF NOT EXISTS items_user_updated_at_idx ON items(user_id, updated_at DESC);
			CREATE INDEX IF NOT EXISTS items_user_type_idx ON items(user_id, type);
			CREATE INDEX IF NOT EXISTS items_user_deleted_at_idx ON items(user_id, deleted_at);

			`,
			DownSQL: `
			DROP TABLE IF EXISTS items;
			DROP TABLE IF EXISTS user_encryption_keys;
			DROP TABLE IF EXISTS users;
			`,
		},
	}

	if err := m.Migrate(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}
