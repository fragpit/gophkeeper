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

// CreateService stores new decrypted items for a user.
//
//go:generate mockgen -destination ./mocks/create_mock_gen.go . CreateService
type CreateService interface {
	CreateItem(
		ctx context.Context,
		userID int,
		item *model.ItemDecrypted,
	) (int, error)
}

type createRequest struct {
	Item *model.ItemDecrypted `json:"item"`
}

type createResponse struct {
	ID    int    `json:"id"`
	Error string `json:"error"`
}

// NewCreateHandler returns an Echo handler that creates a new item.
func NewCreateHandler(svc CreateService) echo.HandlerFunc {
	return func(c *echo.Context) error {
		claims, err := extractClaims[*auth.AccessClaims](c)
		if err != nil {
			return err
		}

		uid := claims.UserID
		reqData := &createRequest{}

		if err := c.Bind(reqData); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		// Валидация размеров item
		if err := ValidateItemSize(reqData.Item); err != nil {
			return c.JSON(
				http.StatusBadRequest,
				&createResponse{Error: err.Error()},
			)
		}

		id, err := svc.CreateItem(c.Request().Context(), uid, reqData.Item)
		if err != nil {
			slog.Error("create item", slog.Any("error", err))

			switch {
			case errors.Is(err, model.ErrAlreadyExists):
				return c.JSON(
					http.StatusConflict,
					&createResponse{Error: model.ErrAlreadyExists.Error()},
				)
			default:
				return c.JSON(
					http.StatusInternalServerError,
					&createResponse{
						Error: http.StatusText(http.StatusInternalServerError),
					},
				)
			}
		}

		return c.JSON(http.StatusOK, &createResponse{ID: id})
	}
}
