package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/fragpit/gophkeeper/internal/model"
	"github.com/fragpit/gophkeeper/internal/service/auth"
	"github.com/labstack/echo/v5"
)

// AuthService provides user registration and authentication operations.
//
//go:generate mockgen -destination ./mocks/auth_mock_gen.go . AuthService
type AuthService interface {
	// Register returns user token
	Register(
		ctx context.Context,
		login, password string,
	) (*model.TokenPair, error)
	// Login returns user token
	Login(ctx context.Context, login, password string) (*model.TokenPair, error)
	Refresh(
		ctx context.Context,
		userID int,
		refreshToken string,
	) (*model.TokenPair, error)
}

type authRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type authResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Error        string `json:"error"`
}

type authRefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Error        string `json:"error"`
}

// NewAuthRegisterHandler handles user registration requests.
func NewAuthRegisterHandler(svc AuthService) echo.HandlerFunc {
	return func(c *echo.Context) error {
		var authReq authRequest
		if err := ValidateParseJSONRequest(c, &authReq); err != nil {
			slog.Error("validate json request", slog.Any("error", err))
			return err
		}

		tokens, err := svc.Register(
			c.Request().Context(),
			authReq.Login,
			authReq.Password,
		)
		if err != nil {
			slog.Error(
				"register user",
				slog.String("user", authReq.Login),
				slog.Any("error", err),
			)
			switch {
			case errors.Is(err, model.ErrUserExists):
				return c.JSON(
					http.StatusConflict,
					&authResponse{Error: http.StatusText(http.StatusConflict)},
				)
			case errors.Is(err, model.ErrPasswordPolicyViolated):
				return c.JSON(
					http.StatusBadRequest,
					&authResponse{
						Error: model.ErrPasswordPolicyViolated.Error(),
					},
				)
			default:
				return c.JSON(
					http.StatusInternalServerError,
					&authResponse{
						Error: http.StatusText(http.StatusInternalServerError),
					},
				)
			}
		}

		return c.JSON(http.StatusOK, &authResponse{
			AccessToken:  tokens.AccessToken,
			RefreshToken: tokens.RefreshToken,
		})
	}
}

// NewAuthLoginHandler handles user login requests.
func NewAuthLoginHandler(svc AuthService) echo.HandlerFunc {
	return func(c *echo.Context) error {
		var authReq authRequest
		if err := ValidateParseJSONRequest(c, &authReq); err != nil {
			slog.Error("validate json request", slog.Any("error", err))
			return err
		}

		tokens, err := svc.Login(
			c.Request().Context(),
			authReq.Login,
			authReq.Password,
		)
		if err != nil {
			slog.Error(
				"login user",
				slog.String("user", authReq.Login),
				slog.Any("error", err),
			)
			switch {
			case errors.Is(err, model.ErrUserNotFound):
				return c.JSON(
					http.StatusNotFound,
					&authResponse{Error: model.ErrUserNotFound.Error()},
				)
			case errors.Is(err, model.ErrInvalidCredentials):
				return c.JSON(
					http.StatusUnauthorized,
					&authResponse{Error: model.ErrInvalidCredentials.Error()},
				)
			default:
				return c.JSON(
					http.StatusInternalServerError,
					&authResponse{
						Error: http.StatusText(http.StatusInternalServerError),
					},
				)
			}
		}

		return c.JSON(http.StatusOK, &authResponse{
			AccessToken:  tokens.AccessToken,
			RefreshToken: tokens.RefreshToken,
		})
	}
}

func NewAuthRefreshHandler(svc AuthService) echo.HandlerFunc {
	return func(c *echo.Context) error {
		claims, err := extractClaims[*auth.RefreshClaims](c)
		if err != nil {
			return c.JSON(
				http.StatusBadRequest,
				&authRefreshResponse{Error: "invalid refresh token claims"},
			)
		}

		if claims == nil || claims.TokenID == "" || claims.UserID == 0 {
			return c.JSON(
				http.StatusBadRequest,
				&authRefreshResponse{Error: "missing token ID or user ID in claims"},
			)
		}

		tokens, err := svc.Refresh(
			c.Request().Context(),
			claims.UserID,
			claims.TokenID,
		)
		if err != nil {
			slog.Error("refresh token", slog.Any("error", err))
			switch {
			case errors.Is(err, model.ErrTokenNotFound):
				return c.JSON(
					http.StatusUnauthorized,
					&authRefreshResponse{Error: "refresh token not found or expired"},
				)
			case errors.Is(err, model.ErrUserNotFound):
				return c.JSON(
					http.StatusUnauthorized,
					&authRefreshResponse{Error: model.ErrUserNotFound.Error()},
				)
			default:
				return c.JSON(
					http.StatusInternalServerError,
					&authRefreshResponse{
						Error: http.StatusText(http.StatusInternalServerError),
					},
				)
			}
		}

		return c.JSON(http.StatusOK, &authRefreshResponse{
			AccessToken:  tokens.AccessToken,
			RefreshToken: tokens.RefreshToken,
		})
	}
}
