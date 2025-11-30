package items

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/fragpit/gophkeeper/internal/model"
	authMocks "github.com/fragpit/gophkeeper/internal/service/auth/mocks"
	mocks "github.com/fragpit/gophkeeper/internal/service/items/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetFileService_GetItemByTitle(t *testing.T) {
	type args struct {
		userID int
		title  string
	}
	tests := []struct {
		name    string
		args    args
		prepare func(*mocks.MockItemsRepository, context.Context, args)
		wantErr bool
	}{
		{
			name: "repository error",
			args: args{userID: 1, title: "file1.txt"},
			prepare: func(r *mocks.MockItemsRepository, ctx context.Context, a args) {
				r.EXPECT().GetItemByTitle(ctx, a.userID, a.title).
					Return(nil, errors.New("db error"))
			},
			wantErr: true,
		},
		{
			name: "success",
			args: args{userID: 1, title: "file1.txt"},
			prepare: func(r *mocks.MockItemsRepository, ctx context.Context, a args) {
				item := &model.ItemEncrypted{
					ItemMeta: &model.ItemMeta{
						Title: "file1.txt",
						Type:  model.ItemTypeFile,
					},
					ObjectKey: "objectkey123",
				}
				r.EXPECT().GetItemByTitle(ctx, a.userID, a.title).
					Return(item, nil)
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
			crypt := mocks.NewMockFileDecryptor(ctrl)
			store := mocks.NewMockObjectStorer(ctrl)
			ctx := context.Background()
			svc := NewGetFileService(repo, userRepo, crypt, store)

			tt.prepare(repo, ctx, tt.args)

			item, err := svc.GetItemByTitle(ctx, tt.args.userID, tt.args.title)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, item)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, item)
			}
		})
	}
}

