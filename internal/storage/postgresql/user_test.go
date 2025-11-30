//go:build integration

package postgresql

import (
	"testing"

	"github.com/fragpit/gophkeeper/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateWithDEK(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := t.Context()

		user := &model.User{
			Login:        "testuser1",
			PasswordHash: "hash123",
		}
		dek := &model.EncryptedDEK{
			EncryptedKey: []byte("encrypted_dek"),
			Nonce:        []byte("nonce"),
		}

		createdUser, err := repos.Users.CreateWithDEK(ctx, user, dek)

		require.NoError(t, err)
		assert.Greater(t, createdUser.ID, 0)
	})

	t.Run("duplicate user", func(t *testing.T) {
		ctx := t.Context()

		user := &model.User{
			Login:        "testuser2",
			PasswordHash: "hash123",
		}
		dek := &model.EncryptedDEK{
			EncryptedKey: []byte("encrypted_dek"),
			Nonce:        []byte("nonce"),
		}

		_, err := repos.Users.CreateWithDEK(ctx, user, dek)
		require.NoError(t, err)

		_, err = repos.Users.CreateWithDEK(ctx, user, dek)
		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrUserExists)
	})
}

func TestGetByLogin(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := t.Context()

		user := &model.User{
			Login:        "testuser3",
			PasswordHash: "hash456",
		}
		dek := &model.EncryptedDEK{
			EncryptedKey: []byte("encrypted_dek"),
			Nonce:        []byte("nonce"),
		}

		createdUser, err := repos.Users.CreateWithDEK(ctx, user, dek)
		require.NoError(t, err)

		retrieved, err := repos.Users.GetByLogin(ctx, "testuser3")

		require.NoError(t, err)
		assert.Equal(t, createdUser.ID, retrieved.ID)
		assert.Equal(t, "testuser3", retrieved.Login)
		assert.Equal(t, "hash456", retrieved.PasswordHash)
	})

	t.Run("not found", func(t *testing.T) {
		ctx := t.Context()

		_, err := repos.Users.GetByLogin(ctx, "nonexistent")

		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrUserNotFound)
	})
}

func TestGetUserDEK(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := t.Context()

		user := &model.User{
			Login:        "testuser4",
			PasswordHash: "hash789",
		}
		expectedDEK := &model.EncryptedDEK{
			EncryptedKey: []byte("encrypted_dek_data"),
			Nonce:        []byte("test_nonce"),
		}

		createdUser, err := repos.Users.CreateWithDEK(ctx, user, expectedDEK)
		require.NoError(t, err)

		retrievedDEK, err := repos.Users.GetUserDEK(ctx, createdUser.ID)

		require.NoError(t, err)
		assert.Equal(t, expectedDEK.EncryptedKey, retrievedDEK.EncryptedKey)
		assert.Equal(t, expectedDEK.Nonce, retrievedDEK.Nonce)
	})

	t.Run("not found", func(t *testing.T) {
		ctx := t.Context()

		_, err := repos.Users.GetUserDEK(ctx, 999999)

		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrUserNotFound)
	})

	t.Run("invalid user id", func(t *testing.T) {
		ctx := t.Context()

		_, err := repos.Users.GetUserDEK(ctx, 0)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user id")
	})

	t.Run("negative user id", func(t *testing.T) {
		ctx := t.Context()

		_, err := repos.Users.GetUserDEK(ctx, -1)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user id")
	})
}

func TestCreateWithDEK_NilInputs(t *testing.T) {
	t.Run("nil user", func(t *testing.T) {
		ctx := t.Context()

		dek := &model.EncryptedDEK{
			EncryptedKey: []byte("encrypted_dek"),
			Nonce:        []byte("nonce"),
		}

		_, err := repos.Users.CreateWithDEK(ctx, nil, dek)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "user is nil")
	})

	t.Run("nil dek", func(t *testing.T) {
		ctx := t.Context()

		user := &model.User{
			Login:        "testuser_nil_dek",
			PasswordHash: "hash123",
		}

		_, err := repos.Users.CreateWithDEK(ctx, user, nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "dek is nil")
	})
}
