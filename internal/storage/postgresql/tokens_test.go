//go:build integration

package postgresql

import (
	"testing"
	"time"

	"github.com/fragpit/gophkeeper/internal/model"
	"github.com/fragpit/gophkeeper/internal/service/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreRefreshToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := t.Context()

		user := &model.User{
			Login:        "tokenuser1",
			PasswordHash: "hash123",
		}
		dek := &model.EncryptedDEK{
			EncryptedKey: []byte("encrypted_dek"),
			Nonce:        []byte("nonce"),
		}
		createdUser, err := repos.Users.CreateWithDEK(ctx, user, dek)
		require.NoError(t, err)

		tokenData := &auth.RefreshTokenData{
			TokenID:   "token_id_1",
			UserID:    createdUser.ID,
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}

		err = repos.Tokens.StoreRefreshToken(ctx, tokenData)

		require.NoError(t, err)
	})

	t.Run("duplicate token", func(t *testing.T) {
		ctx := t.Context()

		user := &model.User{
			Login:        "tokenuser2",
			PasswordHash: "hash123",
		}
		dek := &model.EncryptedDEK{
			EncryptedKey: []byte("encrypted_dek"),
			Nonce:        []byte("nonce"),
		}
		createdUser, err := repos.Users.CreateWithDEK(ctx, user, dek)
		require.NoError(t, err)

		tokenData := &auth.RefreshTokenData{
			TokenID:   "token_id_2",
			UserID:    createdUser.ID,
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}

		err = repos.Tokens.StoreRefreshToken(ctx, tokenData)
		require.NoError(t, err)

		err = repos.Tokens.StoreRefreshToken(ctx, tokenData)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrTokenExists)
	})
}

