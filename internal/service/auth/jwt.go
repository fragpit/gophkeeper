package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims extends JWT registered claims with application specific fields.
type Claims struct {
	jwt.RegisteredClaims
	UserID int
}

// CreateJWTToken creates a signed JWT token for the given user ID with a TTL.
func CreateJWTToken(
	secret string,
	ttl time.Duration,
	userID int,
) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
		},
		UserID: userID,
	})

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// GetUserIDFromJWTToken parses a token and extracts the user ID claim.
func GetUserIDFromJWTToken(secret string, tokenString string) (int, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(t *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		},
	)
	if err != nil {
		return 0, errors.New("failed to parse token")
	}

	if !token.Valid {
		return 0, errors.New("invalid token")
	}

	return claims.UserID, nil
}
