package router

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/fragpit/gophkeeper/internal/model"
	"github.com/fragpit/gophkeeper/internal/service/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

// UserVerifier checks if a user exists in the system.
type UserVerifier interface {
	GetByID(ctx context.Context, userID int) (*model.User, error)
}

// UserExistenceMiddleware creates middleware that verifies the authenticated user exists in the database.
// This middleware should be applied after JWT authentication middleware.
// It prevents access with valid but orphaned JWT tokens (e.g., after user deletion).
func UserExistenceMiddleware(verifier UserVerifier) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			// Extract JWT token from context (set by JWT middleware)
			v := c.Get("user")
			token, ok := v.(*jwt.Token)
			if !ok || token == nil {
				// No token in context - may be a public endpoint, let it pass
				return next(c)
			}

			// Extract claims
			claims, ok := token.Claims.(*auth.AccessClaims)
			if !ok {
				slog.Warn("failed to extract claims from token")
				return c.NoContent(http.StatusUnauthorized)
			}

			// Verify user exists in database
			_, err := verifier.GetByID(c.Request().Context(), claims.UserID)
			if err != nil {
				if errors.Is(err, model.ErrUserNotFound) {
					slog.Warn(
						"user not found in database",
						slog.Int("user_id", claims.UserID),
					)
					return c.NoContent(http.StatusUnauthorized)
				}
				slog.Error(
					"failed to verify user existence",
					slog.Int("user_id", claims.UserID),
					slog.Any("error", err),
				)
				return c.NoContent(http.StatusInternalServerError)
			}

			// User exists, proceed with request
			return next(c)
		}
	}
}
