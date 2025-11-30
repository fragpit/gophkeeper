package handlers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/fragpit/gophkeeper/internal/model"
	"github.com/fragpit/gophkeeper/internal/service/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

// GetFileService retrieves encrypted items and associated file contents.
//
//go:generate mockgen -destination ./mocks/get_file_mock_gen.go . GetFileService
type GetFileService interface {
	GetItemByTitle(
		ctx context.Context,
		userID int,
		title string,
	) (*model.ItemEncrypted, error)
	GetFileByItem(
		ctx context.Context,
		userID int,
		item *model.ItemEncrypted,
	) (io.ReadCloser, error)
}

// NewGetFileHandler returns an Echo handler that downloads a file by item title.
func NewGetFileHandler(svc GetFileService) echo.HandlerFunc {
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
					&deleteResponse{Error: "item not found"},
				)
			default:
				return echo.NewHTTPError(
					http.StatusInternalServerError,
					http.StatusText(http.StatusInternalServerError),
				)
			}
		}

		r, err := svc.GetFileByItem(c.Request().Context(), uid, item)
		if err != nil {
			slog.Error("get file", slog.Any("error", err))

			switch {
			case errors.Is(err, model.ErrItemNotFound):
				return c.JSON(
					http.StatusNotFound,
					&deleteResponse{Error: "item not found"},
				)
			default:
				return echo.NewHTTPError(
					http.StatusInternalServerError,
					http.StatusText(http.StatusInternalServerError),
				)
			}
		}
		defer func() { _ = r.Close() }()

		return c.Stream(http.StatusOK, "application/octet-stream", r)
	}
}
