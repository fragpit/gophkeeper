package auth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/fragpit/gophkeeper/internal/model"
	mocks "github.com/fragpit/gophkeeper/internal/service/auth/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type mockDEKCreator struct {
	shouldFail bool
}

func (m *mockDEKCreator) CreateDEK() (*model.EncryptedDEK, error) {
	if m.shouldFail {
		return nil, errors.New("dek creation failed")
	}
	return &model.EncryptedDEK{
		EncryptedKey: []byte("encrypted_key"),
		Nonce:        []byte("nonce"),
	}, nil
}

type mockTokenRepo struct {
	storeFunc       func(ctx context.Context, data *RefreshTokenData) error
	existsFunc      func(ctx context.Context, tokenID string) (bool, error)
	useFunc         func(ctx context.Context, tokenID string) (int, error)
	deleteFunc      func(ctx context.Context, tokenID string) error
	storeCalls      int
	deleteCalls     int
	useCalls        int
	lastStoredToken *RefreshTokenData
	lastDeleted     string
	lastUsedToken   string
}

func (m *mockTokenRepo) StoreRefreshToken(
	ctx context.Context,
	data *RefreshTokenData,
) error {
	m.storeCalls++
	m.lastStoredToken = data
	if m.storeFunc == nil {
		return nil
	}
	return m.storeFunc(ctx, data)
}

func (m *mockTokenRepo) UseRefreshToken(
	ctx context.Context,
	tokenID string,
) (int, error) {
	m.useCalls++
	m.lastUsedToken = tokenID
	if m.useFunc == nil {
		return 0, fmt.Errorf("token not found")
	}
	return m.useFunc(ctx, tokenID)
}

func (m *mockTokenRepo) RefreshTokenExists(
	ctx context.Context,
	tokenID string,
) (bool, error) {
	if m.existsFunc == nil {
		return false, nil
	}
	return m.existsFunc(ctx, tokenID)
}

func (m *mockTokenRepo) DeleteRefreshToken(
	ctx context.Context,
	tokenID string,
) error {
	m.deleteCalls++
	m.lastDeleted = tokenID
	if m.deleteFunc == nil {
		return nil
	}
	return m.deleteFunc(ctx, tokenID)
}

func TestNewAuthService(t *testing.T) {
	t.Run("creates auth service", func(t *testing.T) {
		svc := NewAuthService("secret", time.Hour, nil, nil, nil)
		assert.NotNil(t, svc)
	})

	t.Run("creates auth service with empty secret", func(t *testing.T) {
		svc := NewAuthService("", 0, nil, nil, nil)
		assert.NotNil(t, svc)
	})
}

