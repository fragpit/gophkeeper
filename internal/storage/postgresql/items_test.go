//go:build integration

package postgresql

import (
	"testing"

	"github.com/fragpit/gophkeeper/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestList(t *testing.T) {
	t.Run("success with items", func(t *testing.T) {
		ctx := t.Context()

		user := &model.User{
			Login:        "testuser5",
			PasswordHash: "hash123",
		}
		dek := &model.EncryptedDEK{
			EncryptedKey: []byte("encrypted_dek"),
			Nonce:        []byte("nonce"),
		}
		createdUser, err := repos.Users.CreateWithDEK(ctx, user, dek)
		require.NoError(t, err)

		item1 := &model.ItemEncrypted{
			ItemMeta: &model.ItemMeta{
				Title: "item1",
				Type:  model.ItemTypeLogin,
			},
			Ciphertext: []byte("encrypted1"),
			Nonce:      []byte("nonce1"),
		}
		item2 := &model.ItemEncrypted{
			ItemMeta: &model.ItemMeta{
				Title: "item2",
				Type:  model.ItemTypeNote,
			},
			Ciphertext: []byte("encrypted2"),
			Nonce:      []byte("nonce2"),
		}

		_, err = repos.Items.CreateItem(ctx, createdUser.ID, item1)
		require.NoError(t, err)
		_, err = repos.Items.CreateItem(ctx, createdUser.ID, item2)
		require.NoError(t, err)

		items, err := repos.Items.List(ctx, createdUser.ID, "")

		require.NoError(t, err)
		assert.Len(t, items, 2)
	})

	t.Run("empty list", func(t *testing.T) {
		ctx := t.Context()

		user := &model.User{
			Login:        "testuser6",
			PasswordHash: "hash123",
		}
		dek := &model.EncryptedDEK{
			EncryptedKey: []byte("encrypted_dek"),
			Nonce:        []byte("nonce"),
		}
		createdUser, err := repos.Users.CreateWithDEK(ctx, user, dek)
		require.NoError(t, err)

		items, err := repos.Items.List(ctx, createdUser.ID, "")

		require.NoError(t, err)
		assert.Empty(t, items)
	})

	t.Run("filter by type", func(t *testing.T) {
		ctx := t.Context()

		user := &model.User{
			Login:        "testuser7",
			PasswordHash: "hash123",
		}
		dek := &model.EncryptedDEK{
			EncryptedKey: []byte("encrypted_dek"),
			Nonce:        []byte("nonce"),
		}
		createdUser, err := repos.Users.CreateWithDEK(ctx, user, dek)
		require.NoError(t, err)

		item1 := &model.ItemEncrypted{
			ItemMeta: &model.ItemMeta{
				Title: "cred1",
				Type:  model.ItemTypeLogin,
			},
			Ciphertext: []byte("encrypted1"),
			Nonce:      []byte("nonce1"),
		}
		item2 := &model.ItemEncrypted{
			ItemMeta: &model.ItemMeta{
				Title: "text1",
				Type:  model.ItemTypeNote,
			},
			Ciphertext: []byte("encrypted2"),
			Nonce:      []byte("nonce2"),
		}

		_, err = repos.Items.CreateItem(ctx, createdUser.ID, item1)
		require.NoError(t, err)
		_, err = repos.Items.CreateItem(ctx, createdUser.ID, item2)
		require.NoError(t, err)

		items, err := repos.Items.List(ctx, createdUser.ID, model.ItemTypeLogin)

		require.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Equal(t, "cred1", items[0].Title)
		assert.Equal(t, model.ItemTypeLogin, items[0].Type)
	})
}

