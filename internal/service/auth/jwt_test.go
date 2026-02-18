package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAccessToken(t *testing.T) {
	tests := []struct {
		name    string
		secret  string
		id      int
		wantErr bool
	}{
		{
			name:    "success",
			secret:  "secret",
			id:      0,
			wantErr: false,
		},
		{
			name:    "success with different user id",
			secret:  "secret",
			id:      123,
			wantErr: false,
		},
		{
			name:    "success with empty secret",
			secret:  "",
			id:      42,
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			token, err := generateAccessToken(tc.secret, tc.id)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotEmpty(t, token)

			claims := &AccessClaims{}
			parsed, err := jwt.ParseWithClaims(
				token,
				claims,
				func(t *jwt.Token) (interface{}, error) {
					return []byte(tc.secret), nil
				},
			)
			require.NoError(t, err)
			require.True(t, parsed.Valid)
			assert.Equal(t, tc.id, claims.UserID)
		})
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	token, tokenID, expiresAt, err := generateRefreshToken(
		"secret",
		10*time.Minute,
		5,
	)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.NotEmpty(t, tokenID)
	require.True(t, expiresAt.After(time.Now()))

	claims := &RefreshClaims{}
	parsed, err := jwt.ParseWithClaims(
		token,
		claims,
		func(t *jwt.Token) (interface{}, error) {
			return []byte("secret"), nil
		},
	)
	require.NoError(t, err)
	require.True(t, parsed.Valid)
	require.Equal(t, 5, claims.UserID)
	require.Equal(t, tokenID, claims.TokenID)
}

func TestGetUserIDFromJWTToken(t *testing.T) {
	validSecret := "secret"
	validToken, err := generateAccessToken(validSecret, 0)
	require.NoError(t, err)

	expiredClaims := AccessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
		},
		UserID: 0,
	}
	expiredTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	expiredToken, err := expiredTokenObj.SignedString([]byte(validSecret))
	require.NoError(t, err)

	// Create token with wrong algorithm (HS384 instead of HS256)
	wrongAlgToken := jwt.NewWithClaims(jwt.SigningMethodHS384, AccessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
		UserID: 42,
	})
	wrongAlgTokenString, err := wrongAlgToken.SignedString([]byte(validSecret))
	require.NoError(t, err)

	// Create token with HS512 algorithm
	hs512Token := jwt.NewWithClaims(jwt.SigningMethodHS512, AccessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
		UserID: 99,
	})
	hs512TokenString, err := hs512Token.SignedString([]byte(validSecret))
	require.NoError(t, err)

	tests := []struct {
		name    string
		secret  string
		token   string
		wantID  int
		wantErr bool
		errMsg  string
	}{
		{
			name:    "success",
			secret:  validSecret,
			token:   validToken,
			wantID:  0,
			wantErr: false,
		},
		{
			name:    "fail invalid secret",
			secret:  "invalid",
			token:   validToken,
			wantErr: true,
			errMsg:  "signature is invalid",
		},
		{
			name:    "fail expired token",
			secret:  validSecret,
			token:   expiredToken,
			wantErr: true,
			errMsg:  "token is expired",
		},
		{
			name:    "fail invalid token format",
			secret:  validSecret,
			token:   "invalid.token.format",
			wantErr: true,
		},
		{
			name:    "fail empty token",
			secret:  validSecret,
			token:   "",
			wantErr: true,
		},
		{
			name:    "fail wrong algorithm HS384",
			secret:  validSecret,
			token:   wrongAlgTokenString,
			wantErr: true,
			errMsg:  "unexpected signing algorithm: HS384",
		},
		{
			name:    "fail wrong algorithm HS512",
			secret:  validSecret,
			token:   hs512TokenString,
			wantErr: true,
			errMsg:  "unexpected signing algorithm: HS512",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &AuthService{jwtSecret: tc.secret}
			id, err := svc.GetUserIDFromJWTToken(tc.token)
			if tc.wantErr {
				assert.Error(t, err)
				if tc.errMsg != "" {
					require.Contains(t, err.Error(), tc.errMsg)
				}
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.wantID, id)
		})
	}
}
