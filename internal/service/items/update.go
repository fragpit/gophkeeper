package items

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/fragpit/gophkeeper/internal/api/handlers"
	"github.com/fragpit/gophkeeper/internal/model"
)

var _ handlers.UpdateService = (*UpdateService)(nil)

// UpdateService handles update of encrypted items.
type UpdateService struct {
	repo      model.ItemsRepository
	userRepo  model.UsersRepository
	encryptor Encryptor
}

// NewUpdateItemService constructs an UpdateService instance.
func NewUpdateItemService(
	repo model.ItemsRepository,
	userRepo model.UsersRepository,
	encryptor Encryptor,
) *UpdateService {
	return &UpdateService{
		repo:      repo,
		userRepo:  userRepo,
		encryptor: encryptor,
	}
}

// UpdateItem encrypts and updates an existing item for the specified user.
func (s *UpdateService) UpdateItem(
	ctx context.Context,
	userID int,
	title string,
	item *model.ItemDecrypted,
) error {
	edek, err := s.userRepo.GetUserDEK(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user dek: %w", err)
	}

	dek, err := s.encryptor.DecryptDEK(edek)
	if err != nil {
		return fmt.Errorf("decrypt user dek: %w", err)
	}

	encData, nonce, err := s.encryptor.Encrypt(dek, item.Data)
	if err != nil {
		return fmt.Errorf("encrypt json data: %w", err)
	}

	itemEncrypted := &model.ItemEncrypted{
		ItemMeta:   item.ItemMeta,
		Ciphertext: encData,
		Nonce:      nonce,
	}

	// For file items, preserve S3 object information
	if item.Type == model.ItemTypeFile {
		slog.Debug("updating file item, fetching existing metadata",
			slog.String("title", title),
			slog.Int("userID", userID))
		existingItem, err := s.repo.GetItemByTitle(ctx, userID, title)
		if err != nil {
			return fmt.Errorf("get existing file item: %w", err)
		}
		// Preserve S3 object metadata
		itemEncrypted.ObjectKey = existingItem.ObjectKey
		itemEncrypted.ChunkSize = existingItem.ChunkSize
		itemEncrypted.NoncePrefix = existingItem.NoncePrefix
		slog.Debug("preserved file metadata",
			slog.String("objectKey", itemEncrypted.ObjectKey),
			slog.Int("chunkSize", itemEncrypted.ChunkSize),
			slog.Int("noncePrefixLen", len(itemEncrypted.NoncePrefix)))
	}

	if err := s.repo.UpdateItem(ctx, userID, title, itemEncrypted); err != nil {
		return err
	}

	return nil
}
