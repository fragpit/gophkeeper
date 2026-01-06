package handlers

import (
	"context"
	"io"
	"log/slog"
	"net/http"

	"github.com/fragpit/gophkeeper/internal/model"
	"github.com/fragpit/gophkeeper/internal/service/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

//go:generate mockgen -destination ./mocks/auth_mock_gen.go . AuthService
// GetFileService retrieves encrypted items and associated file contents.
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
		title := c.QueryParam("title")

		item, err := svc.GetItemByTitle(
			c.Request().Context(),
			uid,
			title,
		)
		if err != nil {
			slog.Error("get item", slog.Any("error", err))
			return echo.NewHTTPError(http.StatusInternalServerError)
		}

		r, err := svc.GetFileByItem(c.Request().Context(), uid, item)
		if err != nil {
			slog.Error("get file", slog.Any("error", err))
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
		defer r.Close()

		return c.Stream(http.StatusOK, "application/octet-stream", r)
	}
}
