package items

import (
	"context"

	"github.com/fragpit/gophkeeper/internal/api/handlers"
	"github.com/fragpit/gophkeeper/internal/model"
)

var _ handlers.ItemsService = (*ItemsService)(nil)

// ItemsService provides read-only item listing operations.
type ItemsService struct {
	repo model.ItemsRepository
}

// NewItemsService creates a new ItemsService with the provided repository.
func NewItemsService(
	repo model.ItemsRepository,
) *ItemsService {
	return &ItemsService{
		repo: repo,
	}
}

// List returns metadata of items for the specified user and type.
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