func TestCreateItem(t *testing.T) {
	t.Run("success credentials", func(t *testing.T) {
		ctx := t.Context()

		user := &model.User{
			Login:        "testuser8",
			PasswordHash: "hash123",
		}
		dek := &model.EncryptedDEK{
			EncryptedKey: []byte("encrypted_dek"),
			Nonce:        []byte("nonce"),
		}
		createdUser, err := repos.Users.CreateWithDEK(ctx, user, dek)
		require.NoError(t, err)

		item := &model.ItemEncrypted{
			ItemMeta: &model.ItemMeta{
				Title: "my_credentials",
				Type:  model.ItemTypeLogin,
			},
			Ciphertext: []byte("encrypted_data"),
			Nonce:      []byte("nonce_data"),
		}

		itemID, err := repos.Items.CreateItem(ctx, createdUser.ID, item)

		require.NoError(t, err)
		assert.Greater(t, itemID, 0)
	})

	t.Run("success file", func(t *testing.T) {
		ctx := t.Context()

		user := &model.User{
			Login:        "testuser9",
			PasswordHash: "hash123",
		}
		dek := &model.EncryptedDEK{
			EncryptedKey: []byte("encrypted_dek"),
			Nonce:        []byte("nonce"),
		}
		createdUser, err := repos.Users.CreateWithDEK(ctx, user, dek)
		require.NoError(t, err)

		item := &model.ItemEncrypted{
			ItemMeta: &model.ItemMeta{
				Title: "my_file",
				Type:  model.ItemTypeFile,
			},
			Ciphertext:  []byte("encrypted_metadata"),
			Nonce:       []byte("nonce_data"),
			ObjectKey:   "s3://bucket/key",
			ChunkSize:   1024,
			NoncePrefix: []byte("prefix"),
		}

		itemID, err := repos.Items.CreateItem(ctx, createdUser.ID, item)

		require.NoError(t, err)
		assert.Greater(t, itemID, 0)
	})

	t.Run("duplicate item", func(t *testing.T) {
		ctx := t.Context()

		user := &model.User{
			Login:        "testuser10",
			PasswordHash: "hash123",
		}
		dek := &model.EncryptedDEK{
			EncryptedKey: []byte("encrypted_dek"),
			Nonce:        []byte("nonce"),
		}
		createdUser, err := repos.Users.CreateWithDEK(ctx, user, dek)
		require.NoError(t, err)

		item := &model.ItemEncrypted{
			ItemMeta: &model.ItemMeta{
				Title: "duplicate",
				Type:  model.ItemTypeNote,
			},
			Ciphertext: []byte("encrypted"),
			Nonce:      []byte("nonce"),
		}

		_, err = repos.Items.CreateItem(ctx, createdUser.ID, item)
		require.NoError(t, err)

		_, err = repos.Items.CreateItem(ctx, createdUser.ID, item)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrItemExists)
	})

	t.Run("file with empty object key", func(t *testing.T) {
		ctx := t.Context()

		user := &model.User{
			Login:        "emptyobjkeyuser",
			PasswordHash: "hash123",
		}
		dek := &model.EncryptedDEK{
			EncryptedKey: []byte("encrypted_dek"),
			Nonce:        []byte("nonce"),
		}
		createdUser, err := repos.Users.CreateWithDEK(ctx, user, dek)
		require.NoError(t, err)

		item := &model.ItemEncrypted{
			ItemMeta: &model.ItemMeta{
				Title: "file_empty_key",
				Type:  model.ItemTypeFile,
			},
			Ciphertext:  []byte("metadata"),
			Nonce:       []byte("nonce"),
			ObjectKey:   "",
			ChunkSize:   0,
			NoncePrefix: nil,
		}

		itemID, err := repos.Items.CreateItem(ctx, createdUser.ID, item)

		require.NoError(t, err)
		assert.Greater(t, itemID, 0)
	})

	t.Run("file with partial fields", func(t *testing.T) {
		ctx := t.Context()

		user := &model.User{
			Login:        "partialfileuser",
			PasswordHash: "hash123",
		}
		dek := &model.EncryptedDEK{
			EncryptedKey: []byte("encrypted_dek"),
			Nonce:        []byte("nonce"),
		}
		createdUser, err := repos.Users.CreateWithDEK(ctx, user, dek)
		require.NoError(t, err)

		item := &model.ItemEncrypted{
			ItemMeta: &model.ItemMeta{
				Title: "partial_file",
				Type:  model.ItemTypeFile,
			},
			Ciphertext:  []byte("metadata"),
			Nonce:       []byte("nonce"),
			ObjectKey:   "key123",
			ChunkSize:   0,
			NoncePrefix: []byte("pre"),
		}

		itemID, err := repos.Items.CreateItem(ctx, createdUser.ID, item)

		require.NoError(t, err)
		assert.Greater(t, itemID, 0)
	})
}