func TestAuthService_Register(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))

	type args struct {
		login    string
		password string
	}
	tests := []struct {
		name      string
		args      args
		prepare   func(*mocks.MockUsersRepository, *mockDEKCreator, context.Context, args)
		wantErr   error
		wantToken bool
	}{
		{
			name: "user already exists",
			args: args{"user", "validpassword12345"},
			prepare: func(r *mocks.MockUsersRepository, _ *mockDEKCreator, ctx context.Context, a args) {
				r.EXPECT().GetByLogin(ctx, a.login).Return(&model.User{ID: 1}, nil)
			},
			wantErr:   model.ErrUserExists,
			wantToken: false,
		},
		{
			name: "invalid password policy",
			args: args{"user", "1"},
			prepare: func(r *mocks.MockUsersRepository, _ *mockDEKCreator, ctx context.Context, a args) {
				r.EXPECT().GetByLogin(ctx, a.login).
					Return(nil, model.ErrUserNotFound)
			},
			wantErr:   model.ErrPasswordPolicyViolated,
			wantToken: false,
		},
		{
			name: "create user and token",
			args: args{"user", "valid_password12"},
			prepare: func(r *mocks.MockUsersRepository, _ *mockDEKCreator, ctx context.Context, a args) {
				r.EXPECT().GetByLogin(ctx, a.login).
					Return(nil, model.ErrUserNotFound)
				r.EXPECT().CreateWithDEK(ctx, gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, u *model.User, dek *model.EncryptedDEK) (*model.User, error) {
						u.ID = 42
						return u, nil
					})
			},
			wantErr:   nil,
			wantToken: true,
		},
		{
			name: "dek creation fails",
			args: args{"user", "valid_password12"},
			prepare: func(r *mocks.MockUsersRepository, d *mockDEKCreator, ctx context.Context, a args) {
				r.EXPECT().GetByLogin(ctx, a.login).
					Return(nil, model.ErrUserNotFound)
				d.shouldFail = true
			},
			wantErr:   nil,
			wantToken: false,
		},
		{
			name: "repository create fails",
			args: args{"user", "valid_password12"},
			prepare: func(r *mocks.MockUsersRepository, _ *mockDEKCreator, ctx context.Context, a args) {
				r.EXPECT().GetByLogin(ctx, a.login).
					Return(nil, model.ErrUserNotFound)
				r.EXPECT().CreateWithDEK(ctx, gomock.Any(), gomock.Any()).
					Return(nil, errors.New("db error"))
			},
			wantErr:   nil,
			wantToken: false,
		},
		{
			name: "database connectivity error on check",
			args: args{"user", "valid_password12"},
			prepare: func(r *mocks.MockUsersRepository, _ *mockDEKCreator, ctx context.Context, a args) {
				r.EXPECT().GetByLogin(ctx, a.login).
					Return(nil, errors.New("connection refused"))
			},
			wantErr:   nil,
			wantToken: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repo := mocks.NewMockUsersRepository(ctrl)
			dekCreator := &mockDEKCreator{}
			tokenRepo := &mockTokenRepo{}
			ctx := context.Background()
			svc := NewAuthService("secret", time.Minute, dekCreator, repo, tokenRepo)

			tt.prepare(repo, dekCreator, ctx, tt.args)

			token, err := svc.Register(ctx, tt.args.login, tt.args.password)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			}

			if tt.wantToken {
				require.NotNil(t, token)
				assert.NotEmpty(t, token.AccessToken)
				assert.NotEmpty(t, token.RefreshToken)
				assert.NoError(t, err)
			}
			if !tt.wantToken && tt.wantErr == nil {
				assert.Nil(t, token)
				assert.Error(t, err)
			}
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))

	type args struct {
		login    string
		password string
	}
	hashed, _ := hashPassword("pass")

	tests := []struct {
		name      string
		args      args
		prepare   func(*mocks.MockUsersRepository, context.Context, args)
		wantErr   error
		wantToken bool
	}{
		{
			name: "user not found",
			args: args{"user", "pass"},
			prepare: func(r *mocks.MockUsersRepository, ctx context.Context, a args) {
				r.EXPECT().GetByLogin(ctx, a.login).
					Return(nil, model.ErrUserNotFound)
			},
			wantErr:   model.ErrUserNotFound,
			wantToken: false,
		},
		{
			name: "invalid password",
			args: args{"user", "invalid_pass"},
			prepare: func(r *mocks.MockUsersRepository, ctx context.Context, a args) {
				r.EXPECT().GetByLogin(ctx, a.login).
					Return(&model.User{ID: 10, Login: a.login, PasswordHash: hashed}, nil)
			},
			wantErr:   model.ErrInvalidCredentials,
			wantToken: false,
		},
		{
			name: "success",
			args: args{"user", "pass"},
			prepare: func(r *mocks.MockUsersRepository, ctx context.Context, a args) {
				r.EXPECT().GetByLogin(ctx, a.login).
					Return(&model.User{ID: 7, Login: a.login, PasswordHash: hashed}, nil)
			},
			wantErr:   nil,
			wantToken: true,
		},
		{
			name: "database error",
			args: args{"user", "pass"},
			prepare: func(r *mocks.MockUsersRepository, ctx context.Context, a args) {
				r.EXPECT().GetByLogin(ctx, a.login).
					Return(nil, errors.New("connection timeout"))
			},
			wantErr:   nil,
			wantToken: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repo := mocks.NewMockUsersRepository(ctrl)
			dekCreator := &mockDEKCreator{}
			tokenRepo := &mockTokenRepo{}
			ctx := context.Background()
			svc := NewAuthService("secret", time.Minute, dekCreator, repo, tokenRepo)

			tt.prepare(repo, ctx, tt.args)

			token, err := svc.Login(ctx, tt.args.login, tt.args.password)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			}

			if tt.wantToken {
				require.NotNil(t, token)
				assert.NotEmpty(t, token.AccessToken)
				assert.NotEmpty(t, token.RefreshToken)
				assert.NoError(t, err)
			}
			if !tt.wantToken {
				assert.Nil(t, token)
			}
		})
	}
}

