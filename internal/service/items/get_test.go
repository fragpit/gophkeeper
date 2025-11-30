package items

import (
	"context"
	"errors"
	"testing"

	"github.com/fragpit/gophkeeper/internal/model"
	authMocks "github.com/fragpit/gophkeeper/internal/service/auth/mocks"
	mocks "github.com/fragpit/gophkeeper/internal/service/items/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetService_GetItemByTitle(t *testing.T) {
	type args struct {
		userID int
		title  string
	}
	tests := []struct {
		name    string
		args    args
		prepare func(*mocks.MockItemsRepository, *authMocks.MockUsersRepository, *mocks.MockEncryptor, context.Context, args)
		wantErr bool
	}{
		{
			name: "repository error",
			args: args{userID: 1, title: "item1"},
			prepare: func(r *mocks.MockItemsRepository, _ *authMocks.MockUsersRepository, _ *mocks.MockEncryptor, ctx context.Context, a args) {
				r.EXPECT().GetItemByTitle(ctx, a.userID, a.title).
					Return(nil, errors.New("db error"))
			},
			wantErr: true,
		},
		{
			name: "user dek error",
			args: args{userID: 1, title: "item1"},
			prepare: func(r *mocks.MockItemsRepository, ur *authMocks.MockUsersRepository, _ *mocks.MockEncryptor, ctx context.Context, a args) {
				itemEncrypted := &model.ItemEncrypted{
					ItemMeta: &model.ItemMeta{
						Title: "item1",
						Type:  model.ItemTypeLogin,
					},
					Ciphertext: []byte("encrypted"),
					Nonce:      []byte("nonce"),
				}
				r.EXPECT().GetItemByTitle(ctx, a.userID, a.title).
					Return(itemEncrypted, nil)
				ur.EXPECT().GetUserDEK(ctx, a.userID).
					Return(nil, errors.New("dek not found"))
			},
			wantErr: true,
		},
		{
			name: "decrypt dek error",
			args: args{userID: 1, title: "item1"},
			prepare: func(r *mocks.MockItemsRepository, ur *authMocks.MockUsersRepository, enc *mocks.MockEncryptor, ctx context.Context, a args) {
				itemEncrypted := &model.ItemEncrypted{
					ItemMeta: &model.ItemMeta{
						Title: "item1",
						Type:  model.ItemTypeLogin,
					},
					Ciphertext: []byte("encrypted"),
					Nonce:      []byte("nonce"),
				}
				edek := &model.EncryptedDEK{
					EncryptedKey: []byte("encrypted_dek"),
					Nonce:        []byte("dek_nonce"),
				}
				r.EXPECT().GetItemByTitle(ctx, a.userID, a.title).
					Return(itemEncrypted, nil)
				ur.EXPECT().GetUserDEK(ctx, a.userID).
					Return(edek, nil)
				enc.EXPECT().DecryptDEK(edek).
					Return(nil, errors.New("decryption failed"))
			},
			wantErr: true,
		},
		{
			name: "decrypt item error",
			args: args{userID: 1, title: "item1"},
			prepare: func(r *mocks.MockItemsRepository, ur *authMocks.MockUsersRepository, enc *mocks.MockEncryptor, ctx context.Context, a args) {
				itemEncrypted := &model.ItemEncrypted{
					ItemMeta: &model.ItemMeta{
						Title: "item1",
						Type:  model.ItemTypeLogin,
					},
					Ciphertext: []byte("encrypted"),
					Nonce:      []byte("nonce"),
				}
				edek := &model.EncryptedDEK{
					EncryptedKey: []byte("encrypted_dek"),
					Nonce:        []byte("dek_nonce"),
				}
				dek := []byte("decrypted_dek")
				r.EXPECT().GetItemByTitle(ctx, a.userID, a.title).
					Return(itemEncrypted, nil)
				ur.EXPECT().GetUserDEK(ctx, a.userID).
					Return(edek, nil)
				enc.EXPECT().DecryptDEK(edek).
					Return(dek, nil)
				enc.EXPECT().
					Decrypt(dek, itemEncrypted.Nonce, itemEncrypted.Ciphertext).
					Return(nil, errors.New("decryption failed"))
			},
			wantErr: true,
		},
		{
			name: "success",
			args: args{userID: 1, title: "item1"},
			prepare: func(r *mocks.MockItemsRepository, ur *authMocks.MockUsersRepository, enc *mocks.MockEncryptor, ctx context.Context, a args) {
				itemEncrypted := &model.ItemEncrypted{
					ItemMeta: &model.ItemMeta{
						Title: "item1",
						Type:  model.ItemTypeLogin,
					},
					Ciphertext: []byte("encrypted"),
					Nonce:      []byte("nonce"),
				}
				edek := &model.EncryptedDEK{
					EncryptedKey: []byte("encrypted_dek"),
					Nonce:        []byte("dek_nonce"),
				}
				dek := []byte("decrypted_dek")
				decryptedData := []byte(`{"username":"user","password":"pass"}`)
				r.EXPECT().GetItemByTitle(ctx, a.userID, a.title).
					Return(itemEncrypted, nil)
				ur.EXPECT().GetUserDEK(ctx, a.userID).
					Return(edek, nil)
				enc.EXPECT().DecryptDEK(edek).
					Return(dek, nil)
				enc.EXPECT().
					Decrypt(dek, itemEncrypted.Nonce, itemEncrypted.Ciphertext).
					Return(decryptedData, nil)
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
			svc := NewGetItemService(repo, userRepo, encryptor)

			tt.prepare(repo, userRepo, encryptor, ctx, tt.args)

			item, err := svc.GetItemByTitle(ctx, tt.args.userID, tt.args.title)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, item)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, item)
				assert.Equal(t, tt.args.title, item.Title)
			}
		})
	}
}