func TestGetItemByTitle(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := t.Context()

		user := &model.User{
			Login:        "testuser11",
			PasswordHash: "hash123",
		}
		dek := &model.EncryptedDEK{
			EncryptedKey: []byte("encrypted_dek"),
			Nonce:        []byte("nonce"),
		}
		createdUser, err := repos.Users.CreateWithDEK(ctx, user, dek)
		require.NoError(t, err)

		item := &model.ItemEncrypted{
			ItemMeta: &model.ItemMeta{
				Title: "findme",
				Type:  model.ItemTypeLogin,
			},
			Ciphertext: []byte("encrypted_card"),
			Nonce:      []byte("card_nonce"),
		}

		_, err = repos.Items.CreateItem(ctx, createdUser.ID, item)
		require.NoError(t, err)

		retrieved, err := repos.Items.GetItemByTitle(ctx, createdUser.ID, "findme")

		require.NoError(t, err)
		assert.Equal(t, "findme", retrieved.Title)
		assert.Equal(t, model.ItemTypeLogin, retrieved.Type)
		assert.Equal(t, []byte("encrypted_card"), retrieved.Ciphertext)
		assert.Equal(t, []byte("card_nonce"), retrieved.Nonce)
	})

	t.Run("not found", func(t *testing.T) {
		ctx := t.Context()

		user := &model.User{
			Login:        "testuser12",
			PasswordHash: "hash123",
		}
		dek := &model.EncryptedDEK{
			EncryptedKey: []byte("encrypted_dek"),
			Nonce:        []byte("nonce"),
		}
		createdUser, err := repos.Users.CreateWithDEK(ctx, user, dek)
		require.NoError(t, err)

		_, err = repos.Items.GetItemByTitle(ctx, createdUser.ID, "nonexistent")

		require.Error(t, err)
	})

	t.Run("file with metadata", func(t *testing.T) {
		ctx := t.Context()

		user := &model.User{
			Login:        "testuser13",
			PasswordHash: "hash123",
		}
		dek := &model.EncryptedDEK{
			EncryptedKey: []byte("encrypted_dek"),
			Nonce:        []byte("nonce"),
		}
		createdUser, err := repos.Users.CreateWithDEK(ctx, user, dek)
		require.NoError(t, err)

		item := &model.ItemEncrypted{
			ItemMeta: &model.ItemMeta{
				Title: "myfile",
				Type:  model.ItemTypeFile,
			},
			Ciphertext:  []byte("file_metadata"),
			Nonce:       []byte("file_nonce"),
			ObjectKey:   "s3://bucket/object",
			ChunkSize:   2048,
			NoncePrefix: []byte("file_prefix"),
		}

		_, err = repos.Items.CreateItem(ctx, createdUser.ID, item)
		require.NoError(t, err)

		retrieved, err := repos.Items.GetItemByTitle(ctx, createdUser.ID, "myfile")

		require.NoError(t, err)
		assert.Equal(t, "myfile", retrieved.Title)
		assert.Equal(t, model.ItemTypeFile, retrieved.Type)
		assert.Equal(t, "s3://bucket/object", retrieved.ObjectKey)
		assert.Equal(t, 2048, retrieved.ChunkSize)
		assert.Equal(t, []byte("file_prefix"), retrieved.NoncePrefix)
	})
}