func TestGetFileService_GetFileByItem(t *testing.T) {
	type args struct {
		userID int
		item   *model.ItemEncrypted
	}
	tests := []struct {
		name    string
		args    args
		prepare func(*authMocks.MockUsersRepository, *mocks.MockFileDecryptor, *mocks.MockObjectStorer, context.Context, args)
		wantErr bool
	}{
		{
			name: "nil item",
			args: args{userID: 1, item: nil},
			prepare: func(_ *authMocks.MockUsersRepository, _ *mocks.MockFileDecryptor, _ *mocks.MockObjectStorer, _ context.Context, _ args) {
			},
			wantErr: true,
		},
		{
			name: "nil item meta",
			args: args{
				userID: 1,
				item:   &model.ItemEncrypted{ItemMeta: nil},
			},
			prepare: func(_ *authMocks.MockUsersRepository, _ *mocks.MockFileDecryptor, _ *mocks.MockObjectStorer, _ context.Context, _ args) {
			},
			wantErr: true,
		},
		{
			name: "invalid item type",
			args: args{
				userID: 1,
				item: &model.ItemEncrypted{
					ItemMeta: &model.ItemMeta{Title: "item1", Type: model.ItemTypeLogin},
				},
			},
			prepare: func(_ *authMocks.MockUsersRepository, _ *mocks.MockFileDecryptor, _ *mocks.MockObjectStorer, _ context.Context, _ args) {
			},
			wantErr: true,
		},
		{
			name: "empty object key",
			args: args{
				userID: 1,
				item: &model.ItemEncrypted{
					ItemMeta: &model.ItemMeta{
						Title: "file1.txt",
						Type:  model.ItemTypeFile,
					},
					ObjectKey: "",
				},
			},
			prepare: func(_ *authMocks.MockUsersRepository, _ *mocks.MockFileDecryptor, _ *mocks.MockObjectStorer, _ context.Context, _ args) {
			},
			wantErr: true,
		},
		{
			name: "empty nonce prefix",
			args: args{
				userID: 1,
				item: &model.ItemEncrypted{
					ItemMeta: &model.ItemMeta{
						Title: "file1.txt",
						Type:  model.ItemTypeFile,
					},
					ObjectKey:   "objectkey123",
					NoncePrefix: nil,
				},
			},
			prepare: func(_ *authMocks.MockUsersRepository, _ *mocks.MockFileDecryptor, _ *mocks.MockObjectStorer, _ context.Context, _ args) {
			},
			wantErr: true,
		},
		{
			name: "empty chunk size",
			args: args{
				userID: 1,
				item: &model.ItemEncrypted{
					ItemMeta: &model.ItemMeta{
						Title: "file1.txt",
						Type:  model.ItemTypeFile,
					},
					ObjectKey:   "objectkey123",
					NoncePrefix: []byte("nonce"),
					ChunkSize:   0,
				},
			},
			prepare: func(_ *authMocks.MockUsersRepository, _ *mocks.MockFileDecryptor, _ *mocks.MockObjectStorer, _ context.Context, _ args) {
			},
			wantErr: true,
		},
		{
			name: "user dek error",
			args: args{
				userID: 1,
				item: &model.ItemEncrypted{
					ItemMeta: &model.ItemMeta{
						Title: "file1.txt",
						Type:  model.ItemTypeFile,
					},
					ObjectKey:   "objectkey123",
					NoncePrefix: []byte("nonce"),
					ChunkSize:   1024,
				},
			},
			prepare: func(ur *authMocks.MockUsersRepository, _ *mocks.MockFileDecryptor, _ *mocks.MockObjectStorer, ctx context.Context, a args) {
				ur.EXPECT().GetUserDEK(ctx, a.userID).
					Return(nil, errors.New("dek not found"))
			},
			wantErr: true,
		},
		{
			name: "decrypt dek error",
			args: args{
				userID: 1,
				item: &model.ItemEncrypted{
					ItemMeta: &model.ItemMeta{
						Title: "file1.txt",
						Type:  model.ItemTypeFile,
					},
					ObjectKey:   "objectkey123",
					NoncePrefix: []byte("nonce"),
					ChunkSize:   1024,
				},
			},
			prepare: func(ur *authMocks.MockUsersRepository, crypt *mocks.MockFileDecryptor, _ *mocks.MockObjectStorer, ctx context.Context, a args) {
				edek := &model.EncryptedDEK{
					EncryptedKey: []byte("encrypted_dek"),
					Nonce:        []byte("dek_nonce"),
				}
				ur.EXPECT().GetUserDEK(ctx, a.userID).
					Return(edek, nil)
				crypt.EXPECT().DecryptDEK(edek).
					Return(nil, errors.New("decryption failed"))
			},
			wantErr: true,
		},
		{
			name: "store get error",
			args: args{
				userID: 1,
				item: &model.ItemEncrypted{
					ItemMeta: &model.ItemMeta{
						Title: "file1.txt",
						Type:  model.ItemTypeFile,
					},
					ObjectKey:   "objectkey123",
					NoncePrefix: []byte("nonce"),
					ChunkSize:   1024,
				},
			},
			prepare: func(ur *authMocks.MockUsersRepository, crypt *mocks.MockFileDecryptor, store *mocks.MockObjectStorer, ctx context.Context, a args) {
				edek := &model.EncryptedDEK{
					EncryptedKey: []byte("encrypted_dek"),
					Nonce:        []byte("dek_nonce"),
				}
				dek := []byte("decrypted_dek")
				ur.EXPECT().GetUserDEK(ctx, a.userID).
					Return(edek, nil)
				crypt.EXPECT().DecryptDEK(edek).
					Return(dek, nil)
				store.EXPECT().Get(gomock.Any(), "gophkeeper", a.item.ObjectKey).
					Return(nil, errors.New("s3 error"))
			},
			wantErr: true,
		},
		{
			name: "success",
			args: args{
				userID: 1,
				item: &model.ItemEncrypted{
					ItemMeta: &model.ItemMeta{
						Title: "file1.txt",
						Type:  model.ItemTypeFile,
					},
					ObjectKey:   "objectkey123",
					NoncePrefix: []byte("nonce"),
					ChunkSize:   1024,
				},
			},
			prepare: func(ur *authMocks.MockUsersRepository, crypt *mocks.MockFileDecryptor, store *mocks.MockObjectStorer, ctx context.Context, a args) {
				edek := &model.EncryptedDEK{
					EncryptedKey: []byte("encrypted_dek"),
					Nonce:        []byte("dek_nonce"),
				}
				dek := []byte("decrypted_dek")
				fileContent := io.NopCloser(strings.NewReader("encrypted file content"))
				ur.EXPECT().GetUserDEK(ctx, a.userID).
					Return(edek, nil)
				crypt.EXPECT().DecryptDEK(edek).
					Return(dek, nil)
				store.EXPECT().Get(gomock.Any(), "gophkeeper", a.item.ObjectKey).
					Return(fileContent, nil)
				crypt.EXPECT().
					DecryptFileStream(dek, gomock.Any(), gomock.Any(), a.item.ChunkSize, a.item.NoncePrefix, []byte(a.item.ObjectKey)).
					Return(nil).AnyTimes()
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
			crypt := mocks.NewMockFileDecryptor(ctrl)
			store := mocks.NewMockObjectStorer(ctrl)
			ctx := context.Background()
			svc := NewGetFileService(repo, userRepo, crypt, store)

			tt.prepare(userRepo, crypt, store, ctx, tt.args)

			reader, err := svc.GetFileByItem(ctx, tt.args.userID, tt.args.item)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, reader)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, reader)
				if reader != nil {
					_ = reader.Close()
				}
			}
		})
	}
}

