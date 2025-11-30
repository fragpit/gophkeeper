package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
)

//go:generate mockgen -destination ./mocks/health_mock_gen.go . HealthService
type HealthService interface {
	Check(ctx context.Context) error
}

func NewHealthHandler(
	svc HealthService,
) echo.HandlerFunc {
	return func(c echo.Context) error {
		if err := svc.Check(c.Request().Context()); err != nil {
			slog.Error("health check failed", slog.Any("error", err))
			return echo.NewHTTPError(http.StatusInternalServerError)
		}

		return c.NoContent(http.StatusOK)
	}
}
