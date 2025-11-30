package items

import (
	"context"
	"fmt"

	"github.com/fragpit/gophkeeper/internal/api/handlers"
	"github.com/fragpit/gophkeeper/internal/model"
)

var _ handlers.DeleteService = (*DeleteService)(nil)

type DeleteService struct {
	repo         model.ItemsRepository
	objectStorer ObjectStorer
}

func NewDeleteService(
	repo model.ItemsRepository,
	objectStorer ObjectStorer,
) *DeleteService {
	return &DeleteService{
		repo:         repo,
		objectStorer: objectStorer,
	}
}

func (s *DeleteService) DeleteItem(
	ctx context.Context,
	userID int,
	title string,
) error {
	item, err := s.repo.GetItemByTitle(ctx, userID, title)
	if err != nil {
		return fmt.Errorf("get item: %w", err)
	}

	if item.Type == model.ItemTypeFile && item.ObjectKey != "" {
		if err := s.objectStorer.Delete(ctx, "gophkeeper", item.ObjectKey); err != nil {
			return fmt.Errorf("delete object from storage: %w", err)
		}
	}

	if err := s.repo.DeleteItem(ctx, userID, title); err != nil {
		return fmt.Errorf("delete item from database: %w", err)
	}

	return nil
}