func TestList_WithTypeFilter(t *testing.T) {
	t.Run("filter returns only login type", func(t *testing.T) {
		ctx := t.Context()

		user := &model.User{
			Login:        "typefilteruser",
			PasswordHash: "hash123",
		}
		dek := &model.EncryptedDEK{
			EncryptedKey: []byte("encrypted_dek"),
			Nonce:        []byte("nonce"),
		}
		createdUser, err := repos.Users.CreateWithDEK(ctx, user, dek)
		require.NoError(t, err)

		loginItem := &model.ItemEncrypted{
			ItemMeta: &model.ItemMeta{
				Title: "login1",
				Type:  model.ItemTypeLogin,
			},
			Ciphertext: []byte("login_data"),
			Nonce:      []byte("login_nonce"),
		}
		noteItem := &model.ItemEncrypted{
			ItemMeta: &model.ItemMeta{
				Title: "note1",
				Type:  model.ItemTypeNote,
			},
			Ciphertext: []byte("note_data"),
			Nonce:      []byte("note_nonce"),
		}

		_, err = repos.Items.CreateItem(ctx, createdUser.ID, loginItem)
		require.NoError(t, err)
		_, err = repos.Items.CreateItem(ctx, createdUser.ID, noteItem)
		require.NoError(t, err)

		items, err := repos.Items.List(ctx, createdUser.ID, model.ItemTypeLogin)

		require.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Equal(t, "login1", items[0].Title)
		assert.Equal(t, model.ItemTypeLogin, items[0].Type)
	})

	t.Run("filter by note type", func(t *testing.T) {
		ctx := t.Context()

		user := &model.User{
			Login:        "notefilteruser",
			PasswordHash: "hash123",
		}
		dek := &model.EncryptedDEK{
			EncryptedKey: []byte("encrypted_dek"),
			Nonce:        []byte("nonce"),
		}
		createdUser, err := repos.Users.CreateWithDEK(ctx, user, dek)
		require.NoError(t, err)

		noteItem1 := &model.ItemEncrypted{
			ItemMeta: &model.ItemMeta{
				Title: "note_a",
				Type:  model.ItemTypeNote,
			},
			Ciphertext: []byte("note_a_data"),
			Nonce:      []byte("nonce_a"),
		}
		noteItem2 := &model.ItemEncrypted{
			ItemMeta: &model.ItemMeta{
				Title: "note_b",
				Type:  model.ItemTypeNote,
			},
			Ciphertext: []byte("note_b_data"),
			Nonce:      []byte("nonce_b"),
		}

		_, err = repos.Items.CreateItem(ctx, createdUser.ID, noteItem1)
		require.NoError(t, err)
		_, err = repos.Items.CreateItem(ctx, createdUser.ID, noteItem2)
		require.NoError(t, err)

		items, err := repos.Items.List(ctx, createdUser.ID, model.ItemTypeNote)

		require.NoError(t, err)
		assert.Len(t, items, 2)
	})

	t.Run("filter by file type", func(t *testing.T) {
		ctx := t.Context()

		user := &model.User{
			Login:        "filefilteruser",
			PasswordHash: "hash123",
		}
		dek := &model.EncryptedDEK{
			EncryptedKey: []byte("encrypted_dek"),
			Nonce:        []byte("nonce"),
		}
		createdUser, err := repos.Users.CreateWithDEK(ctx, user, dek)
		require.NoError(t, err)

		fileItem := &model.ItemEncrypted{
			ItemMeta: &model.ItemMeta{
				Title: "file_item",
				Type:  model.ItemTypeFile,
			},
			Ciphertext:  []byte("file_meta"),
			Nonce:       []byte("file_nonce"),
			ObjectKey:   "s3://bucket/file",
			ChunkSize:   4096,
			NoncePrefix: []byte("prefix"),
		}
		loginItem := &model.ItemEncrypted{
			ItemMeta: &model.ItemMeta{
				Title: "login_item",
				Type:  model.ItemTypeLogin,
			},
			Ciphertext: []byte("login_data"),
			Nonce:      []byte("login_nonce"),
		}

		_, err = repos.Items.CreateItem(ctx, createdUser.ID, fileItem)
		require.NoError(t, err)
		_, err = repos.Items.CreateItem(ctx, createdUser.ID, loginItem)
		require.NoError(t, err)

		items, err := repos.Items.List(ctx, createdUser.ID, model.ItemTypeFile)

		require.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Equal(t, "file_item", items[0].Title)
		assert.Equal(t, model.ItemTypeFile, items[0].Type)
	})
}

