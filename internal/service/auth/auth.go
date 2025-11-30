package auth

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/fragpit/gophkeeper/internal/model"
)

type DEKCreator interface {
	CreateDEK() (*model.EncryptedDEK, error)
}

type AuthService struct {
	repo       model.UsersRepository
	dekCreator DEKCreator

	jwtSecret string
	jwtTTL    time.Duration
}

func NewAuthService(
	repo model.UsersRepository,
	cryptor DEKCreator,

	jwtSecret string,
	jwtTTL time.Duration,
) *AuthService {
	return &AuthService{
		repo:       repo,
		dekCreator: cryptor,

		jwtSecret: jwtSecret,
		jwtTTL:    jwtTTL,
	}
}

func (a *AuthService) Register(
	ctx context.Context,
	login, password string,
) (string, error) {
	if _, err := a.repo.GetByLogin(ctx, login); err == nil {
		return "", model.ErrUserExists
	}

	if err := model.ValidatePassword(password); err != nil {
		return "", model.ErrPasswordPolicyViolated
	}

	u := model.NewUser(login)
	passwordHash, err := HashPassword(password)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	u.PasswordHash = passwordHash

	dek, err := a.dekCreator.CreateDEK()
	if err != nil {
		return "", fmt.Errorf("create user dek: %w", err)
	}

	u, err = a.repo.CreateWithDEK(ctx, u, dek)
	if err != nil {
		return "", fmt.Errorf("failed to create user: %w", err)
	}

	slog.Info("user created", slog.Int("user_id", u.ID))

	token, err := CreateJWTToken(a.jwtSecret, a.jwtTTL, u.ID)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	return token, nil
}

func (a *AuthService) Login(
	ctx context.Context,
	login, password string,
) (string, error) {
	u, err := a.repo.GetByLogin(ctx, login)
	if err != nil {
		return "", model.ErrUserNotFound
	}

	if ok := ComparePasswordHash(password, u.PasswordHash); !ok {
		return "", model.ErrInvalidCredentials
	}

	token, err := CreateJWTToken(a.jwtSecret, a.jwtTTL, u.ID)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return token, nil
}
