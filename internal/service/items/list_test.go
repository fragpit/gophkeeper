package items

import (
	"context"
	"errors"
	"testing"

	"github.com/fragpit/gophkeeper/internal/model"
	mocks "github.com/fragpit/gophkeeper/internal/service/items/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestItemsService_List(t *testing.T) {
	type args struct {
		userID int
		iType  model.ItemType
	}
	tests := []struct {
		name      string
		args      args
		prepare   func(*mocks.MockItemsRepository, context.Context, args)
		wantErr   error
		wantItems int
	}{
		{
			name: "repository error",
			args: args{userID: 1, iType: model.ItemTypeLogin},
			prepare: func(r *mocks.MockItemsRepository, ctx context.Context, a args) {
				dbErr := errors.New("db error")
				r.EXPECT().List(ctx, a.userID, a.iType).
					Return(nil, dbErr)
			},
			wantErr:   errors.New("db error"),
			wantItems: 0,
		},
		{
			name: "success with items",
			args: args{userID: 1, iType: model.ItemTypeLogin},
			prepare: func(r *mocks.MockItemsRepository, ctx context.Context, a args) {
				items := []model.ItemMeta{
					{Title: "item1", Type: model.ItemTypeLogin},
					{Title: "item2", Type: model.ItemTypeLogin},
				}
				r.EXPECT().List(ctx, a.userID, a.iType).
					Return(items, nil)
			},
			wantErr:   nil,
			wantItems: 2,
		},
		{
			name: "success with empty list",
			args: args{userID: 1, iType: model.ItemTypeNote},
			prepare: func(r *mocks.MockItemsRepository, ctx context.Context, a args) {
				r.EXPECT().List(ctx, a.userID, a.iType).
					Return([]model.ItemMeta{}, nil)
			},
			wantErr:   nil,
			wantItems: 0,
		},
		{
			name: "success with file type",
			args: args{userID: 2, iType: model.ItemTypeFile},
			prepare: func(r *mocks.MockItemsRepository, ctx context.Context, a args) {
				items := []model.ItemMeta{
					{Title: "file1.txt", Type: model.ItemTypeFile},
				}
				r.EXPECT().List(ctx, a.userID, a.iType).
					Return(items, nil)
			},
			wantErr:   nil,
			wantItems: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repo := mocks.NewMockItemsRepository(ctrl)
			ctx := context.Background()
			svc := NewItemsService(repo)

			tt.prepare(repo, ctx, tt.args)

			items, err := svc.List(ctx, tt.args.userID, tt.args.iType)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.Nil(t, items)
			} else {
				assert.NoError(t, err)
				assert.Len(t, items, tt.wantItems)
			}
		})
	}
}
