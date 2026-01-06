package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
)

// ValidateParseJSONRequest validates Content-Type and decodes a JSON request body into data.
func ValidateParseJSONRequest(
	c echo.Context,
	data any,
) error {
	r := c.Request()
	if r.Header.Get("Content-Type") != echo.MIMEApplicationJSON {
		slog.Error(
			"request with an empty or unsupported content type",
			slog.String("content_type", r.Header.Get("Content-Type")),
		)
		return errors.New("request with an empty or unsupported content type")
	}

	r.Body = http.MaxBytesReader(c.Response(), r.Body, 1<<20)
	defer func() { _ = r.Body.Close() }()

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(data); err != nil {
		var mberr *http.MaxBytesError
		slog.Warn("invalid JSON", slog.Any("error", err))
		if errors.As(err, &mberr) {
			return errors.New("request body too large")
		}
		return errors.New("invalid JSON")
	}

	if err := dec.Decode(&struct{}{}); err != io.EOF {
		slog.Warn("invalid JSON", slog.Any("error", err))
		return errors.New("invalid JSON")
	}

	return nil
}
