package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateJWTToken(t *testing.T) {
	tests := []struct {
		name    string
		secret  string
		dur     time.Duration
		id      int
		wantErr bool
	}{
		{
			name:    "success",
			secret:  "secret",
			dur:     1 * time.Second,
			id:      0,
			wantErr: false,
		},
		{
			name:    "success with different user id",
			secret:  "secret",
			dur:     1 * time.Hour,
			id:      123,
			wantErr: false,
		},
		{
			name:    "success with empty secret",
			secret:  "",
			dur:     1 * time.Minute,
			id:      42,
			wantErr: false,
		},
		{
			name:    "success with zero duration",
			secret:  "test",
			dur:     0,
			id:      1,
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			token, err := CreateJWTToken(tc.secret, tc.dur, tc.id)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, token)
			}

		})
	}
}

func TestGetUserIDFromJWTToken(t *testing.T) {
	validSecret := "secret"
	validToken, err := CreateJWTToken(validSecret, 1*time.Hour, 0)
	require.NoError(t, err)
	expiredToken, err := CreateJWTToken(validSecret, 0, 0)
	require.NoError(t, err)

	// Create token with wrong algorithm (HS384 instead of HS256)
	wrongAlgToken := jwt.NewWithClaims(jwt.SigningMethodHS384, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
		UserID: 42,
	})
	wrongAlgTokenString, err := wrongAlgToken.SignedString([]byte(validSecret))
	require.NoError(t, err)

	// Create token with HS512 algorithm
	hs512Token := jwt.NewWithClaims(jwt.SigningMethodHS512, Claims{
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
			id, err := GetUserIDFromJWTToken(tc.secret, tc.token)
			if tc.wantErr {
				assert.Error(t, err)
				if tc.errMsg != "" {
					require.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantID, id)
			}
		})
	}
}
