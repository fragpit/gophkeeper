package handlers

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

//go:generate mockgen -destination ./mocks/delete_mock_gen.go . DeleteService
type DeleteService interface {
	DeleteItem(ctx context.Context, userID int, title string) error
}

type deleteResponse struct {
	Error string `json:"error"`
}

func NewDeleteHandler(svc DeleteService) echo.HandlerFunc {
	return func(c *echo.Context) error {
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
		title := c.QueryParam("title")

		if title == "" {
			return c.JSON(
				http.StatusBadRequest,
				&deleteResponse{Error: "title parameter is required"},
			)
		}

		err := svc.DeleteItem(c.Request().Context(), uid, title)
		if err != nil {
			slog.Error("delete item", slog.Any("error", err))

			switch {
			case errors.Is(err, model.ErrItemNotFound):
				return c.JSON(
					http.StatusNotFound,
					&deleteResponse{Error: "item not found"},
				)
			default:
				return c.JSON(
					http.StatusInternalServerError,
					&deleteResponse{
						Error: http.StatusText(http.StatusInternalServerError),
					},
				)
			}
		}

		return c.NoContent(http.StatusNoContent)
	}
}
