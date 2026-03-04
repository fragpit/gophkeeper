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
	jwtSecret string
	jwtTTL    time.Duration

	userRepo   model.UsersRepository
	dekCreator DEKCreator
	tokenRepo  TokenRepository
}

// NewAuthService constructs a new AuthService instance.
func NewAuthService(
	jwtSecret string,
	jwtTTL time.Duration,

	cryptor DEKCreator,
	userRepo model.UsersRepository,
	tokenRepo TokenRepository,
) *AuthService {
	return &AuthService{
		jwtSecret: jwtSecret,
		jwtTTL:    jwtTTL,

		dekCreator: cryptor,
		userRepo:   userRepo,
		tokenRepo:  tokenRepo,
	}
}

// Register creates a new user and returns a signed JWT token.
func (a *AuthService) Register(
	ctx context.Context,
	login, password string,
) (*model.TokenPair, error) {
	if _, err := a.userRepo.GetByLogin(ctx, login); err == nil {
		return nil, model.ErrUserExists
	} else if !errors.Is(err, model.ErrUserNotFound) {
		return nil, fmt.Errorf("check user exists: %w", err)
	}

	if err := model.ValidatePassword(password); err != nil {
		return nil, model.ErrPasswordPolicyViolated
	}

	u := model.NewUser(login)
	passwordHash, err := hashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	u.PasswordHash = passwordHash

	dek, err := a.dekCreator.CreateDEK()
	if err != nil {
		return nil, fmt.Errorf("create user dek: %w", err)
	}

	u, err = a.userRepo.CreateWithDEK(ctx, u, dek)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	slog.Info("user created", slog.Int("user_id", u.ID))

	tokenPair, err := a.generateTokenPair(ctx, u.ID)
	if err != nil {
		return nil, fmt.Errorf("generate token pair: %w", err)
	}

	return tokenPair, nil
}

// Login authenticates a user by login and password and issues a JWT token.
func (a *AuthService) Login(
	ctx context.Context,
	login, password string,
) (*model.TokenPair, error) {
	u, err := a.userRepo.GetByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			return nil, model.ErrUserNotFound
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	if ok := comparePasswordHash(password, u.PasswordHash); !ok {
		return nil, model.ErrInvalidCredentials
	}

	tokenPair, err := a.generateTokenPair(ctx, u.ID)
	if err != nil {
		return nil, fmt.Errorf("generate token pair: %w", err)
	}

	return tokenPair, nil
}

func (a *AuthService) Refresh(
	ctx context.Context,
	userID int,
	refTokenID string,
) (*model.TokenPair, error) {
	tokenUserID, err := a.tokenRepo.UseRefreshToken(ctx, refTokenID)
	if err != nil {
		return nil, fmt.Errorf("use refresh token: %w", err)
	}

	if tokenUserID != userID {
		return nil, model.ErrTokenNotFound
	}

	userExists, err := a.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			return nil, model.ErrUserNotFound
		}
		return nil, fmt.Errorf("get user: %w", err)
	}
	if userExists == nil {
		return nil, model.ErrUserNotFound
	}

	tokenPair, err := a.generateTokenPair(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("generate token pair: %w", err)
	}

	if err := a.tokenRepo.DeleteRefreshToken(ctx, refTokenID); err != nil {
		slog.Warn("failed to delete used refresh token",
			slog.String("token_id", refTokenID),
			slog.String("error", err.Error()),
		)
	}

	return tokenPair, nil
}
