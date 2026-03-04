package items

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"strings"
	"testing"

	"github.com/fragpit/gophkeeper/internal/model"
	authMocks "github.com/fragpit/gophkeeper/internal/service/auth/mocks"
	mocks "github.com/fragpit/gophkeeper/internal/service/items/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type mockMultipartFile struct {
	io.Reader
}

func (m *mockMultipartFile) Close() error { return nil }

func (m *mockMultipartFile) Read(
	p []byte,
) (n int, err error) {
	return m.Reader.Read(p)
}

func (m *mockMultipartFile) ReadAt(
	p []byte,
	off int64,
) (n int, err error) {
	return 0, nil
}

func (m *mockMultipartFile) Seek(
	offset int64,
	whence int,
) (int64, error) {
	return 0, nil
}

func TestCreateFileService_CreateFileItem(t *testing.T) {
	type args struct {
		userID      int
		item        *model.ItemDecrypted
		file        multipart.File
		contentType string
	}
	tests := []struct {
		name    string
		args    args
		prepare func(*mocks.MockItemsRepository, *authMocks.MockUsersRepository, *mocks.MockFileEncryptor, *mocks.MockObjectStorer, context.Context, args)
		wantErr bool
		wantID  int
	}{
		{
			name: "nil item",
			args: args{
				userID:      1,
				item:        nil,
				file:        &mockMultipartFile{Reader: strings.NewReader("content")},
				contentType: "text/plain",
			},
			prepare: func(_ *mocks.MockItemsRepository, _ *authMocks.MockUsersRepository, _ *mocks.MockFileEncryptor, _ *mocks.MockObjectStorer, _ context.Context, _ args) {
			},
			wantErr: true,
			wantID:  0,
		},
		{
			name: "nil item meta",
			args: args{
				userID:      1,
				item:        &model.ItemDecrypted{ItemMeta: nil},
				file:        &mockMultipartFile{Reader: strings.NewReader("content")},
				contentType: "text/plain",
			},
			prepare: func(_ *mocks.MockItemsRepository, _ *authMocks.MockUsersRepository, _ *mocks.MockFileEncryptor, _ *mocks.MockObjectStorer, _ context.Context, _ args) {
			},
			wantErr: true,
			wantID:  0,
		},
		{
			name: "invalid item type",
			args: args{
				userID: 1,
				item: &model.ItemDecrypted{
					ItemMeta: &model.ItemMeta{Title: "item1", Type: model.ItemTypeLogin},
				},
				file:        &mockMultipartFile{Reader: strings.NewReader("content")},
				contentType: "text/plain",
			},
			prepare: func(_ *mocks.MockItemsRepository, _ *authMocks.MockUsersRepository, _ *mocks.MockFileEncryptor, _ *mocks.MockObjectStorer, _ context.Context, _ args) {
			},
			wantErr: true,
			wantID:  0,
		},
		{
			name: "invalid file data json",
			args: args{
				userID: 1,
				item: &model.ItemDecrypted{
					ItemMeta: &model.ItemMeta{
						Title: "file1.txt",
						Type:  model.ItemTypeFile,
					},
					Data: json.RawMessage(`{invalid json`),
				},
				file:        &mockMultipartFile{Reader: strings.NewReader("content")},
				contentType: "text/plain",
			},
			prepare: func(_ *mocks.MockItemsRepository, _ *authMocks.MockUsersRepository, _ *mocks.MockFileEncryptor, _ *mocks.MockObjectStorer, _ context.Context, _ args) {
			},
			wantErr: true,
			wantID:  0,
		},
		{
			name: "nonce prefix error",
			args: args{
				userID: 1,
				item: &model.ItemDecrypted{
					ItemMeta: &model.ItemMeta{
						Title: "file1.txt",
						Type:  model.ItemTypeFile,
					},
					Data: json.RawMessage(`{}`),
				},
				file:        &mockMultipartFile{Reader: strings.NewReader("content")},
				contentType: "text/plain",
			},
			prepare: func(_ *mocks.MockItemsRepository, _ *authMocks.MockUsersRepository, enc *mocks.MockFileEncryptor, _ *mocks.MockObjectStorer, _ context.Context, _ args) {
				enc.EXPECT().NewFileNoncePrefix().
					Return(nil, errors.New("nonce generation failed"))
			},
			wantErr: true,
			wantID:  0,
		},
		{
			name: "user dek error",
			args: args{
				userID: 1,
				item: &model.ItemDecrypted{
					ItemMeta: &model.ItemMeta{
						Title: "file1.txt",
						Type:  model.ItemTypeFile,
					},
					Data: json.RawMessage(`{}`),
				},
				file:        &mockMultipartFile{Reader: strings.NewReader("content")},
				contentType: "text/plain",
			},
			prepare: func(_ *mocks.MockItemsRepository, ur *authMocks.MockUsersRepository, enc *mocks.MockFileEncryptor, _ *mocks.MockObjectStorer, ctx context.Context, a args) {
				enc.EXPECT().NewFileNoncePrefix().
					Return([]byte("nonce_prefix"), nil)
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
					ItemMeta: &model.ItemMeta{
						Title: "file1.txt",
						Type:  model.ItemTypeFile,
					},
					Data: json.RawMessage(`{}`),
				},
				file:        &mockMultipartFile{Reader: strings.NewReader("content")},
				contentType: "text/plain",
			},
			prepare: func(_ *mocks.MockItemsRepository, ur *authMocks.MockUsersRepository, enc *mocks.MockFileEncryptor, _ *mocks.MockObjectStorer, ctx context.Context, a args) {
				edek := &model.EncryptedDEK{
					EncryptedKey: []byte("encrypted_dek"),
					Nonce:        []byte("dek_nonce"),
				}
				enc.EXPECT().NewFileNoncePrefix().
					Return([]byte("nonce_prefix"), nil)
				ur.EXPECT().GetUserDEK(ctx, a.userID).
					Return(edek, nil)
				enc.EXPECT().DecryptDEK(edek).
					Return(nil, errors.New("decryption failed"))
			},
			wantErr: true,
			wantID:  0,
		},
		{
			name: "encrypt file stream error",
			args: args{
				userID: 1,
				item: &model.ItemDecrypted{
					ItemMeta: &model.ItemMeta{
						Title: "file1.txt",
						Type:  model.ItemTypeFile,
					},
					Data: json.RawMessage(`{}`),
				},
				file:        &mockMultipartFile{Reader: strings.NewReader("content")},
				contentType: "text/plain",
			},
			prepare: func(_ *mocks.MockItemsRepository, ur *authMocks.MockUsersRepository, enc *mocks.MockFileEncryptor, store *mocks.MockObjectStorer, ctx context.Context, a args) {
				edek := &model.EncryptedDEK{
					EncryptedKey: []byte("encrypted_dek"),
					Nonce:        []byte("dek_nonce"),
				}
				dek := []byte("decrypted_dek")
				enc.EXPECT().NewFileNoncePrefix().
					Return([]byte("nonce_prefix"), nil)
				ur.EXPECT().GetUserDEK(ctx, a.userID).
					Return(edek, nil)
				enc.EXPECT().DecryptDEK(edek).
					Return(dek, nil)
				enc.EXPECT().
					EncryptFileStream(dek, gomock.Any(), gomock.Any(), fileChunkSize, gomock.Any(), gomock.Any()).
					Return(int64(0), errors.New("encryption failed"))
				store.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("stream closed")).AnyTimes()
			},
			wantErr: true,
			wantID:  0,
		},
		{
			name: "store put error",
			args: args{
				userID: 1,
				item: &model.ItemDecrypted{
					ItemMeta: &model.ItemMeta{
						Title: "file1.txt",
						Type:  model.ItemTypeFile,
					},
					Data: json.RawMessage(`{}`),
				},
				file:        &mockMultipartFile{Reader: strings.NewReader("content")},
				contentType: "text/plain",
			},
			prepare: func(_ *mocks.MockItemsRepository, ur *authMocks.MockUsersRepository, enc *mocks.MockFileEncryptor, store *mocks.MockObjectStorer, ctx context.Context, a args) {
				edek := &model.EncryptedDEK{
					EncryptedKey: []byte("encrypted_dek"),
					Nonce:        []byte("dek_nonce"),
				}
				dek := []byte("decrypted_dek")
				enc.EXPECT().NewFileNoncePrefix().
					Return([]byte("nonce_prefix"), nil)
				ur.EXPECT().GetUserDEK(ctx, a.userID).
					Return(edek, nil)
				enc.EXPECT().DecryptDEK(edek).
					Return(dek, nil)
				enc.EXPECT().
					EncryptFileStream(dek, gomock.Any(), gomock.Any(), fileChunkSize, gomock.Any(), gomock.Any()).
					Return(int64(100), nil)
				store.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("s3 error"))
			},
			wantErr: true,
			wantID:  0,
		},
		{
			name: "repository error",
			args: args{
				userID: 1,
				item: &model.ItemDecrypted{
					ItemMeta: &model.ItemMeta{
						Title: "file1.txt",
						Type:  model.ItemTypeFile,
					},
					Data: json.RawMessage(`{}`),
				},
				file:        &mockMultipartFile{Reader: strings.NewReader("content")},
				contentType: "text/plain",
			},
			prepare: func(r *mocks.MockItemsRepository, ur *authMocks.MockUsersRepository, enc *mocks.MockFileEncryptor, store *mocks.MockObjectStorer, ctx context.Context, a args) {
				edek := &model.EncryptedDEK{
					EncryptedKey: []byte("encrypted_dek"),
					Nonce:        []byte("dek_nonce"),
				}
				dek := []byte("decrypted_dek")
				enc.EXPECT().NewFileNoncePrefix().
					Return([]byte("nonce_prefix"), nil)
				ur.EXPECT().GetUserDEK(ctx, a.userID).
					Return(edek, nil)
				enc.EXPECT().DecryptDEK(edek).
					Return(dek, nil)
				enc.EXPECT().
					EncryptFileStream(dek, gomock.Any(), gomock.Any(), fileChunkSize, gomock.Any(), gomock.Any()).
					Return(int64(100), nil)
				store.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
				enc.EXPECT().Encrypt(dek, gomock.Any()).
					Return([]byte("encrypted_meta"), []byte("nonce"), nil)
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
					ItemMeta: &model.ItemMeta{
						Title: "file1.txt",
						Type:  model.ItemTypeFile,
					},
					Data: json.RawMessage(`{}`),
				},
				file:        &mockMultipartFile{Reader: strings.NewReader("content")},
				contentType: "text/plain",
			},
			prepare: func(r *mocks.MockItemsRepository, ur *authMocks.MockUsersRepository, enc *mocks.MockFileEncryptor, store *mocks.MockObjectStorer, ctx context.Context, a args) {
				edek := &model.EncryptedDEK{
					EncryptedKey: []byte("encrypted_dek"),
					Nonce:        []byte("dek_nonce"),
				}
				dek := []byte("decrypted_dek")
				enc.EXPECT().NewFileNoncePrefix().
					Return([]byte("nonce_prefix"), nil)
				ur.EXPECT().GetUserDEK(ctx, a.userID).
					Return(edek, nil)
				enc.EXPECT().DecryptDEK(edek).
					Return(dek, nil)
				enc.EXPECT().
					EncryptFileStream(dek, gomock.Any(), gomock.Any(), fileChunkSize, gomock.Any(), gomock.Any()).
					Return(int64(100), nil)
				store.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
				enc.EXPECT().Encrypt(dek, gomock.Any()).
					Return([]byte("encrypted_meta"), []byte("nonce"), nil)
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
			encryptor := mocks.NewMockFileEncryptor(ctrl)
			store := mocks.NewMockObjectStorer(ctrl)
			ctx := context.Background()
			svc := NewCreateFileService(repo, userRepo, encryptor, store)

			tt.prepare(repo, userRepo, encryptor, store, ctx, tt.args)

			id, err := svc.CreateFileItem(
				ctx,
				tt.args.userID,
				tt.args.item,
				tt.args.file,
				tt.args.contentType,
			)

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
