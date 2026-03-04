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

// UpdateService updates existing items for a user.
//
//go:generate mockgen -destination ./mocks/update_mock_gen.go . UpdateService
type UpdateService interface {
	UpdateItem(
		ctx context.Context,
		userID int,
		title string,
		item *model.ItemDecrypted,
	) error
}

type updateRequest struct {
	Title string               `json:"title"`
	Item  *model.ItemDecrypted `json:"item"`
}

type updateResponse struct {
	Error string `json:"error,omitempty"`
}

// NewUpdateHandler returns an Echo handler that updates an existing item.
func NewUpdateHandler(svc UpdateService) echo.HandlerFunc {
	return func(c *echo.Context) error {
		claims, err := extractClaims[*auth.AccessClaims](c)
		if err != nil {
			return err
		}

		uid := claims.UserID
		reqData := &updateRequest{}

		if err := c.Bind(reqData); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		if reqData.Title == "" {
			return c.JSON(
				http.StatusBadRequest,
				&updateResponse{Error: "title is required"},
			)
		}

		// Валидация размеров item
		if err := ValidateItemSize(reqData.Item); err != nil {
			return c.JSON(
				http.StatusBadRequest,
				&updateResponse{Error: err.Error()},
			)
		}

		if err := svc.UpdateItem(
			c.Request().Context(),
			uid,
			reqData.Title,
			reqData.Item,
		); err != nil {
			slog.Error("update item", slog.Any("error", err))

			switch {
			case errors.Is(err, model.ErrAlreadyExists):
				return c.JSON(
					http.StatusConflict,
					&updateResponse{Error: http.StatusText(http.StatusConflict)},
				)
			default:
				return c.JSON(
					http.StatusInternalServerError,
					&updateResponse{
						Error: http.StatusText(http.StatusInternalServerError),
					},
				)
			}
		}

		return c.JSON(http.StatusOK, &updateResponse{})
	}
}
