package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/fragpit/gophkeeper/internal/model"
	"github.com/fragpit/gophkeeper/internal/service/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

//go:generate mockgen -destination ./mocks/auth_mock_gen.go . AuthService
type ItemsService interface {
	List(
		ctx context.Context,
		userID int,
		iType model.ItemType,
	) ([]model.ItemMeta, error)
}

type listResponse struct {
	Items []model.ItemMeta `json:"items"`
	Error string           `json:"error"`
}

func NewListHandler(svc ItemsService) echo.HandlerFunc {
	return func(c echo.Context) error {
		v := c.Get("user")
		token, ok := v.(*jwt.Token)
		if !ok || token == nil {
			return c.NoContent(http.StatusUnauthorized)
		}

		claims, ok := token.Claims.(*auth.Claims)
		if !ok {
			return c.NoContent(http.StatusUnauthorized)
		}

		uid := claims.UserID
		iType := c.QueryParam("type")

		items, err := svc.List(
			c.Request().Context(),
			uid,
			model.ItemType(iType),
		)
		if err != nil {
			slog.Error("list items", slog.Any("error", err))

			switch {
			case errors.Is(err, model.ErrUserExists):
				return c.JSON(
					http.StatusConflict,
					&listResponse{Error: http.StatusText(http.StatusConflict)},
				)
			case errors.Is(err, model.ErrPasswordPolicyViolated):
				return c.JSON(
					http.StatusBadRequest,
					&listResponse{Error: model.ErrPasswordPolicyViolated.Error()},
				)
			default:
				return c.JSON(
					http.StatusInternalServerError,
					&listResponse{Error: http.StatusText(http.StatusInternalServerError)},
				)
			}
		}

		return c.JSON(http.StatusOK, &listResponse{Items: items})
	}
}