func TestGetFileService_GetFileByItem_ValidationErrors(t *testing.T) {
	ctx := context.Background()
	userID := 1

	tests := []struct {
		name string
		item *model.ItemEncrypted
		want error
	}{
		{
			name: "nil item returns ErrInvalidItemData",
			item: nil,
			want: ErrInvalidItemData,
		},
		{
			name: "nil ItemMeta returns ErrInvalidItemData",
			item: &model.ItemEncrypted{},
			want: ErrInvalidItemData,
		},
		{
			name: "wrong type returns ErrInvalidItemType",
			item: &model.ItemEncrypted{
				ItemMeta: &model.ItemMeta{Type: model.ItemTypeLogin},
			},
			want: ErrInvalidItemType,
		},
		{
			name: "empty object key returns ErrEmptyObjectKey",
			item: &model.ItemEncrypted{
				ItemMeta:  &model.ItemMeta{Type: model.ItemTypeFile},
				ObjectKey: "",
			},
			want: ErrEmptyObjectKey,
		},
		{
			name: "empty nonce prefix returns ErrEmptyNoncePrefix",
			item: &model.ItemEncrypted{
				ItemMeta:    &model.ItemMeta{Type: model.ItemTypeFile},
				ObjectKey:   "key",
				NoncePrefix: nil,
			},
			want: ErrEmptyNoncePrefix,
		},
		{
			name: "zero chunk size returns ErrEmptyChunkSize",
			item: &model.ItemEncrypted{
				ItemMeta:    &model.ItemMeta{Type: model.ItemTypeFile},
				ObjectKey:   "key",
				NoncePrefix: []byte("nonce"),
				ChunkSize:   0,
			},
			want: ErrEmptyChunkSize,
		},
		{
			name: "negative chunk size returns ErrEmptyChunkSize",
			item: &model.ItemEncrypted{
				ItemMeta:    &model.ItemMeta{Type: model.ItemTypeFile},
				ObjectKey:   "key",
				NoncePrefix: []byte("nonce"),
				ChunkSize:   -100,
			},
			want: ErrEmptyChunkSize,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewGetFileService(nil, nil, nil, nil)

			_, err := svc.GetFileByItem(ctx, userID, tt.item)

			require.Error(t, err)
			assert.ErrorIs(t, err, tt.want)
		})
	}
}
