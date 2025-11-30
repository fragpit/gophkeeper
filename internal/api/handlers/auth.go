package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/fragpit/gophkeeper/internal/model"
	"github.com/labstack/echo/v5"
)

// AuthService provides user registration and authentication operations.
//
//go:generate mockgen -destination ./mocks/auth_mock_gen.go . AuthService
type AuthService interface {
	Register(ctx context.Context, login, password string) (string, error)
	Login(ctx context.Context, login, password string) (string, error)
}

type authRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string `json:"token"`
	Error string `json:"error"`
}

// NewAuthRegisterHandler handles user registration requests.
func NewAuthRegisterHandler(svc AuthService) echo.HandlerFunc {
	return func(c *echo.Context) error {
		var authReq authRequest
		if err := ValidateParseJSONRequest(c, &authReq); err != nil {
			slog.Error("validate json request", slog.Any("error", err))
			return err
		}

		token, err := svc.Register(
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
					&authResponse{Error: model.ErrPasswordPolicyViolated.Error()},
				)
			default:
				return c.JSON(
					http.StatusInternalServerError,
					&authResponse{Error: http.StatusText(http.StatusInternalServerError)},
				)
			}
		}

		return authJSONResponse(c, token)
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

		token, err := svc.Login(
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
					&authResponse{Error: http.StatusText(http.StatusInternalServerError)},
				)
			}
		}

		return authJSONResponse(c, token)
	}
}

func authJSONResponse(
	c *echo.Context,
	token string,
) error {
	authResp := &authResponse{
		Token: token,
	}

	c.Response().
		Header().
		Set("Authorization", fmt.Sprintf("Bearer %s", authResp.Token))

	return c.JSON(http.StatusOK, authResp)
}