func TestRefreshTokenExists(t *testing.T) {
	t.Run("token exists and not expired", func(t *testing.T) {
		ctx := t.Context()

		user := &model.User{
			Login:        "tokenuser3",
			PasswordHash: "hash123",
		}
		dek := &model.EncryptedDEK{
			EncryptedKey: []byte("encrypted_dek"),
			Nonce:        []byte("nonce"),
		}
		createdUser, err := repos.Users.CreateWithDEK(ctx, user, dek)
		require.NoError(t, err)

		tokenData := &auth.RefreshTokenData{
			TokenID:   "token_id_3",
			UserID:    createdUser.ID,
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}

		err = repos.Tokens.StoreRefreshToken(ctx, tokenData)
		require.NoError(t, err)

		exists, err := repos.Tokens.RefreshTokenExists(ctx, "token_id_3")

		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("token does not exist", func(t *testing.T) {
		ctx := t.Context()

		exists, err := repos.Tokens.RefreshTokenExists(ctx, "nonexistent_token")

		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("token expired", func(t *testing.T) {
		ctx := t.Context()

		user := &model.User{
			Login:        "tokenuser4",
			PasswordHash: "hash123",
		}
		dek := &model.EncryptedDEK{
			EncryptedKey: []byte("encrypted_dek"),
			Nonce:        []byte("nonce"),
		}
		createdUser, err := repos.Users.CreateWithDEK(ctx, user, dek)
		require.NoError(t, err)

		tokenData := &auth.RefreshTokenData{
			TokenID:   "token_id_4",
			UserID:    createdUser.ID,
			CreatedAt: time.Now().Add(-48 * time.Hour),
			ExpiresAt: time.Now().Add(-24 * time.Hour),
		}

		err = repos.Tokens.StoreRefreshToken(ctx, tokenData)
		require.NoError(t, err)

		exists, err := repos.Tokens.RefreshTokenExists(ctx, "token_id_4")

		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestDeleteRefreshToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := t.Context()

		user := &model.User{
			Login:        "tokenuser5",
			PasswordHash: "hash123",
		}
		dek := &model.EncryptedDEK{
			EncryptedKey: []byte("encrypted_dek"),
			Nonce:        []byte("nonce"),
		}
		createdUser, err := repos.Users.CreateWithDEK(ctx, user, dek)
		require.NoError(t, err)

		tokenData := &auth.RefreshTokenData{
			TokenID:   "token_id_5",
			UserID:    createdUser.ID,
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}

		err = repos.Tokens.StoreRefreshToken(ctx, tokenData)
		require.NoError(t, err)

		err = repos.Tokens.DeleteRefreshToken(ctx, "token_id_5")
		require.NoError(t, err)

		exists, err := repos.Tokens.RefreshTokenExists(ctx, "token_id_5")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("delete nonexistent token", func(t *testing.T) {
		ctx := t.Context()

		err := repos.Tokens.DeleteRefreshToken(ctx, "nonexistent_token_delete")

		assert.NoError(t, err)
	})
}

func TestUseRefreshToken(t *testing.T) {
	t.Run("success - mark token as used", func(t *testing.T) {
		ctx := t.Context()

		user := &model.User{
			Login:        "tokenuser6",
			PasswordHash: "hash123",
		}
		dek := &model.EncryptedDEK{
			EncryptedKey: []byte("encrypted_dek"),
			Nonce:        []byte("nonce"),
		}
		createdUser, err := repos.Users.CreateWithDEK(ctx, user, dek)
		require.NoError(t, err)

		tokenData := &auth.RefreshTokenData{
			TokenID:   "token_id_6",
			UserID:    createdUser.ID,
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}

		err = repos.Tokens.StoreRefreshToken(ctx, tokenData)
		require.NoError(t, err)

		userID, err := repos.Tokens.UseRefreshToken(ctx, "token_id_6")

		require.NoError(t, err)
		assert.Equal(t, createdUser.ID, userID)
	})

	t.Run("race condition - token already used", func(t *testing.T) {
		ctx := t.Context()

		user := &model.User{
			Login:        "tokenuser7",
			PasswordHash: "hash123",
		}
		dek := &model.EncryptedDEK{
			EncryptedKey: []byte("encrypted_dek"),
			Nonce:        []byte("nonce"),
		}
		createdUser, err := repos.Users.CreateWithDEK(ctx, user, dek)
		require.NoError(t, err)

		tokenData := &auth.RefreshTokenData{
			TokenID:   "token_id_7",
			UserID:    createdUser.ID,
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}

		err = repos.Tokens.StoreRefreshToken(ctx, tokenData)
		require.NoError(t, err)

		// First use - should succeed
		userID, err := repos.Tokens.UseRefreshToken(ctx, "token_id_7")
		require.NoError(t, err)
		assert.Equal(t, createdUser.ID, userID)

		// Second use - should fail (simulating race condition)
		_, err = repos.Tokens.UseRefreshToken(ctx, "token_id_7")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "use refresh token")
	})

	t.Run("token does not exist", func(t *testing.T) {
		ctx := t.Context()

		_, err := repos.Tokens.UseRefreshToken(ctx, "nonexistent_token_use")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "use refresh token")
	})

	t.Run("token expired", func(t *testing.T) {
		ctx := t.Context()

		user := &model.User{
			Login:        "tokenuser8",
			PasswordHash: "hash123",
		}
		dek := &model.EncryptedDEK{
			EncryptedKey: []byte("encrypted_dek"),
			Nonce:        []byte("nonce"),
		}
		createdUser, err := repos.Users.CreateWithDEK(ctx, user, dek)
		require.NoError(t, err)

		tokenData := &auth.RefreshTokenData{
			TokenID:   "token_id_8",
			UserID:    createdUser.ID,
			CreatedAt: time.Now().Add(-48 * time.Hour),
			ExpiresAt: time.Now().Add(-24 * time.Hour),
		}

		err = repos.Tokens.StoreRefreshToken(ctx, tokenData)
		require.NoError(t, err)

		_, err = repos.Tokens.UseRefreshToken(ctx, "token_id_8")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "use refresh token")
	})
}
