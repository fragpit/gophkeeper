package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/fragpit/gophkeeper/internal/model"
	"github.com/fragpit/gophkeeper/internal/service/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

// ItemsService describes listing operations for user items.
//
//go:generate mockgen -destination ./mocks/items_mock_gen.go . ItemsService
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

// NewListHandler returns an Echo handler that lists items for the authenticated user.
func NewListHandler(svc ItemsService) echo.HandlerFunc {
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
		iType := model.ItemType(c.QueryParam("type"))

		// Empty type means "all types", otherwise validate
		if iType != "" && !iType.Valid() {
			return c.JSON(
				http.StatusBadRequest,
				&listResponse{Error: "invalid item type"},
			)
		}

		items, err := svc.List(
			c.Request().Context(),
			uid,
			iType,
		)
		if err != nil {
			slog.Error("list items", slog.Any("error", err))
			return c.JSON(
				http.StatusInternalServerError,
				&listResponse{Error: http.StatusText(http.StatusInternalServerError)},
			)
		}

		return c.JSON(http.StatusOK, &listResponse{Items: items})
	}
}
