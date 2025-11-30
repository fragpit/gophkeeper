package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"
)

var (
	ErrUnsupportedContent  = errors.New("empty or unsupported content type")
	ErrInvalidJSON         = errors.New("invalid JSON")
	ErrRequestBodyTooLarge = errors.New("request body too large")
)

// ValidateParseJSONRequest validates Content-Type and decodes a JSON request body into data.
func ValidateParseJSONRequest(
	c *echo.Context,
	data any,
) error {
	r := c.Request()
	if r.Header.Get("Content-Type") != echo.MIMEApplicationJSON {
		slog.Error(
			"empty or unsupported content type",
			slog.String("content_type", r.Header.Get("Content-Type")),
		)
		return echo.NewHTTPError(
			http.StatusUnsupportedMediaType,
			ErrUnsupportedContent.Error(),
		)
	}

	r.Body = http.MaxBytesReader(c.Response(), r.Body, 1<<20)
	defer func() { _ = r.Body.Close() }()

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(data); err != nil {
		var mberr *http.MaxBytesError
		slog.Warn("invalid JSON", slog.Any("error", err))
		if errors.As(err, &mberr) {
			return c.JSON(
				http.StatusRequestEntityTooLarge,
				&authResponse{Error: ErrRequestBodyTooLarge.Error()},
			)
		}
		return fmt.Errorf("decode json: %w", err)
	}

	if err := dec.Decode(&struct{}{}); err != io.EOF {
		slog.Warn("invalid JSON", slog.Any("error", err))
		return c.JSON(
			http.StatusBadRequest,
			&authResponse{Error: ErrInvalidJSON.Error()},
		)
	}

	return nil
}
