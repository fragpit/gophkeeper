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

// GetService fetches a single decrypted item for a user.
//
//go:generate mockgen -destination ./mocks/get_mock_gen.go . GetService
type GetService interface {
	GetItemByTitle(
		ctx context.Context,
		userID int,
		title string,
	) (*model.ItemDecrypted, error)
}

type getResponse struct {
	Item  *model.ItemDecrypted `json:"item"`
	Error string               `json:"error"`
}

// NewGetHandler returns an Echo handler that fetches an item by title.
func NewGetHandler(svc GetService) echo.HandlerFunc {
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

		item, err := svc.GetItemByTitle(
			c.Request().Context(),
			uid,
			title,
		)
		if err != nil {
			slog.Error("get item", slog.Any("error", err))

			switch {
			case errors.Is(err, model.ErrItemNotFound):
				return c.JSON(
					http.StatusNotFound,
					&getResponse{Error: http.StatusText(http.StatusNotFound)},
				)
			default:
				return c.JSON(
					http.StatusInternalServerError,
					&getResponse{Error: http.StatusText(http.StatusInternalServerError)},
				)
			}
		}

		return c.JSON(http.StatusOK, &getResponse{Item: item})
	}
}
