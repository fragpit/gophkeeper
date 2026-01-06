package items

import (
	"context"
	"errors"
	"fmt"

	"github.com/fragpit/gophkeeper/internal/api/handlers"
	"github.com/fragpit/gophkeeper/internal/model"
)

var _ handlers.CreateService = (*CreateService)(nil)

// CreateService handles creation of encrypted items.
type CreateService struct {
	repo      model.ItemsRepository
	userRepo  model.UsersRepository
	encryptor Encryptor
}

// NewCreateItemService constructs a CreateService instance.
func NewCreateItemService(
	repo model.ItemsRepository,
	userRepo model.UsersRepository,
	encryptor Encryptor,
) *CreateService {
	return &CreateService{
		repo:      repo,
		userRepo:  userRepo,
		encryptor: encryptor,
	}
}

// CreateItem encrypts and stores a decrypted item for the specified user.
func (s *CreateService) CreateItem(
	ctx context.Context,
	userID int,
	item *model.ItemDecrypted,
) (int, error) {
	// TODO: validation
	if !item.Type.Valid() {
		return 0, errors.New("unknown item type")
	}

	edek, err := s.userRepo.GetUserDEK(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("get user dek: %w", err)
	}

	dek, err := s.encryptor.DecryptDEK(edek)
	if err != nil {
		return 0, fmt.Errorf("decrypt user dek: %w", err)
	}

	encData, nonce, err := s.encryptor.Encrypt(dek, item.Data)
	if err != nil {
		return 0, fmt.Errorf("encrypt json data: %w", err)
	}

	itemEncrypted := &model.ItemEncrypted{
		ItemMeta:   item.ItemMeta,
		Ciphertext: encData,
		Nonce:      nonce,
	}

	id, err := s.repo.CreateItem(
		ctx,
		userID,
		itemEncrypted,
	)
	if err != nil {
		return 0, err
	}

	return id, nil
}
