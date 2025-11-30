package auth

import (
	"context"
	"errors"
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

func TestNewAuthService(t *testing.T) {
	t.Run("creates auth service", func(t *testing.T) {
		svc := NewAuthService(nil, nil, "secret", time.Hour)
		assert.NotNil(t, svc)
	})

	t.Run("creates auth service with empty secret", func(t *testing.T) {
		svc := NewAuthService(nil, nil, "", 0)
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
			ctx := context.Background()
			svc := NewAuthService(repo, dekCreator, "secret", time.Minute)

			tt.prepare(repo, dekCreator, ctx, tt.args)

			token, err := svc.Register(ctx, tt.args.login, tt.args.password)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			}

			if tt.wantToken {
				assert.NotEmpty(t, token)
				assert.NoError(t, err)
			}
			if !tt.wantToken && tt.wantErr == nil {
				assert.Empty(t, token)
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
	hashed, _ := HashPassword("pass")

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
			ctx := context.Background()
			svc := NewAuthService(repo, dekCreator, "secret", time.Minute)

			tt.prepare(repo, ctx, tt.args)

			token, err := svc.Login(ctx, tt.args.login, tt.args.password)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			}

			if tt.wantToken {
				assert.NotEmpty(t, token)
				assert.NoError(t, err)
			}
			if !tt.wantToken {
				assert.Empty(t, token)
			}
		})
	}
}
