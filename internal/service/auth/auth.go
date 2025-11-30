package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/fragpit/gophkeeper/internal/model"
)

// DEKCreator produces encrypted data encryption keys for users.
type DEKCreator interface {
	CreateDEK() (*model.EncryptedDEK, error)
}

// AuthService manages user registration and authentication logic.
type AuthService struct {
	repo       model.UsersRepository
	dekCreator DEKCreator

	jwtSecret string
	jwtTTL    time.Duration
}

// NewAuthService constructs a new AuthService instance.
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

// Register creates a new user and returns a signed JWT token.
func (a *AuthService) Register(
	ctx context.Context,
	login, password string,
) (string, error) {
	if _, err := a.repo.GetByLogin(ctx, login); err == nil {
		return "", model.ErrUserExists
	} else if !errors.Is(err, model.ErrUserNotFound) {
		return "", fmt.Errorf("check user exists: %w", err)
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

// Login authenticates a user by login and password and issues a JWT token.
func (a *AuthService) Login(
	ctx context.Context,
	login, password string,
) (string, error) {
	u, err := a.repo.GetByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			return "", model.ErrUserNotFound
		}
		return "", fmt.Errorf("get user: %w", err)
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
