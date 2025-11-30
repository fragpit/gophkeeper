package items

import (
	"context"

	"github.com/fragpit/gophkeeper/internal/api/handlers"
	"github.com/fragpit/gophkeeper/internal/model"
)

var _ handlers.ItemsService = (*ItemsService)(nil)

type ItemsService struct {
	repo model.ItemsRepository
}

func NewItemsService(
	repo model.ItemsRepository,
) *ItemsService {
	return &ItemsService{
		repo: repo,
	}
}

func (s *ItemsService) List(
	ctx context.Context,
	userID int,
	iType model.ItemType,
) ([]model.ItemMeta, error) {
	items, err := s.repo.List(ctx, userID, iType)
	if err != nil {
		return nil, model.ErrUserExists
	}

	return items, nil
}