func TestAuthService_Refresh(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))

	tests := []struct {
		name        string
		userID      int
		tokenID     string
		setup       func(*mocks.MockUsersRepository, *mockTokenRepo)
		wantErr     error
		wantToken   bool
		errContains string
	}{
		{
			name:    "token not found",
			userID:  10,
			tokenID: "missing",
			setup: func(_ *mocks.MockUsersRepository, tr *mockTokenRepo) {
				tr.useFunc = func(ctx context.Context, tokenID string) (int, error) {
					return 0, fmt.Errorf("token not found")
				}
			},
			errContains: "use refresh token",
		},
		{
			name:    "token already used (race condition)",
			userID:  10,
			tokenID: "token-used",
			setup: func(_ *mocks.MockUsersRepository, tr *mockTokenRepo) {
				tr.useFunc = func(ctx context.Context, tokenID string) (int, error) {
					return 0, fmt.Errorf("token already used")
				}
			},
			errContains: "use refresh token",
		},
		{
			name:    "user ID mismatch",
			userID:  10,
			tokenID: "token-1",
			setup: func(_ *mocks.MockUsersRepository, tr *mockTokenRepo) {
				tr.useFunc = func(ctx context.Context, tokenID string) (int, error) {
					return 99, nil // Different user ID
				}
			},
			wantErr: model.ErrTokenNotFound,
		},
		{
			name:    "user not found",
			userID:  10,
			tokenID: "token-2",
			setup: func(r *mocks.MockUsersRepository, tr *mockTokenRepo) {
				tr.useFunc = func(ctx context.Context, tokenID string) (int, error) {
					return 10, nil
				}
				r.EXPECT().GetByID(gomock.Any(), 10).Return(nil, model.ErrUserNotFound)
			},
			wantErr: model.ErrUserNotFound,
		},
		{
			name:    "user nil",
			userID:  10,
			tokenID: "token-3",
			setup: func(r *mocks.MockUsersRepository, tr *mockTokenRepo) {
				tr.useFunc = func(ctx context.Context, tokenID string) (int, error) {
					return 10, nil
				}
				r.EXPECT().GetByID(gomock.Any(), 10).Return(nil, nil)
			},
			wantErr: model.ErrUserNotFound,
		},
		{
			name:    "success",
			userID:  7,
			tokenID: "token-4",
			setup: func(r *mocks.MockUsersRepository, tr *mockTokenRepo) {
				tr.useFunc = func(ctx context.Context, tokenID string) (int, error) {
					return 7, nil
				}
				r.EXPECT().GetByID(gomock.Any(), 7).Return(&model.User{ID: 7}, nil)
			},
			wantToken: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repo := mocks.NewMockUsersRepository(ctrl)
			tokenRepo := &mockTokenRepo{}
			svc := NewAuthService(
				"secret",
				time.Minute,
				&mockDEKCreator{},
				repo,
				tokenRepo,
			)

			if tt.setup != nil {
				tt.setup(repo, tokenRepo)
			}

			got, err := svc.Refresh(context.Background(), tt.userID, tt.tokenID)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else if tt.errContains != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			}

			if tt.wantToken {
				require.NoError(t, err)
				require.NotNil(t, got)
				assert.NotEmpty(t, got.AccessToken)
				assert.NotEmpty(t, got.RefreshToken)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}
