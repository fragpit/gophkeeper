package items

import (
	"context"
	"fmt"

	"github.com/fragpit/gophkeeper/internal/api/handlers"
	"github.com/fragpit/gophkeeper/internal/model"
)

var _ handlers.GetService = (*GetService)(nil)

type GetService struct {
	repo      model.ItemsRepository
	userRepo  model.UsersRepository
	encryptor Encryptor
}

func NewGetItemService(
	repo model.ItemsRepository,
	userRepo model.UsersRepository,
	encryptor Encryptor,
) *GetService {
	return &GetService{
		repo:      repo,
		userRepo:  userRepo,
		encryptor: encryptor,
	}
}

func (s *GetService) GetItemByTitle(
	ctx context.Context,
	userID int,
	title string,
) (*model.ItemDecrypted, error) {
	itemEncrypted, err := s.repo.GetItemByTitle(ctx, userID, title)
	if err != nil {
		return nil, err
	}

	edek, err := s.userRepo.GetUserDEK(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user dek: %w", err)
	}

	dek, err := s.encryptor.DecryptDEK(edek)
	if err != nil {
		return nil, fmt.Errorf("decrypt dek: %w", err)
	}

	data, err := s.encryptor.Decrypt(
		dek,
		itemEncrypted.Nonce,
		itemEncrypted.Ciphertext,
	)
	if err != nil {
		return nil, fmt.Errorf("decrypt item: %w", err)
	}

	item := &model.ItemDecrypted{
		ItemMeta: itemEncrypted.ItemMeta,
		Data:     data,
	}

	return item, nil
}
