package auth

import (
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
			return []byte(secret), nil
		},
	)
	if err != nil {
		return 0, fmt.Errorf("parse token: %w", err)
	}

	return claims.UserID, nil
}
