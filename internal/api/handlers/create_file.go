package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"mime/multipart"
	"net/http"

	"github.com/fragpit/gophkeeper/internal/model"
	"github.com/fragpit/gophkeeper/internal/service/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

//go:generate mockgen -destination ./mocks/auth_mock_gen.go . AuthService
type CreateFileService interface {
	CreateFileItem(
		ctx context.Context,
		userID int,
		item *model.ItemDecrypted,
		fileContentReader multipart.File,
		contentType string,
	) (int, error)
}

type createFileResponse struct {
	ID    int    `json:"id"`
	Error string `json:"error"`
}

func NewCreateFileHandler(svc CreateFileService) echo.HandlerFunc {
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

		itemRaw := c.FormValue("item")
		if itemRaw == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "item is required")
		}

		var item model.ItemDecrypted
		if err := json.Unmarshal([]byte(itemRaw), &item); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid item format")
		}

		file, err := c.FormFile("file")
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "file is required")
		}
		f, err := file.Open()
		if err != nil {
			return echo.NewHTTPError(
				http.StatusBadRequest,
				"cannot open uploaded file",
			)
		}
		defer f.Close()

		contentType := file.Header.Get("Content-Type")
		id, err := svc.CreateFileItem(
			c.Request().Context(),
			uid,
			&item,
			f,
			contentType,
		)
		if err != nil {
			slog.Error("create file", slog.Any("error", err))
			if errors.Is(err, model.ErrAlreadyExists) {
				return echo.NewHTTPError(http.StatusConflict, "item already exists")
			}
			return echo.NewHTTPError(
				http.StatusInternalServerError,
				"failed to create file",
			)
		}

		return c.JSON(http.StatusOK, &createFileResponse{ID: id})
	}
}
