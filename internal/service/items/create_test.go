package items

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/fragpit/gophkeeper/internal/model"
	authMocks "github.com/fragpit/gophkeeper/internal/service/auth/mocks"
	mocks "github.com/fragpit/gophkeeper/internal/service/items/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCreateService_CreateItem(t *testing.T) {
	type args struct {
		userID int
		item   *model.ItemDecrypted
	}
	tests := []struct {
		name    string
		args    args
		prepare func(*mocks.MockItemsRepository, *authMocks.MockUsersRepository, *mocks.MockEncryptor, context.Context, args)
		wantErr bool
		wantID  int
	}{
		{
			name: "user dek error",
			args: args{
				userID: 1,
				item: &model.ItemDecrypted{
					ItemMeta: &model.ItemMeta{Title: "item1", Type: model.ItemTypeLogin},
					Data:     json.RawMessage(`{"username":"user"}`),
				},
			},
			prepare: func(_ *mocks.MockItemsRepository, ur *authMocks.MockUsersRepository, _ *mocks.MockEncryptor, ctx context.Context, a args) {
				ur.EXPECT().GetUserDEK(ctx, a.userID).
					Return(nil, errors.New("dek not found"))
			},
			wantErr: true,
			wantID:  0,
		},
		{
			name: "decrypt dek error",
			args: args{
				userID: 1,
				item: &model.ItemDecrypted{
					ItemMeta: &model.ItemMeta{Title: "item1", Type: model.ItemTypeLogin},
					Data:     json.RawMessage(`{"username":"user"}`),
				},
			},
			prepare: func(_ *mocks.MockItemsRepository, ur *authMocks.MockUsersRepository, enc *mocks.MockEncryptor, ctx context.Context, a args) {
				edek := &model.EncryptedDEK{
					EncryptedKey: []byte("encrypted_dek"),
					Nonce:        []byte("dek_nonce"),
				}
				ur.EXPECT().GetUserDEK(ctx, a.userID).
					Return(edek, nil)
				enc.EXPECT().DecryptDEK(edek).
					Return(nil, errors.New("decryption failed"))
			},
			wantErr: true,
			wantID:  0,
		},
		{
			name: "encrypt data error",
			args: args{
				userID: 1,
				item: &model.ItemDecrypted{
					ItemMeta: &model.ItemMeta{Title: "item1", Type: model.ItemTypeLogin},
					Data:     json.RawMessage(`{"username":"user"}`),
				},
			},
			prepare: func(_ *mocks.MockItemsRepository, ur *authMocks.MockUsersRepository, enc *mocks.MockEncryptor, ctx context.Context, a args) {
				edek := &model.EncryptedDEK{
					EncryptedKey: []byte("encrypted_dek"),
					Nonce:        []byte("dek_nonce"),
				}
				dek := []byte("decrypted_dek")
				ur.EXPECT().GetUserDEK(ctx, a.userID).
					Return(edek, nil)
				enc.EXPECT().DecryptDEK(edek).
					Return(dek, nil)
				enc.EXPECT().Encrypt(dek, a.item.Data).
					Return(nil, nil, errors.New("encryption failed"))
			},
			wantErr: true,
			wantID:  0,
		},
		{
			name: "repository error",
			args: args{
				userID: 1,
				item: &model.ItemDecrypted{
					ItemMeta: &model.ItemMeta{Title: "item1", Type: model.ItemTypeLogin},
					Data:     json.RawMessage(`{"username":"user"}`),
				},
			},
			prepare: func(r *mocks.MockItemsRepository, ur *authMocks.MockUsersRepository, enc *mocks.MockEncryptor, ctx context.Context, a args) {
				edek := &model.EncryptedDEK{
					EncryptedKey: []byte("encrypted_dek"),
					Nonce:        []byte("dek_nonce"),
				}
				dek := []byte("decrypted_dek")
				encryptedData := []byte("encrypted_data")
				nonce := []byte("nonce")
				ur.EXPECT().GetUserDEK(ctx, a.userID).
					Return(edek, nil)
				enc.EXPECT().DecryptDEK(edek).
					Return(dek, nil)
				enc.EXPECT().Encrypt(dek, a.item.Data).
					Return(encryptedData, nonce, nil)
				r.EXPECT().CreateItem(ctx, a.userID, gomock.Any()).
					Return(0, errors.New("db error"))
			},
			wantErr: true,
			wantID:  0,
		},
		{
			name: "success",
			args: args{
				userID: 1,
				item: &model.ItemDecrypted{
					ItemMeta: &model.ItemMeta{Title: "item1", Type: model.ItemTypeLogin},
					Data:     json.RawMessage(`{"username":"user"}`),
				},
			},
			prepare: func(r *mocks.MockItemsRepository, ur *authMocks.MockUsersRepository, enc *mocks.MockEncryptor, ctx context.Context, a args) {
				edek := &model.EncryptedDEK{
					EncryptedKey: []byte("encrypted_dek"),
					Nonce:        []byte("dek_nonce"),
				}
				dek := []byte("decrypted_dek")
				encryptedData := []byte("encrypted_data")
				nonce := []byte("nonce")
				ur.EXPECT().GetUserDEK(ctx, a.userID).
					Return(edek, nil)
				enc.EXPECT().DecryptDEK(edek).
					Return(dek, nil)
				enc.EXPECT().Encrypt(dek, a.item.Data).
					Return(encryptedData, nonce, nil)
				r.EXPECT().CreateItem(ctx, a.userID, gomock.Any()).
					Return(42, nil)
			},
			wantErr: false,
			wantID:  42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repo := mocks.NewMockItemsRepository(ctrl)
			userRepo := authMocks.NewMockUsersRepository(ctrl)
			encryptor := mocks.NewMockEncryptor(ctrl)
			ctx := context.Background()
			svc := NewCreateItemService(repo, userRepo, encryptor)

			tt.prepare(repo, userRepo, encryptor, ctx, tt.args)

			id, err := svc.CreateItem(ctx, tt.args.userID, tt.args.item)

			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, 0, id)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantID, id)
			}
		})
	}
}