func TestUpdateItem(t *testing.T) {
	t.Run("success update credentials", func(t *testing.T) {
		ctx := t.Context()

		user := &model.User{
			Login:        "updateuser1",
			PasswordHash: "hash123",
		}
		dek := &model.EncryptedDEK{
			EncryptedKey: []byte("encrypted_dek"),
			Nonce:        []byte("nonce"),
		}
		createdUser, err := repos.Users.CreateWithDEK(ctx, user, dek)
		require.NoError(t, err)

		item := &model.ItemEncrypted{
			ItemMeta: &model.ItemMeta{
				Title: "my_login",
				Type:  model.ItemTypeLogin,
			},
			Ciphertext: []byte("old_encrypted_data"),
			Nonce:      []byte("old_nonce"),
		}

		_, err = repos.Items.CreateItem(ctx, createdUser.ID, item)
		require.NoError(t, err)

		updatedItem := &model.ItemEncrypted{
			ItemMeta: &model.ItemMeta{
				Title: "my_login",
				Type:  model.ItemTypeLogin,
			},
			Ciphertext: []byte("new_encrypted_data"),
			Nonce:      []byte("new_nonce"),
		}

		err = repos.Items.UpdateItem(ctx, createdUser.ID, "my_login", updatedItem)

		require.NoError(t, err)

		retrieved, err := repos.Items.GetItemByTitle(
			ctx,
			createdUser.ID,
			"my_login",
		)
		require.NoError(t, err)
		assert.Equal(t, []byte("new_encrypted_data"), retrieved.Ciphertext)
		assert.Equal(t, []byte("new_nonce"), retrieved.Nonce)
	})

	t.Run("success update note", func(t *testing.T) {
		ctx := t.Context()

		user := &model.User{
			Login:        "updateuser2",
			PasswordHash: "hash123",
		}
		dek := &model.EncryptedDEK{
			EncryptedKey: []byte("encrypted_dek"),
			Nonce:        []byte("nonce"),
		}
		createdUser, err := repos.Users.CreateWithDEK(ctx, user, dek)
		require.NoError(t, err)

		item := &model.ItemEncrypted{
			ItemMeta: &model.ItemMeta{
				Title: "my_note",
				Type:  model.ItemTypeNote,
			},
			Ciphertext: []byte("old_note_data"),
			Nonce:      []byte("old_note_nonce"),
		}

		_, err = repos.Items.CreateItem(ctx, createdUser.ID, item)
		require.NoError(t, err)

		updatedItem := &model.ItemEncrypted{
			ItemMeta: &model.ItemMeta{
				Title: "my_note",
				Type:  model.ItemTypeNote,
			},
			Ciphertext: []byte("updated_note_data"),
			Nonce:      []byte("updated_note_nonce"),
		}

		err = repos.Items.UpdateItem(ctx, createdUser.ID, "my_note", updatedItem)

		require.NoError(t, err)

		retrieved, err := repos.Items.GetItemByTitle(ctx, createdUser.ID, "my_note")
		require.NoError(t, err)
		assert.Equal(t, []byte("updated_note_data"), retrieved.Ciphertext)
		assert.Equal(t, []byte("updated_note_nonce"), retrieved.Nonce)
	})

	t.Run("success update file with metadata", func(t *testing.T) {
		ctx := t.Context()

		user := &model.User{
			Login:        "updateuser3",
			PasswordHash: "hash123",
		}
		dek := &model.EncryptedDEK{
			EncryptedKey: []byte("encrypted_dek"),
			Nonce:        []byte("nonce"),
		}
		createdUser, err := repos.Users.CreateWithDEK(ctx, user, dek)
		require.NoError(t, err)

		item := &model.ItemEncrypted{
			ItemMeta: &model.ItemMeta{
				Title: "my_file",
				Type:  model.ItemTypeFile,
			},
			Ciphertext:  []byte("old_file_metadata"),
			Nonce:       []byte("old_file_nonce"),
			ObjectKey:   "s3://bucket/old_key",
			ChunkSize:   1024,
			NoncePrefix: []byte("old_prefix"),
		}

		_, err = repos.Items.CreateItem(ctx, createdUser.ID, item)
		require.NoError(t, err)

		updatedItem := &model.ItemEncrypted{
			ItemMeta: &model.ItemMeta{
				Title: "my_file",
				Type:  model.ItemTypeFile,
			},
			Ciphertext:  []byte("new_file_metadata"),
			Nonce:       []byte("new_file_nonce"),
			ObjectKey:   "s3://bucket/new_key",
			ChunkSize:   2048,
			NoncePrefix: []byte("new_prefix"),
		}

		err = repos.Items.UpdateItem(ctx, createdUser.ID, "my_file", updatedItem)

		require.NoError(t, err)

		retrieved, err := repos.Items.GetItemByTitle(ctx, createdUser.ID, "my_file")
		require.NoError(t, err)
		assert.Equal(t, []byte("new_file_metadata"), retrieved.Ciphertext)
		assert.Equal(t, []byte("new_file_nonce"), retrieved.Nonce)
		assert.Equal(t, "s3://bucket/new_key", retrieved.ObjectKey)
		assert.Equal(t, 2048, retrieved.ChunkSize)
		assert.Equal(t, []byte("new_prefix"), retrieved.NoncePrefix)
	})

	t.Run("item not found", func(t *testing.T) {
		ctx := t.Context()

		user := &model.User{
			Login:        "updateuser4",
			PasswordHash: "hash123",
		}
		dek := &model.EncryptedDEK{
			EncryptedKey: []byte("encrypted_dek"),
			Nonce:        []byte("nonce"),
		}
		createdUser, err := repos.Users.CreateWithDEK(ctx, user, dek)
		require.NoError(t, err)

		updatedItem := &model.ItemEncrypted{
			ItemMeta: &model.ItemMeta{
				Title: "nonexistent",
				Type:  model.ItemTypeNote,
			},
			Ciphertext: []byte("data"),
			Nonce:      []byte("nonce"),
		}

		err = repos.Items.UpdateItem(
			ctx,
			createdUser.ID,
			"nonexistent",
			updatedItem,
		)

		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrItemNotFound)
	})

	t.Run("update deleted item should fail", func(t *testing.T) {
		ctx := t.Context()

		user := &model.User{
			Login:        "updateuser5",
			PasswordHash: "hash123",
		}
		dek := &model.EncryptedDEK{
			EncryptedKey: []byte("encrypted_dek"),
			Nonce:        []byte("nonce"),
		}
		createdUser, err := repos.Users.CreateWithDEK(ctx, user, dek)
		require.NoError(t, err)

		item := &model.ItemEncrypted{
			ItemMeta: &model.ItemMeta{
				Title: "to_delete",
				Type:  model.ItemTypeLogin,
			},
			Ciphertext: []byte("data"),
			Nonce:      []byte("nonce"),
		}

		_, err = repos.Items.CreateItem(ctx, createdUser.ID, item)
		require.NoError(t, err)

		err = repos.Items.DeleteItem(ctx, createdUser.ID, "to_delete")
		require.NoError(t, err)

		updatedItem := &model.ItemEncrypted{
			ItemMeta: &model.ItemMeta{
				Title: "to_delete",
				Type:  model.ItemTypeLogin,
			},
			Ciphertext: []byte("new_data"),
			Nonce:      []byte("new_nonce"),
		}

		err = repos.Items.UpdateItem(ctx, createdUser.ID, "to_delete", updatedItem)

		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrItemNotFound)
	})

	t.Run("update with empty file fields", func(t *testing.T) {
		ctx := t.Context()

		user := &model.User{
			Login:        "updateuser6",
			PasswordHash: "hash123",
		}
		dek := &model.EncryptedDEK{
			EncryptedKey: []byte("encrypted_dek"),
			Nonce:        []byte("nonce"),
		}
		createdUser, err := repos.Users.CreateWithDEK(ctx, user, dek)
		require.NoError(t, err)

		item := &model.ItemEncrypted{
			ItemMeta: &model.ItemMeta{
				Title: "file_item",
				Type:  model.ItemTypeFile,
			},
			Ciphertext:  []byte("metadata"),
			Nonce:       []byte("nonce"),
			ObjectKey:   "s3://bucket/key",
			ChunkSize:   1024,
			NoncePrefix: []byte("prefix"),
		}

		_, err = repos.Items.CreateItem(ctx, createdUser.ID, item)
		require.NoError(t, err)

		updatedItem := &model.ItemEncrypted{
			ItemMeta: &model.ItemMeta{
				Title: "file_item",
				Type:  model.ItemTypeFile,
			},
			Ciphertext:  []byte("new_metadata"),
			Nonce:       []byte("new_nonce"),
			ObjectKey:   "",
			ChunkSize:   0,
			NoncePrefix: nil,
		}

		err = repos.Items.UpdateItem(ctx, createdUser.ID, "file_item", updatedItem)

		require.NoError(t, err)

		retrieved, err := repos.Items.GetItemByTitle(
			ctx,
			createdUser.ID,
			"file_item",
		)
		require.NoError(t, err)
		assert.Equal(t, []byte("new_metadata"), retrieved.Ciphertext)
		assert.Equal(t, []byte("new_nonce"), retrieved.Nonce)
	})

	t.Run("update with different user should not affect", func(t *testing.T) {
		ctx := t.Context()

		user1 := &model.User{
			Login:        "updateuser7a",
			PasswordHash: "hash123",
		}
		user2 := &model.User{
			Login:        "updateuser7b",
			PasswordHash: "hash123",
		}
		dek := &model.EncryptedDEK{
			EncryptedKey: []byte("encrypted_dek"),
			Nonce:        []byte("nonce"),
		}
		createdUser1, err := repos.Users.CreateWithDEK(ctx, user1, dek)
		require.NoError(t, err)
		createdUser2, err := repos.Users.CreateWithDEK(ctx, user2, dek)
		require.NoError(t, err)

		item := &model.ItemEncrypted{
			ItemMeta: &model.ItemMeta{
				Title: "shared_title",
				Type:  model.ItemTypeLogin,
			},
			Ciphertext: []byte("user1_data"),
			Nonce:      []byte("user1_nonce"),
		}

		_, err = repos.Items.CreateItem(ctx, createdUser1.ID, item)
		require.NoError(t, err)

		updatedItem := &model.ItemEncrypted{
			ItemMeta: &model.ItemMeta{
				Title: "shared_title",
				Type:  model.ItemTypeLogin,
			},
			Ciphertext: []byte("user2_attempt"),
			Nonce:      []byte("user2_nonce"),
		}

		err = repos.Items.UpdateItem(
			ctx,
			createdUser2.ID,
			"shared_title",
			updatedItem,
		)

		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrItemNotFound)

		retrieved, err := repos.Items.GetItemByTitle(
			ctx,
			createdUser1.ID,
			"shared_title",
		)
		require.NoError(t, err)
		assert.Equal(t, []byte("user1_data"), retrieved.Ciphertext)
	})

	t.Run("success update title", func(t *testing.T) {
		ctx := t.Context()

		user := &model.User{
			Login:        "updateuser8",
			PasswordHash: "hash123",
		}
		dek := &model.EncryptedDEK{
			EncryptedKey: []byte("encrypted_dek"),
			Nonce:        []byte("nonce"),
		}
		createdUser, err := repos.Users.CreateWithDEK(ctx, user, dek)
		require.NoError(t, err)

		item := &model.ItemEncrypted{
			ItemMeta: &model.ItemMeta{
				Title: "old_title",
				Type:  model.ItemTypeLogin,
			},
			Ciphertext: []byte("encrypted_data"),
			Nonce:      []byte("nonce_data"),
		}

		_, err = repos.Items.CreateItem(ctx, createdUser.ID, item)
		require.NoError(t, err)

		updatedItem := &model.ItemEncrypted{
			ItemMeta: &model.ItemMeta{
				Title: "new_title",
				Type:  model.ItemTypeLogin,
			},
			Ciphertext: []byte("updated_encrypted_data"),
			Nonce:      []byte("updated_nonce"),
		}

		err = repos.Items.UpdateItem(ctx, createdUser.ID, "old_title", updatedItem)

		require.NoError(t, err)

		// Old title should not exist
		_, err = repos.Items.GetItemByTitle(ctx, createdUser.ID, "old_title")
		require.Error(t, err)

		// New title should exist
		retrieved, err := repos.Items.GetItemByTitle(
			ctx,
			createdUser.ID,
			"new_title",
		)
		require.NoError(t, err)
		assert.Equal(t, "new_title", retrieved.Title)
		assert.Equal(t, []byte("updated_encrypted_data"), retrieved.Ciphertext)
		assert.Equal(t, []byte("updated_nonce"), retrieved.Nonce)
	})

	t.Run("update title conflict with existing item", func(t *testing.T) {
		ctx := t.Context()

		user := &model.User{
			Login:        "updateuser9",
			PasswordHash: "hash123",
		}
		dek := &model.EncryptedDEK{
			EncryptedKey: []byte("encrypted_dek"),
			Nonce:        []byte("nonce"),
		}
		createdUser, err := repos.Users.CreateWithDEK(ctx, user, dek)
		require.NoError(t, err)

		item1 := &model.ItemEncrypted{
			ItemMeta: &model.ItemMeta{
				Title: "first_item",
				Type:  model.ItemTypeLogin,
			},
			Ciphertext: []byte("data1"),
			Nonce:      []byte("nonce1"),
		}
		item2 := &model.ItemEncrypted{
			ItemMeta: &model.ItemMeta{
				Title: "second_item",
				Type:  model.ItemTypeNote,
			},
			Ciphertext: []byte("data2"),
			Nonce:      []byte("nonce2"),
		}

		_, err = repos.Items.CreateItem(ctx, createdUser.ID, item1)
		require.NoError(t, err)
		_, err = repos.Items.CreateItem(ctx, createdUser.ID, item2)
		require.NoError(t, err)

		// Try to update second_item to have the same title as first_item
		updatedItem := &model.ItemEncrypted{
			ItemMeta: &model.ItemMeta{
				Title: "first_item",
				Type:  model.ItemTypeNote,
			},
			Ciphertext: []byte("updated_data2"),
			Nonce:      []byte("updated_nonce2"),
		}

		err = repos.Items.UpdateItem(
			ctx,
			createdUser.ID,
			"second_item",
			updatedItem,
		)

		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrAlreadyExists)

		// Verify original items are unchanged
		retrieved1, err := repos.Items.GetItemByTitle(
			ctx,
			createdUser.ID,
			"first_item",
		)
		require.NoError(t, err)
		assert.Equal(t, []byte("data1"), retrieved1.Ciphertext)

		retrieved2, err := repos.Items.GetItemByTitle(
			ctx,
			createdUser.ID,
			"second_item",
		)
		require.NoError(t, err)
		assert.Equal(t, []byte("data2"), retrieved2.Ciphertext)
	})
}
