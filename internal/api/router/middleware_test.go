package router

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fragpit/gophkeeper/internal/model"
	"github.com/fragpit/gophkeeper/internal/service/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockUserVerifier struct {
	users map[int]*model.User
	err   error
}

func (m *mockUserVerifier) GetByID(
	_ context.Context,
	userID int,
) (*model.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	if u, ok := m.users[userID]; ok {
		return u, nil
	}
	return nil, model.ErrUserNotFound
}

func TestUserExistenceMiddleware(t *testing.T) {
	jwtSecret := "test-secret"

	tests := []struct {
		name           string
		userID         int
		mockUsers      map[int]*model.User
		mockErr        error
		expectedStatus int
		expectedCalled bool
	}{
		{
			name:   "user exists - should pass",
			userID: 1,
			mockUsers: map[int]*model.User{
				1: {ID: 1, Login: "testuser"},
			},
			expectedStatus: http.StatusOK,
			expectedCalled: true,
		},
		{
			name:           "user not found - should return 401",
			userID:         999,
			mockUsers:      map[int]*model.User{},
			expectedStatus: http.StatusUnauthorized,
			expectedCalled: false,
		},
		{
			name:   "database error - should return 500",
			userID: 1,
			mockUsers: map[int]*model.User{
				1: {ID: 1, Login: "testuser"},
			},
			mockErr:        errors.New("database error"),
			expectedStatus: http.StatusInternalServerError,
			expectedCalled: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock verifier
			verifier := &mockUserVerifier{
				users: tc.mockUsers,
				err:   tc.mockErr,
			}

			// Create test echo instance
			e := echo.New()

			// Create JWT token
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, &auth.Claims{
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
				},
				UserID: tc.userID,
			})
			tokenString, err := token.SignedString([]byte(jwtSecret))
			require.NoError(t, err)

			// Create test request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", "Bearer "+tokenString)
			rec := httptest.NewRecorder()

			// Setup context with JWT token (simulating JWT middleware)
			c := e.NewContext(req, rec)
			c.Set("user", token)

			// Track if next handler was called
			nextCalled := false
			nextHandler := func(c *echo.Context) error {
				nextCalled = true
				return c.String(http.StatusOK, "success")
			}

			// Apply middleware
			middleware := UserExistenceMiddleware(verifier)
			handler := middleware(nextHandler)

			// Execute
			_ = handler(c)

			// Assert
			assert.Equal(t, tc.expectedStatus, rec.Code)
			assert.Equal(t, tc.expectedCalled, nextCalled)
		})
	}
}

func TestUserExistenceMiddleware_NoToken(t *testing.T) {
	verifier := &mockUserVerifier{
		users: map[int]*model.User{},
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// No token set in context (public endpoint)
	nextCalled := false
	nextHandler := func(c *echo.Context) error {
		nextCalled = true
		return c.String(http.StatusOK, "success")
	}

	middleware := UserExistenceMiddleware(verifier)
	handler := middleware(nextHandler)

	_ = handler(c)

	// Should pass through for public endpoints
	assert.True(t, nextCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
}
