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
				CONSTRAINT items_user_title_unique UNIQUE (user_id, title)
			);

			CREATE TABLE IF NOT EXISTS refresh_tokens (
				id VARCHAR(36) PRIMARY KEY,
				user_id INTEGER NOT NULL,
				expires_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
				created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
				used_at TIMESTAMP WITH TIME ZONE NULL,
				FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
			);

			CREATE INDEX IF NOT EXISTS idx_user_id ON refresh_tokens(user_id);
			CREATE INDEX IF NOT EXISTS idx_expires_at ON refresh_tokens(expires_at);
			CREATE INDEX IF NOT EXISTS idx_refresh_tokens_used_at ON refresh_tokens(used_at);

			CREATE INDEX IF NOT EXISTS items_user_updated_at_idx ON items(user_id, updated_at DESC);
			CREATE INDEX IF NOT EXISTS items_user_type_idx ON items(user_id, type);

			CREATE TABLE IF NOT EXISTS token_cleanup_log (
				id SERIAL PRIMARY KEY,
				last_cleanup_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
			);

			INSERT INTO token_cleanup_log (last_cleanup_at)
			VALUES (NOW() - INTERVAL '2 hours');

			CREATE OR REPLACE FUNCTION cleanup_expired_refresh_tokens()
			RETURNS TRIGGER AS $$
			DECLARE
				last_cleanup TIMESTAMP WITH TIME ZONE;
			BEGIN
				SELECT last_cleanup_at INTO last_cleanup
				FROM token_cleanup_log
				LIMIT 1;

				IF last_cleanup IS NULL OR last_cleanup < NOW() - INTERVAL '1 hour' THEN
					DELETE FROM refresh_tokens WHERE expires_at < NOW();
					DELETE FROM refresh_tokens WHERE used_at IS NOT NULL AND used_at < NOW() - INTERVAL '1 hour';
					UPDATE token_cleanup_log SET last_cleanup_at = NOW();
				END IF;

				RETURN NULL;
			END;
			$$ LANGUAGE plpgsql;

			CREATE TRIGGER trigger_cleanup_expired_tokens
			AFTER INSERT ON refresh_tokens
			FOR EACH STATEMENT
			EXECUTE FUNCTION cleanup_expired_refresh_tokens();
			`,
			DownSQL: `
			DROP TRIGGER IF EXISTS trigger_cleanup_expired_tokens ON refresh_tokens;
			DROP FUNCTION IF EXISTS cleanup_expired_refresh_tokens();
			DROP TABLE IF EXISTS token_cleanup_log;
			DROP TABLE IF EXISTS refresh_tokens;
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
