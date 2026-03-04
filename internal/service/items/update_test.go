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

func TestUpdateService_UpdateItem(t *testing.T) {
	type args struct {
		userID int
		title  string
		item   *model.ItemDecrypted
	}
	tests := []struct {
		name    string
		args    args
		prepare func(*mocks.MockItemsRepository, *authMocks.MockUsersRepository, *mocks.MockEncryptor, context.Context, args)
		wantErr bool
	}{
		{
			name: "user dek error",
			args: args{
				userID: 1,
				title:  "item1",
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
		},
		{
			name: "decrypt dek error",
			args: args{
				userID: 1,
				title:  "item1",
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
		},
		{
			name: "encrypt data error",
			args: args{
				userID: 1,
				title:  "item1",
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
		},
		{
			name: "repository error",
			args: args{
				userID: 1,
				title:  "item1",
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
				r.EXPECT().UpdateItem(ctx, a.userID, a.title, gomock.Any()).
					Return(errors.New("db error"))
			},
			wantErr: true,
		},
		{
			name: "success",
			args: args{
				userID: 1,
				title:  "item1",
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
				r.EXPECT().UpdateItem(ctx, a.userID, a.title, gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "success with different title",
			args: args{
				userID: 2,
				title:  "oldtitle",
				item: &model.ItemDecrypted{
					ItemMeta: &model.ItemMeta{
						Title: "newtitle",
						Type:  model.ItemTypeNote,
					},
					Data: json.RawMessage(`{"text":"secret"}`),
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
				r.EXPECT().UpdateItem(ctx, a.userID, a.title, gomock.Any()).
					Return(nil)
			},
			wantErr: false,
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
			svc := NewUpdateItemService(repo, userRepo, encryptor)

			tt.prepare(repo, userRepo, encryptor, ctx, tt.args)

			err := svc.UpdateItem(ctx, tt.args.userID, tt.args.title, tt.args.item)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
