package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/fragpit/gophkeeper/internal/model"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	accessTokenTTL = 5 * time.Second
	// accessTokenTTL = 5 * time.Minute
)

type TokenRepository interface {
	StoreRefreshToken(ctx context.Context, data *RefreshTokenData) error
	// UseRefreshToken atomically marks token as used and returns userID.
	// Returns ErrTokenNotFound if token doesn't exist, expired, or already used.
	UseRefreshToken(ctx context.Context, tokenID string) (userID int, err error)
	DeleteRefreshToken(ctx context.Context, tokenID string) error
}

// AccessClaims extends JWT registered claims with application specific fields.
type AccessClaims struct {
	UserID int

	jwt.RegisteredClaims
}

// RefreshClaims extends JWT registered claims with application specific fields.
type RefreshClaims struct {
	UserID  int
	TokenID string

	jwt.RegisteredClaims
}

type RefreshTokenData struct {
	UserID    int
	TokenID   string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// GenerateTokenPair creates a signed JWT token for the given user ID with a TTL.
func (a *AuthService) generateTokenPair(
	ctx context.Context,
	userID int,
) (*model.TokenPair, error) {
	accessToken, err := generateAccessToken(a.jwtSecret, userID)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, tokenID, expiresAt, err := generateRefreshToken(
		a.jwtSecret,
		a.jwtTTL,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	err = a.tokenRepo.StoreRefreshToken(ctx, &RefreshTokenData{
		UserID:    userID,
		TokenID:   tokenID,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &model.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func generateAccessToken(
	secret string,
	userID int,
) (string, error) {
	claims := AccessClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(accessTokenTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func generateRefreshToken(
	secret string,
	ttl time.Duration,
	userID int,
) (string, string, time.Time, error) {
	tokenID := uuid.NewString()
	claims := RefreshClaims{
		UserID:  userID,
		TokenID: tokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(secret))
	return signedToken, claims.TokenID, claims.ExpiresAt.Time, err
}

// GetUserIDFromJWTToken parses a token and extracts the user ID claim.
func (a *AuthService) GetUserIDFromJWTToken(tokenString string) (int, error) {
	claims := &AccessClaims{}
	_, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, fmt.Errorf(
					"unexpected signing algorithm: %s",
					t.Method.Alg(),
				)
			}
			return []byte(a.jwtSecret), nil
		},
	)
	if err != nil {
		return 0, fmt.Errorf("parse token: %w", err)
	}

	return claims.UserID, nil
}
