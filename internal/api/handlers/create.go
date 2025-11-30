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

func NewCreateHandler(svc CreateService) echo.HandlerFunc {
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
		reqData := &createRequest{}

		if err := c.Bind(reqData); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		id, err := svc.CreateItem(c.Request().Context(), uid, reqData.Item)
		if err != nil {
			slog.Error("create item", slog.Any("error", err))

			switch {
			case errors.Is(err, model.ErrUserExists):
				return c.JSON(
					http.StatusConflict,
					&createResponse{Error: http.StatusText(http.StatusConflict)},
				)
			case errors.Is(err, model.ErrPasswordPolicyViolated):
				return c.JSON(
					http.StatusBadRequest,
					&createResponse{Error: model.ErrPasswordPolicyViolated.Error()},
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
